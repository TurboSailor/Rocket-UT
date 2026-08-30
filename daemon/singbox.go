package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type jmap = map[string]any

// nodeOutbound строит outbound sing-box для узла.
// Узел awg не имеет своего протокола: он маршрутизируется через awg0 по fwmark.
func nodeOutbound(n Node, tag string) (jmap, error) {
	o := jmap{"tag": tag, "server": n.Server, "server_port": n.Port}
	switch n.Type {
	case "vless":
		o["type"] = "vless"
		o["uuid"] = n.UUID
		if n.Flow != "" {
			o["flow"] = n.Flow
		}
	case "vmess":
		o["type"] = "vmess"
		o["uuid"] = n.UUID
		sec := n.Method
		switch sec {
		case "", "none", "auto":
			sec = "auto"
		}
		o["security"] = sec
		o["alter_id"] = n.AlterID
	case "ss":
		o["type"] = "shadowsocks"
		o["method"] = n.Method
		o["password"] = n.Password
	case "trojan":
		o["type"] = "trojan"
		o["password"] = n.Password
	case "socks5":
		o["type"] = "socks"
		o["version"] = "5"
		if n.User != "" {
			o["username"] = n.User
			o["password"] = n.Password
		}
	case "http":
		o["type"] = "http"
		if n.User != "" {
			o["username"] = n.User
			o["password"] = n.Password
		}
	case "ssh":
		o["type"] = "ssh"
		o["user"] = n.User
		if n.Password != "" {
			o["password"] = n.Password
		}
		if n.PrivKey != "" {
			o["private_key"] = n.PrivKey
		}
	case "awg":
		// Прямой выход через awg0. Нужны ОБА поля:
		// routing_mark уводит сокет в таблицу AwgTable (там default dev awg0),
		// а bind_interface обязателен, потому что route.auto_detect_interface
		// иначе привязывает сокет к wlan0 через SO_BINDTODEVICE и марка
		// перестаёт влиять на выбор маршрута (проверено на устройстве).
		return jmap{
			"type":           "direct",
			"tag":            tag,
			"routing_mark":   AwgMark,
			"bind_interface": AwgName,
		}, nil
	default:
		return nil, fmt.Errorf("node type %s not supported", n.Type)
	}

	if n.TLS && n.Type != "ssh" {
		tls := jmap{"enabled": true}
		sni := n.SNI
		if sni == "" {
			sni = n.Host
		}
		if sni == "" && net.ParseIP(n.Server) == nil {
			sni = n.Server
		}
		if sni != "" {
			tls["server_name"] = sni
		}
		if n.Insecure {
			tls["insecure"] = true
		}
		if n.ALPN != "" {
			tls["alpn"] = splitCSV(n.ALPN)
		}
		if n.FP != "" {
			tls["utls"] = jmap{"enabled": true, "fingerprint": n.FP}
		}
		if n.Reality.PublicKey != "" {
			r := jmap{"enabled": true, "public_key": n.Reality.PublicKey}
			if n.Reality.ShortID != "" {
				r["short_id"] = n.Reality.ShortID
			}
			tls["reality"] = r
			// REALITY требует uTLS.
			if n.FP == "" {
				tls["utls"] = jmap{"enabled": true, "fingerprint": "chrome"}
			}
		}
		o["tls"] = tls
	}

	switch n.Network {
	case "ws":
		t := jmap{"type": "ws"}
		if n.Path != "" {
			t["path"] = n.Path
		}
		if n.Host != "" {
			t["headers"] = jmap{"Host": n.Host}
		}
		o["transport"] = t
	case "grpc":
		t := jmap{"type": "grpc"}
		if n.Path != "" {
			t["service_name"] = strings.TrimPrefix(n.Path, "/")
		}
		o["transport"] = t
	case "http":
		t := jmap{"type": "http"}
		if n.Path != "" {
			t["path"] = n.Path
		}
		if n.Host != "" {
			t["host"] = []string{n.Host}
		}
		o["transport"] = t
	case "httpupgrade":
		t := jmap{"type": "httpupgrade"}
		if n.Path != "" {
			t["path"] = n.Path
		}
		if n.Host != "" {
			t["host"] = n.Host
		}
		o["transport"] = t
	}
	return o, nil
}

// policyTag переводит политику Shadowrocket в тег outbound sing-box.
// Второе значение — true, если политика означает reject (action, а не outbound).
func policyTag(policy string, groups map[string]bool, proxyTag string) (string, bool) {
	p := strings.ToUpper(strings.TrimSpace(policy))
	switch p {
	case "DIRECT":
		return tagDirect, false
	case "REJECT", "REJECT-TINYGIF", "REJECT-DICT", "REJECT-200", "REJECT-IMG":
		return "", true
	case "PROXY", "":
		return proxyTag, false
	}
	if groups[policy] {
		return "group-" + policyTagSafe(policy), false
	}
	// Неизвестная политика трактуется как PROXY — так же ведёт себя Shadowrocket.
	return proxyTag, false
}

func policyTagSafe(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		}
		return '-'
	}, s)
}

// cleanArg убирает мусор, реально встречающийся в публичных списках правил:
// zero-width space/joiner, BOM и типографские тире вместо дефиса.
func cleanArg(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\ufeff':
			return -1
		case '\u2010', '\u2011', '\u2012', '\u2013', '\u2014', '\u2015':
			return '-'
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

// parsePortRange разбирает "596-599" в пару портов.
func parsePortRange(s string) (int, int, bool) {
	i := strings.IndexByte(s, '-')
	if i <= 0 || i == len(s)-1 {
		return 0, 0, false
	}
	lo, err1 := strconv.Atoi(strings.TrimSpace(s[:i]))
	hi, err2 := strconv.Atoi(strings.TrimSpace(s[i+1:]))
	if err1 != nil || err2 != nil || lo <= 0 || hi <= 0 ||
		lo > 65535 || hi > 65535 || lo > hi {
		return 0, 0, false
	}
	return lo, hi, true
}

// ruleToSB транслирует одно правило Shadowrocket в route-правило sing-box.
// Возвращает ошибку, если правило не выражается напрямую.
func ruleToSB(r Rule) (jmap, error) {
	switch r.Type {
	case "DOMAIN":
		d := punyDomain(cleanArg(r.Arg))
		if d == "" {
			return nil, fmt.Errorf("empty domain")
		}
		return jmap{"domain": []string{d}}, nil
	case "DOMAIN-SUFFIX":
		d := punyDomain(cleanArg(r.Arg))
		if d == "" {
			return nil, fmt.Errorf("empty domain suffix")
		}
		return jmap{"domain_suffix": []string{d}}, nil
	case "DOMAIN-KEYWORD":
		d := punyDomain(cleanArg(r.Arg))
		if d == "" {
			return nil, fmt.Errorf("empty keyword")
		}
		return jmap{"domain_keyword": []string{d}}, nil
	case "IP-CIDR", "IP-CIDR6", "IP6-CIDR":
		cidr := r.Arg
		if !strings.Contains(cidr, "/") {
			if ip := net.ParseIP(cidr); ip != nil {
				if ip.To4() != nil {
					cidr += "/32"
				} else {
					cidr += "/128"
				}
			}
		}
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return nil, fmt.Errorf("bad cidr %q", r.Arg)
		}
		return jmap{"ip_cidr": []string{cidr}}, nil
	case "SRC-IP-CIDR":
		cidr := r.Arg
		if !strings.Contains(cidr, "/") {
			cidr += "/32"
		}
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return nil, fmt.Errorf("bad cidr %q", r.Arg)
		}
		return jmap{"source_ip_cidr": []string{cidr}}, nil
	case "DST-PORT":
		// Реальные списки содержат диапазоны и мусор: en-dash, zero-width space.
		arg := cleanArg(r.Arg)
		if lo, hi, ok := parsePortRange(arg); ok {
			return jmap{"port_range": []string{strconv.Itoa(lo) + ":" + strconv.Itoa(hi)}}, nil
		}
		p, err := strconv.Atoi(arg)
		if err != nil || p <= 0 || p > 65535 {
			return nil, fmt.Errorf("bad port %q", r.Arg)
		}
		return jmap{"port": []int{p}}, nil
	case "GEOIP":
		cc := strings.ToLower(r.Arg)
		if cc == "" {
			return nil, fmt.Errorf("geoip: empty country")
		}
		return jmap{"rule_set": []string{"geoip-" + cc}}, nil
	}
	return nil, fmt.Errorf("rule type %s not supported", r.Type)
}

func geoipRuleSet(cc string) jmap {
	return jmap{
		"type":   "remote",
		"tag":    "geoip-" + cc,
		"format": "binary",
		"url":    "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-" + cc + ".srs",
	}
}

// skipProxyEntry превращает элемент skip-proxy в route-правило direct.
func skipProxyEntry(e string) jmap {
	e = strings.TrimSpace(e)
	if e == "" {
		return nil
	}
	if strings.Contains(e, "/") {
		if _, _, err := net.ParseCIDR(e); err == nil {
			return jmap{"ip_cidr": []string{e}}
		}
		return nil
	}
	if ip := net.ParseIP(e); ip != nil {
		suffix := "/32"
		if ip.To4() == nil {
			suffix = "/128"
		}
		return jmap{"ip_cidr": []string{e + suffix}}
	}
	if strings.HasPrefix(e, "*.") {
		return jmap{"domain_suffix": []string{punyDomain(strings.TrimPrefix(e, "*"))}}
	}
	return jmap{"domain": []string{punyDomain(e)}}
}

// dnsServer строит DNS-сервер sing-box из записи Shadowrocket dns-server.
func dnsServer(spec, tag string) jmap {
	spec = strings.TrimSpace(spec)
	switch {
	case spec == "" || strings.EqualFold(spec, "system"):
		return nil
	case strings.HasPrefix(spec, "https://"):
		u := strings.TrimPrefix(spec, "https://")
		host := u
		path := "/dns-query"
		if i := strings.Index(u, "/"); i >= 0 {
			host, path = u[:i], u[i:]
		}
		return jmap{"type": "https", "tag": tag, "server": host, "path": path}
	case strings.HasPrefix(spec, "tls://"):
		h := strings.TrimPrefix(spec, "tls://")
		port := 853
		if host, p, err := hostPort(h); err == nil {
			h, port = host, p
		}
		return jmap{"type": "tls", "tag": tag, "server": h, "server_port": port}
	case strings.HasPrefix(spec, "quic://"):
		return jmap{"type": "quic", "tag": tag, "server": strings.TrimPrefix(spec, "quic://")}
	case strings.HasPrefix(spec, "h3://"):
		return jmap{"type": "h3", "tag": tag, "server": strings.TrimPrefix(spec, "h3://")}
	}
	// Голый адрес — UDP. Порт может быть указан как host:port.
	if strings.Count(spec, ":") == 1 {
		if host, p, err := hostPort(spec); err == nil {
			return jmap{"type": "udp", "tag": tag, "server": host, "server_port": p}
		}
	}
	if net.ParseIP(spec) == nil {
		return nil
	}
	return jmap{"type": "udp", "tag": tag, "server": spec}
}

type sbBuild struct {
	Config  jmap
	Skipped []string
}

// buildSingbox формирует конфиг sing-box из состояния, активного узла и .conf.
// mode: config — правила из .conf; proxy — всё в узел; direct — всё напрямую.
func buildSingbox(mode string, active *Node, conf *Conf, rs *rulesetCache) (*sbBuild, error) {
	b := &sbBuild{}
	ipv6 := false
	if conf != nil && truthy(conf.General["ipv6"]) {
		ipv6 = true
	}

	// --- outbounds ---
	outs := []jmap{{"type": "direct", "tag": tagDirect}}
	proxyTag := tagDirect
	if active != nil {
		tag := tagProxy
		if active.Type == "awg" {
			tag = tagAwg
		}
		o, err := nodeOutbound(*active, tag)
		if err != nil {
			return nil, err
		}
		outs = append(outs, o)
		proxyTag = tag
	}

	// Группы политик: select → selector, остальные → urltest.
	groupSet := map[string]bool{}
	if conf != nil && active != nil {
		for _, g := range conf.Groups {
			groupSet[g.Name] = true
		}
		for _, g := range conf.Groups {
			// Члены группы — узлы из [Proxy] этого же .conf, сопоставляем по имени.
			var members []string
			for _, m := range g.Members {
				if m == "DIRECT" {
					members = append(members, tagDirect)
					continue
				}
				if m == "PROXY" {
					members = append(members, proxyTag)
					continue
				}
				if groupSet[m] {
					members = append(members, "group-"+policyTagSafe(m))
				}
			}
			if len(members) == 0 {
				members = []string{proxyTag}
			}
			gt := "selector"
			if g.Type != "select" {
				gt = "urltest"
			}
			go_ := jmap{"type": gt, "tag": "group-" + policyTagSafe(g.Name), "outbounds": members}
			if gt == "urltest" {
				if g.URL != "" {
					go_["url"] = g.URL
				}
				go_["interval"] = strconv.Itoa(g.Interval) + "s"
			}
			if g.Type == "fallback" || g.Type == "load-balance" {
				b.Skipped = append(b.Skipped,
					fmt.Sprintf("group %q: %s mapped to url-test", g.Name, g.Type))
			}
			outs = append(outs, go_)
		}
	}

	// --- DNS ---
	var dnsServers []jmap
	dnsFinal := ""
	if conf != nil {
		specs := splitCSV(conf.General["dns-server"])
		specs = append(specs, splitCSV(conf.General["fallback-dns-server"])...)
		for i, sp := range specs {
			s := dnsServer(sp, fmt.Sprintf("dns-%d", i))
			if s == nil {
				continue
			}
			if !ipv6 && net.ParseIP(fmt.Sprint(s["server"])) != nil &&
				net.ParseIP(fmt.Sprint(s["server"])).To4() == nil {
				continue // IPv6-адрес сервера при выключенном IPv6
			}
			dnsServers = append(dnsServers, s)
			if dnsFinal == "" {
				dnsFinal = fmt.Sprint(s["tag"])
			}
			if len(dnsServers) >= 4 {
				break
			}
		}
	}
	if len(dnsServers) == 0 {
		dnsServers = []jmap{{"type": "udp", "tag": "dns-remote", "server": "1.1.1.1"}}
		dnsFinal = "dns-remote"
	}
	// DoH/DoT, заданные именем хоста, требуют domain_resolver — иначе
	// sing-box 1.13 отказывается стартовать ("missing domain resolver").
	// Ищем сервер с литеральным IP как bootstrap, при отсутствии — добавляем.
	bootstrap := ""
	for _, s := range dnsServers {
		if srv, ok := s["server"].(string); ok && net.ParseIP(srv) != nil {
			bootstrap = fmt.Sprint(s["tag"])
			break
		}
	}
	if bootstrap == "" {
		bootstrap = "dns-bootstrap"
		dnsServers = append([]jmap{
			{"type": "udp", "tag": bootstrap, "server": "1.1.1.1"},
		}, dnsServers...)
	}
	for _, s := range dnsServers {
		srv, ok := s["server"].(string)
		if !ok || srv == "" || net.ParseIP(srv) != nil {
			continue
		}
		if fmt.Sprint(s["tag"]) == bootstrap {
			continue
		}
		s["domain_resolver"] = bootstrap
	}
	dns := jmap{"servers": dnsServers, "final": dnsFinal}
	if !ipv6 {
		dns["strategy"] = "ipv4_only"
	}
	// [Host] → DNS-сервер типа hosts (проверено на sing-box 1.13.21).
	if conf != nil && len(conf.Hosts) > 0 {
		pre := jmap{}
		var domains []string
		for h, ip := range conf.Hosts {
			if net.ParseIP(ip) == nil {
				b.Skipped = append(b.Skipped, fmt.Sprintf("host %q: %q is not an IP", h, ip))
				continue
			}
			h = punyDomain(h)
			pre[h] = []string{ip}
			domains = append(domains, h)
		}
		if len(domains) > 0 {
			dns["servers"] = append([]jmap{{
				"type": "hosts", "tag": "dns-hosts", "predefined": pre,
			}}, dnsServers...)
			dns["rules"] = []jmap{{"domain": domains, "server": "dns-hosts"}}
		}
	}

	// --- TUN inbound ---
	// sniff/sniff_override_destination — legacy-поля, удалены в sing-box 1.13:
	// вместо них route-правила с action sniff/hijack-dns.
	tun := jmap{
		"type":                 "tun",
		"tag":                  "tun-in",
		"interface_name":       TunName,
		"address":              []string{TunAddr4},
		"mtu":                  TunMTU,
		"stack":                "gvisor",
		"auto_route":           true,
		"auto_redirect":        false,
		"strict_route":         false,
		"iproute2_table_index": TunTable,
		"iproute2_rule_index":  TunRuleIndex,
	}
	if conf != nil {
		var excl []string
		for _, k := range []string{"bypass-tun", "tun-excluded-routes"} {
			for _, e := range splitCSV(conf.General[k]) {
				if _, _, err := net.ParseCIDR(e); err == nil {
					excl = append(excl, e)
				}
			}
		}
		if len(excl) > 0 {
			tun["route_exclude_address"] = excl
		}
	}

	// --- route rules ---
	rules := []jmap{
		{"action": "sniff"},
		{"protocol": "dns", "action": "hijack-dns"},
	}
	geo := map[string]bool{}
	final := tagDirect

	switch mode {
	case "direct":
		final = tagDirect
	case "proxy":
		final = proxyTag
	default: // config
		if conf != nil {
			// skip-proxy — всегда напрямую, до остальных правил.
			for _, e := range splitCSV(conf.General["skip-proxy"]) {
				if r := skipProxyEntry(e); r != nil {
					r["outbound"] = tagDirect
					rules = append(rules, r)
				}
			}
			for _, r := range conf.Rules {
				if r.Type == "FINAL" {
					continue
				}
				expanded := []Rule{r}
				if r.Type == "RULE-SET" {
					sub, err := rs.expand(r)
					if err != nil {
						b.Skipped = append(b.Skipped, fmt.Sprintf("rule-set %s: %v", r.Arg, err))
						continue
					}
					expanded = sub
				}
				for _, er := range expanded {
					sr, err := ruleToSB(er)
					if err != nil {
						b.Skipped = append(b.Skipped, err.Error())
						continue
					}
					if er.Type == "GEOIP" {
						geo[strings.ToLower(er.Arg)] = true
					}
					tag, reject := policyTag(er.Policy, groupSet, proxyTag)
					if reject {
						sr["action"] = "reject"
					} else {
						sr["outbound"] = tag
					}
					rules = append(rules, sr)
				}
			}
			ft, reject := policyTag(conf.Final(), groupSet, proxyTag)
			if reject {
				// FINAL,REJECT как outbound недоступен: добавляем правило-заглушку.
				rules = append(rules, jmap{"ip_cidr": []string{"0.0.0.0/0", "::/0"}, "action": "reject"})
				final = tagDirect
			} else {
				final = ft
			}
		}
	}

	rules = mergeRules(rules)

	route := jmap{
		"auto_detect_interface":   true,
		"default_domain_resolver": jmap{"server": dnsFinal},
		"rules":                   rules,
		"final":                   final,
	}
	if len(geo) > 0 {
		var sets []jmap
		for cc := range geo {
			sets = append(sets, geoipRuleSet(cc))
		}
		route["rule_set"] = sets
	}

	cfg := jmap{
		"log": jmap{"level": "info", "timestamp": true},
		"experimental": jmap{
			"clash_api":  jmap{"external_controller": ClashAddr},
			"cache_file": jmap{"enabled": true, "path": StateDir + "/cache.db"},
		},
		"dns":       dns,
		"inbounds":  []jmap{tun},
		"outbounds": outs,
		"route":     route,
	}
	b.Config = cfg
	return b, nil
}

// mergeRules склеивает соседние однотипные правила с одинаковым исходом.
// Без этого RULE-SET на 10k строк даёт 10k правил и sing-box стартует минуты.
func mergeRules(in []jmap) []jmap {
	keys := []string{"domain", "domain_suffix", "domain_keyword", "ip_cidr", "source_ip_cidr"}
	out := make([]jmap, 0, len(in))
	for _, r := range in {
		merged := false
		if len(out) > 0 {
			prev := out[len(out)-1]
			for _, k := range keys {
				pv, pok := prev[k].([]string)
				cv, cok := r[k].([]string)
				if !pok || !cok || len(prev) != 2 || len(r) != 2 {
					continue
				}
				if fmt.Sprint(prev["outbound"]) != fmt.Sprint(r["outbound"]) ||
					fmt.Sprint(prev["action"]) != fmt.Sprint(r["action"]) {
					continue
				}
				prev[k] = append(pv, cv...)
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, r)
		}
	}
	return out
}

func marshalConfig(cfg jmap) (string, error) {
	b, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
