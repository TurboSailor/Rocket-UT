package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Rule — правило маршрутизации Shadowrocket.
type Rule struct {
	Type   string   `json:"type"`
	Arg    string   `json:"arg"`
	Policy string   `json:"policy"`
	Opts   []string `json:"opts,omitempty"`
	Line   int      `json:"line"`
}

// Group — политика-группа из [Proxy Group].
type Group struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"` // select|url-test|fallback|load-balance
	Members  []string `json:"members"`
	Interval int      `json:"interval"`
	Timeout  int      `json:"timeout"`
	URL      string   `json:"url"`
}

// Conf — разобранный Shadowrocket .conf.
type Conf struct {
	General map[string]string `json:"general"`
	Proxies []Node            `json:"proxies"`
	Groups  []Group           `json:"groups"`
	Rules   []Rule            `json:"rules"`
	Hosts   map[string]string `json:"hosts"`
	Skipped []string          `json:"skipped"`
	Raw     []string          `json:"-"`
}

// Типы правил, которые умеем транслировать в sing-box.
var ruleTypeOK = map[string]bool{
	"DOMAIN": true, "DOMAIN-SUFFIX": true, "DOMAIN-KEYWORD": true,
	"IP-CIDR": true, "IP-CIDR6": true, "IP6-CIDR": true,
	"DST-PORT": true, "SRC-IP-CIDR": true, "GEOIP": true,
	"RULE-SET": true, "FINAL": true,
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// stripComment убирает комментарии Shadowrocket: строки на `#` и на `//`
// (второй вариант встречается в реальных шаблонах), а также хвостовой `#`.
// Хвостовой `#` вырезается только после пробела: иначе ломались бы URL с фрагментом.
func stripComment(s string) string {
	t := strings.TrimSpace(s)
	if strings.HasPrefix(t, "#") || strings.HasPrefix(t, ";") || strings.HasPrefix(t, "//") {
		return ""
	}
	if i := strings.Index(t, " #"); i >= 0 {
		return strings.TrimSpace(t[:i])
	}
	if i := strings.Index(t, "\t#"); i >= 0 {
		return strings.TrimSpace(t[:i])
	}
	return t
}

func sectionName(line string) (string, bool) {
	l := strings.TrimSpace(line)
	if len(l) >= 2 && strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
		return strings.ToLower(strings.TrimSpace(l[1 : len(l)-1])), true
	}
	return "", false
}

// ParseConf разбирает Shadowrocket-конфиг. Нераспознанное уходит в Skipped,
// разбор продолжается — один битый узел не должен убивать весь файл.
func ParseConf(text string) (*Conf, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("empty config")
	}
	c := &Conf{
		General: map[string]string{},
		Hosts:   map[string]string{},
		Raw:     strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n"),
	}
	section := ""
	for i, raw := range c.Raw {
		lineNo := i + 1
		if s, ok := sectionName(raw); ok {
			section = s
			continue
		}
		line := stripComment(raw)
		if line == "" {
			continue
		}
		switch section {
		case "general":
			k, v, ok := splitKV(line)
			if ok {
				c.General[strings.ToLower(k)] = v
			}
		case "host":
			if k, v, ok := splitKV(line); ok {
				c.Hosts[strings.ToLower(k)] = v
			}
		case "proxy":
			k, v, ok := splitKV(line)
			if !ok {
				c.Skipped = append(c.Skipped, fmt.Sprintf("line %d: malformed [Proxy] entry", lineNo))
				continue
			}
			n, err := parseProxyLine(k, v)
			if err != nil {
				c.Skipped = append(c.Skipped, fmt.Sprintf("line %d: %v", lineNo, err))
				continue
			}
			c.Proxies = append(c.Proxies, n)
		case "proxy group":
			k, v, ok := splitKV(line)
			if !ok {
				c.Skipped = append(c.Skipped, fmt.Sprintf("line %d: malformed [Proxy Group] entry", lineNo))
				continue
			}
			g, err := parseGroupLine(k, v)
			if err != nil {
				c.Skipped = append(c.Skipped, fmt.Sprintf("line %d: %v", lineNo, err))
				continue
			}
			c.Groups = append(c.Groups, g)
		case "rule":
			r, err := parseRuleLine(line)
			if err != nil {
				c.Skipped = append(c.Skipped, fmt.Sprintf("line %d: %v", lineNo, err))
				continue
			}
			r.Line = lineNo
			c.Rules = append(c.Rules, r)
		default:
			// [URL Rewrite], [MITM], [Header Rewrite] и прочее — iOS-специфично.
			if section != "" {
				c.Skipped = append(c.Skipped,
					fmt.Sprintf("line %d: section [%s] not supported", lineNo, section))
			}
		}
	}
	return c, nil
}

func splitKV(line string) (string, string, bool) {
	i := strings.Index(line, "=")
	if i < 0 {
		return "", "", false
	}
	k := strings.TrimSpace(line[:i])
	v := strings.TrimSpace(line[i+1:])
	if k == "" {
		return "", "", false
	}
	return k, v, true
}

func parseRuleLine(line string) (Rule, error) {
	f := splitCSV(line)
	if len(f) < 2 {
		return Rule{}, fmt.Errorf("malformed rule %q", line)
	}
	typ := strings.ToUpper(f[0])
	if typ == "FINAL" {
		return Rule{Type: "FINAL", Policy: strings.ToUpper(f[1])}, nil
	}
	if len(f) < 3 {
		return Rule{}, fmt.Errorf("rule %s: missing policy", typ)
	}
	if !ruleTypeOK[typ] {
		return Rule{}, fmt.Errorf("rule type %s not supported", typ)
	}
	r := Rule{Type: typ, Arg: f[1], Policy: strings.ToUpper(f[2])}
	if len(f) > 3 {
		r.Opts = f[3:]
	}
	// RULE-SET несёт policy третьим полем, как и остальные.
	return r, nil
}

// parseProxyLine разбирает `Alias = type, host, port, ...`.
func parseProxyLine(alias, val string) (Node, error) {
	f := splitCSV(val)
	if len(f) == 0 {
		return Node{}, fmt.Errorf("proxy %q: empty definition", alias)
	}
	typ := strings.ToLower(f[0])
	n := Node{Name: alias}
	// Позиционные поля до первого key=value.
	var pos []string
	kv := map[string]string{}
	for _, p := range f[1:] {
		if k, v, ok := splitKV(p); ok {
			kv[strings.ToLower(k)] = v
			continue
		}
		pos = append(pos, p)
	}
	if len(pos) < 2 {
		return Node{}, fmt.Errorf("proxy %q: missing server/port", alias)
	}
	n.Server = pos[0]
	port, err := strconv.Atoi(pos[1])
	if err != nil || port <= 0 || port > 65535 {
		return Node{}, fmt.Errorf("proxy %q: bad port %q", alias, pos[1])
	}
	n.Port = port

	switch typ {
	case "socks5", "socks":
		n.Type = "socks5"
		if len(pos) > 2 {
			n.User = pos[2]
		}
		if len(pos) > 3 {
			n.Password = pos[3]
		}
	case "socks5-tls":
		n.Type = "socks5"
		n.TLS = true
		if len(pos) > 2 {
			n.User = pos[2]
		}
		if len(pos) > 3 {
			n.Password = pos[3]
		}
	case "http", "https":
		n.Type = "http"
		n.TLS = typ == "https"
		if len(pos) > 2 {
			n.User = pos[2]
		}
		if len(pos) > 3 {
			n.Password = pos[3]
		}
	case "vmess":
		n.Type = "vmess"
		if len(pos) > 2 {
			n.Method = pos[2]
		}
		if len(pos) > 3 {
			n.UUID = pos[3]
		}
		if n.UUID == "" {
			n.UUID = kv["uuid"]
		}
		if n.UUID == "" {
			return Node{}, fmt.Errorf("proxy %q: vmess without uuid", alias)
		}
	case "vless":
		n.Type = "vless"
		if len(pos) > 2 {
			n.UUID = pos[2]
		}
		if n.UUID == "" {
			n.UUID = kv["uuid"]
		}
		if n.UUID == "" {
			return Node{}, fmt.Errorf("proxy %q: vless without uuid", alias)
		}
	case "ss", "shadowsocks":
		n.Type = "ss"
		if len(pos) > 2 {
			n.Method = pos[2]
		}
		if len(pos) > 3 {
			n.Password = pos[3]
		}
		if n.Password == "" {
			n.Password = kv["password"]
		}
	case "trojan":
		n.Type = "trojan"
		if len(pos) > 2 {
			n.Password = pos[2]
		}
		if n.Password == "" {
			n.Password = kv["password"]
		}
		n.TLS = true
	case "ssh":
		n.Type = "ssh"
		if len(pos) > 2 {
			n.User = pos[2]
		}
		if len(pos) > 3 {
			n.Password = pos[3]
		}
		if n.User == "" {
			n.User = kv["username"]
		}
		if n.Password == "" {
			n.Password = kv["password"]
		}
	default:
		return Node{}, fmt.Errorf("proxy %q: type %s not supported", alias, typ)
	}

	applyProxyKV(&n, kv)
	n.setID()
	if n.Latency == 0 {
		n.Latency = -1
	}
	return n, nil
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func applyProxyKV(n *Node, kv map[string]string) {
	if v, ok := kv["tls"]; ok {
		n.TLS = truthy(v)
	}
	if v, ok := kv["over-tls"]; ok {
		n.TLS = truthy(v)
	}
	if v := kv["sni"]; v != "" {
		n.SNI = v
	}
	if v := kv["peer"]; v != "" && n.SNI == "" {
		n.SNI = v
	}
	if v := kv["alpn"]; v != "" {
		n.ALPN = v
	}
	if v := kv["fp"]; v != "" {
		n.FP = v
	}
	if v := kv["flow"]; v != "" {
		n.Flow = v
	}
	if v := kv["pbk"]; v != "" {
		n.Reality.PublicKey = v
		n.TLS = true
	}
	if v := kv["sid"]; v != "" {
		n.Reality.ShortID = v
	}
	if v := kv["path"]; v != "" {
		n.Path = v
	}
	if v := kv["host"]; v != "" {
		n.Host = v
	}
	if v := kv["obfsparam"]; v != "" && n.Host == "" {
		n.Host = v
	}
	if v := kv["allowinsecure"]; truthy(v) {
		n.Insecure = true
	}
	if v := kv["alterid"]; v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			n.AlterID = i
		}
	}
	// Транспорт: transport=ws|grpc|h2 либо Shadowrocket-style obfs=websocket.
	net := strings.ToLower(kv["transport"])
	if net == "" {
		switch strings.ToLower(kv["obfs"]) {
		case "websocket", "ws":
			net = "ws"
		case "grpc":
			net = "grpc"
		case "h2", "http":
			net = "http"
		}
	}
	switch net {
	case "ws", "websocket":
		n.Network = "ws"
	case "grpc":
		n.Network = "grpc"
	case "h2", "http":
		n.Network = "http"
	case "httpupgrade":
		n.Network = "httpupgrade"
	case "tcp", "":
		// tcp — дефолт, поле остаётся пустым
	default:
		n.Network = net
	}
	if v := kv["servicename"]; v != "" {
		n.Path = v
	}
	if v := kv["privatekey"]; v != "" {
		n.PrivKey = v
	}
}

func parseGroupLine(name, val string) (Group, error) {
	f := splitCSV(val)
	if len(f) == 0 {
		return Group{}, fmt.Errorf("group %q: empty definition", name)
	}
	g := Group{Name: name, Type: strings.ToLower(f[0]), Interval: 600, Timeout: 5}
	switch g.Type {
	case "select", "url-test", "fallback", "load-balance":
	default:
		return Group{}, fmt.Errorf("group %q: type %s not supported", name, g.Type)
	}
	for _, p := range f[1:] {
		k, v, ok := splitKV(p)
		if !ok {
			g.Members = append(g.Members, p)
			continue
		}
		switch strings.ToLower(k) {
		case "interval":
			if i, err := strconv.Atoi(v); err == nil {
				g.Interval = i
			}
		case "timeout":
			if i, err := strconv.Atoi(v); err == nil {
				g.Timeout = i
			}
		case "url":
			g.URL = v
		case "policy-regex-filter":
			return Group{}, fmt.Errorf("group %q: policy-regex-filter not supported", name)
		}
	}
	if len(g.Members) == 0 {
		return Group{}, fmt.Errorf("group %q: no members", name)
	}
	return g, nil
}

// InsertRule вставляет правило первой строкой секции [Rule].
// Если секции нет — добавляет её в конец файла. Возвращает новый набор строк.
func (c *Conf) InsertRule(r Rule) []string {
	line := r.Type + "," + r.Arg + "," + r.Policy
	if r.Type == "FINAL" {
		line = "FINAL," + r.Policy
	}
	for i, raw := range c.Raw {
		if s, ok := sectionName(raw); ok && s == "rule" {
			out := make([]string, 0, len(c.Raw)+1)
			out = append(out, c.Raw[:i+1]...)
			out = append(out, line)
			out = append(out, c.Raw[i+1:]...)
			c.Raw = out
			return out
		}
	}
	out := append([]string{}, c.Raw...)
	if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
		out = append(out, "")
	}
	out = append(out, "[Rule]", line)
	c.Raw = out
	return out
}

// ruleText сериализует правило в строку Shadowrocket.
func ruleText(r Rule) string {
	if r.Type == "FINAL" {
		return "FINAL," + r.Policy
	}
	s := r.Type + "," + r.Arg + "," + r.Policy
	if len(r.Opts) > 0 {
		s += "," + strings.Join(r.Opts, ",")
	}
	return s
}

// ruleLines отдаёт номера строк (1-based) всех разобранных правил по порядку.
func (c *Conf) ruleLines() []int {
	out := make([]int, 0, len(c.Rules))
	for _, r := range c.Rules {
		out = append(out, r.Line)
	}
	return out
}

func (c *Conf) ruleAt(line int) (Rule, bool) {
	for _, r := range c.Rules {
		if r.Line == line {
			return r, true
		}
	}
	return Rule{}, false
}

// DeleteRuleLine удаляет строку правила по её номеру.
func (c *Conf) DeleteRuleLine(line int) bool {
	if _, ok := c.ruleAt(line); !ok {
		return false
	}
	i := line - 1
	c.Raw = append(c.Raw[:i:i], c.Raw[i+1:]...)
	return true
}

// UpdateRuleLine заменяет строку правила новым правилом.
func (c *Conf) UpdateRuleLine(line int, r Rule) bool {
	if _, ok := c.ruleAt(line); !ok {
		return false
	}
	c.Raw[line-1] = ruleText(r)
	return true
}

// MoveRuleLine меняет правило местами с соседним правилом.
// Порядок в [Rule] — это приоритет (first match wins), поэтому перестановка
// работает по соседнему ПРАВИЛУ, а не по соседней строке файла.
func (c *Conf) MoveRuleLine(line int, up bool) (int, bool) {
	lines := c.ruleLines()
	idx := -1
	for i, l := range lines {
		if l == line {
			idx = i
			break
		}
	}
	if idx < 0 {
		return 0, false
	}
	j := idx + 1
	if up {
		j = idx - 1
	}
	if j < 0 || j >= len(lines) {
		return 0, false
	}
	a, b := line-1, lines[j]-1
	c.Raw[a], c.Raw[b] = c.Raw[b], c.Raw[a]
	return lines[j], true
}

func (c *Conf) Text() string { return strings.Join(c.Raw, "\n") }

// Final возвращает политику FINAL (по умолчанию DIRECT).
func (c *Conf) Final() string {
	for _, r := range c.Rules {
		if r.Type == "FINAL" {
			return r.Policy
		}
	}
	return "DIRECT"
}
