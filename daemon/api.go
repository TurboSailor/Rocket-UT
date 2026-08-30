package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// API — HTTP-интерфейс демона, только на 127.0.0.1.
type API struct {
	st *Store
	en *Engine
}

func reply(w http.ResponseWriter, code int, obj any) {
	b, err := json.Marshal(obj)
	if err != nil {
		b, code = []byte(`{"error":"marshal failed"}`), 500
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(code)
	_, _ = w.Write(b)
}

func fail(w http.ResponseWriter, code int, format string, a ...any) {
	reply(w, code, map[string]string{"error": fmt.Sprintf(format, a...)})
}

func body(r *http.Request) (string, error) {
	b, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// status собирает полный снимок для UI.
func (a *API) status() map[string]any {
	st := a.st.State()
	rx, tx := a.en.log.Traffic()
	out := map[string]any{
		"up": st.Up, "mode": st.Mode, "node_id": st.NodeID,
		"conf_name": st.ConfName, "ts": time.Now().Unix(),
		"rx": rx, "tx": tx,
		"confs": confNames(), "skipped": a.en.Skipped(),
		"nodes_count": len(a.st.Nodes()),
	}
	if st.Err != "" {
		out["error"] = st.Err
	}
	if n, ok := a.st.Node(st.NodeID); ok {
		out["node"] = n
	}
	if st.Up {
		out["singbox"] = a.en.sb.Running()
		if aw := a.en.awgStatus(); aw != nil {
			for k, v := range aw {
				out[k] = v
			}
		}
	}
	return out
}

func (a *API) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		reply(w, 200, a.status())
	})

	mux.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		if err := a.en.Up(); err != nil {
			a.st.SetState(func(s *State) { s.Err = err.Error() })
			st := a.status()
			st["error"] = err.Error()
			reply(w, 500, st)
			return
		}
		reply(w, 200, a.status())
	})

	mux.HandleFunc("/down", func(w http.ResponseWriter, r *http.Request) {
		a.en.Down()
		reply(w, 200, a.status())
	})

	mux.HandleFunc("/mode", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			v := r.URL.Query().Get("v")
			switch v {
			case "config", "proxy", "direct":
			default:
				fail(w, 400, "mode must be config|proxy|direct")
				return
			}
			a.st.SetState(func(s *State) { s.Mode = v })
			if err := a.en.Restack(); err != nil {
				st := a.status()
				st["error"] = err.Error()
				reply(w, 500, st)
				return
			}
		}
		reply(w, 200, a.status())
	})

	mux.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		reply(w, 200, map[string]any{"nodes": a.st.Nodes()})
	})

	mux.HandleFunc("/nodes/import", func(w http.ResponseWriter, r *http.Request) {
		text, err := body(r)
		if err != nil {
			fail(w, 400, "read body: %v", err)
			return
		}
		// sub:// — это подписка, а не узел. Однозначно, поэтому добавляем сразу:
		// иначе вставка такой ссылки из Shadowrocket падала с «not a uri».
		if link, ok := parseSubLink(text); ok {
			name := strings.TrimSpace(r.URL.Query().Get("subname"))
			if name == "" {
				name = "sub"
			}
			sub := Sub{ID: id12(link), Name: name, URL: link}
			a.st.AddSub(sub)
			cnt, ferr := fetchSub(a.st, sub)
			out := a.status()
			out["subs"] = a.st.Subs()
			out["nodes"] = a.st.Nodes()
			out["added"] = cnt
			out["kind"] = "subscription"
			out["url"] = link
			if ferr != nil {
				out["error"] = ferr.Error()
			}
			reply(w, 200, out)
			return
		}
		nodes, awgName, errs := importText(r.URL.Query().Get("name"), text)
		if len(nodes) == 0 {
			msg := "nothing importable"
			if len(errs) > 0 {
				msg = strings.Join(errs, "; ")
			}
			fail(w, 400, "%s", msg)
			return
		}
		added := a.st.AddNodes(nodes)
		out := a.status()
		out["added"] = added
		out["nodes"] = a.st.Nodes()
		if awgName != "" {
			out["awg_conf"] = awgName
		}
		if len(errs) > 0 {
			out["warnings"] = errs
		}
		reply(w, 200, out)
	})

	mux.HandleFunc("/nodes/update", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		old, ok := a.st.Node(id)
		if !ok {
			fail(w, 404, "no such node")
			return
		}
		raw, err := body(r)
		if err != nil {
			fail(w, 400, "read body: %v", err)
			return
		}
		// Правим копию: неуказанные поля сохраняют прежние значения.
		edited := old
		if err := json.Unmarshal([]byte(raw), &edited); err != nil {
			fail(w, 400, "bad json: %v", err)
			return
		}
		edited.Type = strings.ToLower(strings.TrimSpace(edited.Type))
		edited.Server = strings.TrimSpace(edited.Server)
		edited.Name = strings.TrimSpace(edited.Name)
		if edited.Name == "" {
			edited.Name = old.Name
		}
		// Имя, которое приложение сгенерировало само (`тип-адрес`), следует за
		// адресом; имя, заданное пользователем, не трогаем.
		if edited.Name == old.Type+"-"+old.Server && edited.Server != old.Server {
			edited.Name = edited.Type + "-" + edited.Server
		}
		if edited.Type == "awg" {
			// Конфиг AmneziaWG правится как файл, а не полями узла.
			edited.AWGConf = old.AWGConf
		} else if edited.Server == "" || edited.Port <= 0 || edited.Port > 65535 {
			fail(w, 400, "server and port are required")
			return
		}
		// Проверяем, что узел вообще транслируется в outbound.
		if _, err := nodeOutbound(edited, tagProxy); err != nil {
			fail(w, 400, "%v", err)
			return
		}
		newID, ok := a.st.ReplaceNode(id, edited)
		if !ok {
			fail(w, 404, "no such node")
			return
		}
		// Активный узел переезжает на новый id.
		if a.st.State().NodeID == id {
			a.st.SetState(func(s *State) { s.NodeID = newID; s.Err = "" })
			if err := a.en.Restack(); err != nil {
				st := a.status()
				st["error"] = err.Error()
				st["nodes"] = a.st.Nodes()
				reply(w, 500, st)
				return
			}
		}
		out := a.status()
		out["nodes"] = a.st.Nodes()
		out["id"] = newID
		reply(w, 200, out)
	})

	mux.HandleFunc("/nodes/select", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if _, ok := a.st.Node(id); !ok {
			fail(w, 404, "no such node")
			return
		}
		a.st.SetState(func(s *State) { s.NodeID = id; s.Err = "" })
		if err := a.en.Restack(); err != nil {
			st := a.status()
			st["error"] = err.Error()
			reply(w, 500, st)
			return
		}
		reply(w, 200, a.status())
	})

	mux.HandleFunc("/nodes/delete", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if !a.st.DeleteNode(id) {
			fail(w, 404, "no such node")
			return
		}
		if a.st.State().NodeID == id {
			a.st.SetState(func(s *State) { s.NodeID = "" })
			a.en.Down()
		}
		out := a.status()
		out["nodes"] = a.st.Nodes()
		reply(w, 200, out)
	})

	mux.HandleFunc("/nodes/test", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		targets := a.st.Nodes()
		if id != "" {
			n, ok := a.st.Node(id)
			if !ok {
				fail(w, 404, "no such node")
				return
			}
			targets = []Node{n}
		}
		for _, n := range targets {
			a.st.SetLatency(n.ID, tcpLatency(n))
		}
		reply(w, 200, map[string]any{"nodes": a.st.Nodes()})
	})

	mux.HandleFunc("/subs", func(w http.ResponseWriter, r *http.Request) {
		reply(w, 200, map[string]any{"subs": a.st.Subs()})
	})

	mux.HandleFunc("/subs/add", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		name, raw := strings.TrimSpace(q.Get("name")), strings.TrimSpace(q.Get("url"))
		if name == "" {
			name = "sub"
		}
		// Ссылку вида sub://<base64> раскрываем в реальный адрес подписки.
		link := raw
		if resolved, ok := parseSubLink(raw); ok {
			link = resolved
		}
		if !isHTTPURL(link) {
			fail(w, 400, "url must be http(s) or sub://<base64>")
			return
		}
		sub := Sub{ID: id12(link), Name: name, URL: link}
		a.st.AddSub(sub)
		n, err := fetchSub(a.st, sub)
		out := map[string]any{"subs": a.st.Subs(), "nodes": a.st.Nodes(), "count": n}
		if err != nil {
			out["error"] = err.Error()
			reply(w, 200, out) // подписка сохранена, ошибка показана в UI
			return
		}
		reply(w, 200, out)
	})

	mux.HandleFunc("/subs/update", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		subs := a.st.Subs()
		if id != "" {
			found := false
			for _, s := range subs {
				if s.ID == id {
					subs, found = []Sub{s}, true
					break
				}
			}
			if !found {
				fail(w, 404, "no such subscription")
				return
			}
		}
		total, var_errs := 0, []string{}
		for _, s := range subs {
			n, err := fetchSub(a.st, s)
			if err != nil {
				var_errs = append(var_errs, s.Name+": "+err.Error())
				continue
			}
			total += n
		}
		// Активный узел мог исчезнуть после обновления.
		stt := a.st.State()
		if stt.NodeID != "" {
			if _, ok := a.st.Node(stt.NodeID); !ok {
				a.en.Down()
				a.st.SetState(func(s *State) {
					s.NodeID = ""
					s.Err = "active node removed by subscription update"
				})
			}
		}
		out := map[string]any{"subs": a.st.Subs(), "nodes": a.st.Nodes(), "count": total}
		if len(var_errs) > 0 {
			out["warnings"] = var_errs
		}
		reply(w, 200, out)
	})

	mux.HandleFunc("/subs/delete", func(w http.ResponseWriter, r *http.Request) {
		if !a.st.DeleteSub(r.URL.Query().Get("id")) {
			fail(w, 404, "no such subscription")
			return
		}
		reply(w, 200, map[string]any{"subs": a.st.Subs(), "nodes": a.st.Nodes()})
	})

	mux.HandleFunc("/confs", func(w http.ResponseWriter, r *http.Request) {
		reply(w, 200, map[string]any{"confs": confNames(), "active": a.st.State().ConfName})
	})

	mux.HandleFunc("/conf", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = a.st.State().ConfName
		}
		if !validName(name) {
			fail(w, 400, "bad name")
			return
		}
		switch r.Method {
		case "GET":
			text, err := readConf(name)
			if err != nil {
				fail(w, 404, "no such config")
				return
			}
			c, perr := ParseConf(text)
			out := map[string]any{"name": name, "text": text}
			if perr == nil {
				out["skipped"] = c.Skipped
				out["rules"] = len(c.Rules)
				out["proxies"] = len(c.Proxies)
			}
			reply(w, 200, out)
		case "POST":
			text, err := body(r)
			if err != nil {
				fail(w, 400, "read body: %v", err)
				return
			}
			c, perr := ParseConf(text)
			if perr != nil {
				fail(w, 400, "%v", perr)
				return
			}
			if err := writeSecret(confPath(name), text); err != nil {
				fail(w, 500, "save: %v", err)
				return
			}
			if a.st.State().ConfName == name {
				if err := a.en.Reload(); err != nil {
					logf("reload after conf save: %v", err)
				}
			}
			reply(w, 200, map[string]any{"name": name, "skipped": c.Skipped,
				"rules": len(c.Rules), "proxies": len(c.Proxies)})
		default:
			fail(w, 405, "method not allowed")
		}
	})

	mux.HandleFunc("/conf/select", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name != "" {
			if !validName(name) {
				fail(w, 400, "bad name")
				return
			}
			if _, err := readConf(name); err != nil {
				fail(w, 404, "no such config")
				return
			}
		}
		a.st.SetState(func(s *State) { s.ConfName = name })
		if err := a.en.Restack(); err != nil {
			st := a.status()
			st["error"] = err.Error()
			reply(w, 500, st)
			return
		}
		reply(w, 200, a.status())
	})

	mux.HandleFunc("/conf/delete", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if !validName(name) {
			fail(w, 400, "bad name")
			return
		}
		if err := os.Remove(confPath(name)); err != nil {
			fail(w, 404, "no such config")
			return
		}
		if a.st.State().ConfName == name {
			a.st.SetState(func(s *State) { s.ConfName = "" })
			if err := a.en.Reload(); err != nil {
				logf("reload after conf delete: %v", err)
			}
		}
		reply(w, 200, map[string]any{"confs": confNames(), "active": a.st.State().ConfName})
	})

	mux.HandleFunc("/conf/fromurl", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		name, url := q.Get("name"), q.Get("url")
		if !validName(name) {
			fail(w, 400, "bad name")
			return
		}
		text, err := httpGetText(url, maxSubBody)
		if err != nil {
			fail(w, 400, "fetch: %v", err)
			return
		}
		c, perr := ParseConf(text)
		if perr != nil {
			fail(w, 400, "%v", perr)
			return
		}
		if err := writeSecret(confPath(name), text); err != nil {
			fail(w, 500, "save: %v", err)
			return
		}
		added := 0
		if len(c.Proxies) > 0 {
			added = a.st.AddNodes(c.Proxies)
		}
		reply(w, 200, map[string]any{"name": name, "skipped": c.Skipped,
			"rules": len(c.Rules), "proxies": len(c.Proxies), "added": added,
			"confs": confNames()})
	})

	mux.HandleFunc("/rules", func(w http.ResponseWriter, r *http.Request) {
		name, c, err := a.activeConfForEdit()
		if err != nil {
			fail(w, 500, "%v", err)
			return
		}
		reply(w, 200, map[string]any{
			"conf": name, "rules": c.Rules, "skipped": c.Skipped,
		})
	})

	mux.HandleFunc("/rules/add", func(w http.ResponseWriter, r *http.Request) {
		raw, err := body(r)
		if err != nil {
			fail(w, 400, "read body: %v", err)
			return
		}
		rule, err := parseRuleInput(raw)
		if err != nil {
			fail(w, 400, "%v", err)
			return
		}
		a.editRules(w, func(c *Conf) error {
			c.InsertRule(rule)
			return nil
		})
	})

	mux.HandleFunc("/rules/update", func(w http.ResponseWriter, r *http.Request) {
		line, err := strconv.Atoi(r.URL.Query().Get("line"))
		if err != nil {
			fail(w, 400, "bad line")
			return
		}
		raw, err := body(r)
		if err != nil {
			fail(w, 400, "read body: %v", err)
			return
		}
		rule, err := parseRuleInput(raw)
		if err != nil {
			fail(w, 400, "%v", err)
			return
		}
		a.editRules(w, func(c *Conf) error {
			if !c.UpdateRuleLine(line, rule) {
				return fmt.Errorf("no rule at line %d", line)
			}
			return nil
		})
	})

	mux.HandleFunc("/rules/delete", func(w http.ResponseWriter, r *http.Request) {
		line, err := strconv.Atoi(r.URL.Query().Get("line"))
		if err != nil {
			fail(w, 400, "bad line")
			return
		}
		a.editRules(w, func(c *Conf) error {
			if !c.DeleteRuleLine(line) {
				return fmt.Errorf("no rule at line %d", line)
			}
			return nil
		})
	})

	mux.HandleFunc("/rules/move", func(w http.ResponseWriter, r *http.Request) {
		line, err := strconv.Atoi(r.URL.Query().Get("line"))
		if err != nil {
			fail(w, 400, "bad line")
			return
		}
		dir := r.URL.Query().Get("dir")
		if dir != "up" && dir != "down" {
			fail(w, 400, "dir must be up|down")
			return
		}
		a.editRules(w, func(c *Conf) error {
			if _, ok := c.MoveRuleLine(line, dir == "up"); !ok {
				return fmt.Errorf("cannot move rule at line %d", line)
			}
			return nil
		})
	})

	mux.HandleFunc("/qrdecode", func(w http.ResponseWriter, r *http.Request) {
		p, err := body(r)
		if err != nil {
			fail(w, 400, "read body: %v", err)
			return
		}
		// Кадр камеры лежит в /tmp и разрешён отдельным исключением.
		img, err := safeFileExtra(strings.TrimSpace(p), imgExt, QRFrame)
		if err != nil {
			fail(w, 400, "%v", err)
			return
		}
		text, err := decodeQR(img)
		if err != nil {
			fail(w, 422, "%v", err)
			return
		}
		reply(w, 200, map[string]any{"text": text})
	})

	mux.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		var since int64
		if v := r.URL.Query().Get("since"); v != "" {
			since, _ = strconv.ParseInt(v, 10, 64)
		}
		reply(w, 200, map[string]any{"entries": a.en.log.Since(since)})
	})

	mux.HandleFunc("/listdir", func(w http.ResponseWriter, r *http.Request) {
		dirs, files, parent, err := listDir(r.URL.Query().Get("path"))
		if err != nil {
			fail(w, 400, "%v", err)
			return
		}
		reply(w, 200, map[string]any{"dirs": dirs, "files": files, "parent": parent})
	})

	mux.HandleFunc("/readfile", func(w http.ResponseWriter, r *http.Request) {
		p, err := body(r)
		if err != nil {
			fail(w, 400, "read body: %v", err)
			return
		}
		rp, err := safeFile(strings.TrimSpace(p), confExt)
		if err != nil {
			fail(w, 400, "%v", err)
			return
		}
		f, err := os.Open(rp)
		if err != nil {
			fail(w, 400, "%v", err)
			return
		}
		defer f.Close()
		b, err := io.ReadAll(io.LimitReader(f, maxBody))
		if err != nil {
			fail(w, 400, "%v", err)
			return
		}
		reply(w, 200, map[string]any{"text": string(b)})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fail(w, 404, "not found")
	})
	return mux
}

// parseRuleInput разбирает JSON правила из UI и валидирует его.
func parseRuleInput(raw string) (Rule, error) {
	var in struct {
		Type   string `json:"type"`
		Arg    string `json:"arg"`
		Policy string `json:"policy"`
	}
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return Rule{}, fmt.Errorf("bad json: %v", err)
	}
	in.Type = strings.ToUpper(strings.TrimSpace(in.Type))
	in.Policy = strings.ToUpper(strings.TrimSpace(in.Policy))
	in.Arg = strings.TrimSpace(in.Arg)
	if in.Type == "FINAL" {
		switch in.Policy {
		case "PROXY", "DIRECT", "REJECT":
			return Rule{Type: "FINAL", Policy: in.Policy}, nil
		}
		return Rule{}, fmt.Errorf("policy must be PROXY|DIRECT|REJECT")
	}
	if in.Arg == "" || !ruleTypeOK[in.Type] || in.Type == "RULE-SET" {
		return Rule{}, fmt.Errorf("unsupported rule %s,%s", in.Type, in.Arg)
	}
	switch in.Policy {
	case "PROXY", "DIRECT", "REJECT":
	default:
		return Rule{}, fmt.Errorf("policy must be PROXY|DIRECT|REJECT")
	}
	rule := Rule{Type: in.Type, Arg: in.Arg, Policy: in.Policy}
	if _, err := ruleToSB(rule); err != nil {
		return Rule{}, err
	}
	return rule, nil
}

// activeConfForEdit отдаёт активный .conf, создавая default при его отсутствии:
// иначе правка правил из UI была бы невозможна на чистой установке.
func (a *API) activeConfForEdit() (string, *Conf, error) {
	name := a.st.State().ConfName
	if name == "" {
		name = "default"
		if _, err := readConf(name); err != nil {
			if err := writeSecret(confPath(name), defaultConfText()); err != nil {
				return "", nil, fmt.Errorf("create default conf: %w", err)
			}
		}
		a.st.SetState(func(s *State) { s.ConfName = name })
	}
	text, err := readConf(name)
	if err != nil {
		return "", nil, fmt.Errorf("read conf: %w", err)
	}
	c, err := ParseConf(text)
	if err != nil {
		return "", nil, err
	}
	return name, c, nil
}

// editRules применяет правку к активному .conf, сохраняет и перезагружает стек.
func (a *API) editRules(w http.ResponseWriter, fn func(*Conf) error) {
	name, c, err := a.activeConfForEdit()
	if err != nil {
		fail(w, 500, "%v", err)
		return
	}
	if err := fn(c); err != nil {
		fail(w, 400, "%v", err)
		return
	}
	if err := writeSecret(confPath(name), c.Text()); err != nil {
		fail(w, 500, "save: %v", err)
		return
	}
	// Перечитываем: номера строк после правки сдвигаются.
	updated, perr := ParseConf(c.Text())
	if perr != nil {
		fail(w, 500, "%v", perr)
		return
	}
	if err := a.en.Reload(); err != nil {
		logf("reload after rule edit: %v", err)
	}
	reply(w, 200, map[string]any{
		"conf": name, "rules": updated.Rules,
		"skipped": updated.Skipped, "text": c.Text(),
	})
}

// decodeQR распознаёт QR через zxing из состава пакета.
func decodeQR(img string) (string, error) {
	bin := tool("zxing")
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("zxing not found in %s", binDir())
	}
	out, err := run(60*time.Second, bin, img)
	if err != nil && out == "" {
		return "", fmt.Errorf("zxing failed: %v", err)
	}
	return parseZxingOutput(out)
}

// parseZxingOutput вытаскивает payload из вывода zxing.
// `text:` — последнее поле вывода, за ним идёт весь payload целиком, поэтому
// обрезать по первому переводу строки нельзя: конфиг AmneziaWG многострочный.
func parseZxingOutput(out string) (string, error) {
	i := strings.Index(out, "text:")
	if i < 0 {
		return "", fmt.Errorf("QR not found")
	}
	text := out[i+len("text:"):]
	text = strings.TrimLeft(text, " \t")
	text = strings.TrimRight(text, " \t\r\n")
	if text == "" {
		return "", fmt.Errorf("QR is empty")
	}
	return text, nil
}

// importText разбирает вставленный текст: URI-строки или INI AmneziaWG.
func importText(name, text string) ([]Node, string, []string) {
	var errs []string
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, "", []string{"empty input"}
	}
	// INI AmneziaWG/WireGuard.
	low := strings.ToLower(trimmed)
	if strings.Contains(low, "[interface]") && strings.Contains(low, "[peer]") {
		if name == "" {
			name = "awg"
		}
		if !validName(name) {
			return nil, "", []string{"bad awg config name"}
		}
		n, clean, err := ParseAWGConf(name, trimmed)
		if err != nil {
			return nil, "", []string{err.Error()}
		}
		if err := writeSecret(awgConfPath(name), clean); err != nil {
			return nil, "", []string{"save awg conf: " + err.Error()}
		}
		return []Node{n}, name, nil
	}
	// Shadowrocket .conf со секцией [Proxy].
	if strings.Contains(trimmed, "[Proxy]") {
		c, err := ParseConf(trimmed)
		if err == nil && len(c.Proxies) > 0 {
			return c.Proxies, "", c.Skipped
		}
	}
	nodes, bad := parseSubBody(trimmed)
	if bad > 0 {
		errs = append(errs, fmt.Sprintf("%d unparsable lines skipped", bad))
	}
	return nodes, "", errs
}

// tcpLatency измеряет TCP-хендшейк до узла; -1 при ошибке.
func tcpLatency(n Node) int {
	if n.Server == "" || n.Port == 0 {
		return -1
	}
	start := time.Now()
	c, err := net.DialTimeout("tcp",
		net.JoinHostPort(n.Server, strconv.Itoa(n.Port)), 5*time.Second)
	if err != nil {
		return -1
	}
	c.Close()
	ms := int(time.Since(start).Milliseconds())
	if ms == 0 {
		ms = 1
	}
	return ms
}
