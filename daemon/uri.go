package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// b64decode принимает std/url-safe base64 с любым паддингом.
func b64decode(s string) ([]byte, bool) {
	s = strings.TrimSpace(s)
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	if s == "" {
		return nil, false
	}
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.URLEncoding,
		base64.RawStdEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, true
		}
	}
	return nil, false
}

func hostPort(s string) (string, int, error) {
	h, p, err := net.SplitHostPort(s)
	if err != nil {
		return "", 0, fmt.Errorf("bad host:port %q", s)
	}
	port, err := strconv.Atoi(p)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("bad port %q", p)
	}
	return h, port, nil
}

// ParseURI разбирает ссылку на узел одной из поддерживаемых схем.
func ParseURI(s string) (Node, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Node{}, fmt.Errorf("empty uri")
	}
	i := strings.Index(s, "://")
	if i < 0 {
		return Node{}, fmt.Errorf("not a uri")
	}
	scheme := strings.ToLower(s[:i])
	switch scheme {
	case "vless":
		return parseVLESS(s)
	case "vmess":
		return parseVMess(s)
	case "ss":
		return parseSS(s)
	case "trojan":
		return parseTrojan(s)
	case "socks", "socks5", "socks5h":
		return parseSocksHTTP(s, "socks5")
	case "http", "https":
		return parseSocksHTTP(s, "http")
	case "ssh":
		return parseSSH(s)
	}
	return Node{}, fmt.Errorf("scheme %s not supported", scheme)
}

func fragName(u *url.URL, fallback string) string {
	if u.Fragment != "" {
		return u.Fragment
	}
	return fallback
}

func applyTLSQuery(n *Node, q url.Values) {
	sec := strings.ToLower(q.Get("security"))
	switch sec {
	case "tls":
		n.TLS = true
	case "reality":
		n.TLS = true
		n.Reality.PublicKey = q.Get("pbk")
		n.Reality.ShortID = q.Get("sid")
	case "none", "":
	default:
		n.TLS = true
	}
	if n.SNI == "" {
		n.SNI = q.Get("sni")
	}
	if n.SNI == "" {
		n.SNI = q.Get("peer")
	}
	n.ALPN = q.Get("alpn")
	n.FP = q.Get("fp")
	if truthy(q.Get("allowInsecure")) || truthy(q.Get("insecure")) {
		n.Insecure = true
	}
	switch strings.ToLower(q.Get("type")) {
	case "ws":
		n.Network = "ws"
	case "grpc":
		n.Network = "grpc"
	case "http", "h2":
		n.Network = "http"
	case "httpupgrade":
		n.Network = "httpupgrade"
	}
	if p := q.Get("path"); p != "" {
		n.Path = p
	}
	if sn := q.Get("serviceName"); sn != "" {
		n.Path = sn
	}
	if h := q.Get("host"); h != "" {
		n.Host = h
	}
}

func parseVLESS(s string) (Node, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Node{}, err
	}
	host, port, err := hostPort(u.Host)
	if err != nil {
		return Node{}, err
	}
	if u.User == nil || u.User.Username() == "" {
		return Node{}, fmt.Errorf("vless: missing uuid")
	}
	q := u.Query()
	n := Node{
		Type:   "vless",
		Server: host,
		Port:   port,
		UUID:   u.User.Username(),
		Flow:   q.Get("flow"),
	}
	applyTLSQuery(&n, q)
	n.Name = fragName(u, fmt.Sprintf("vless-%s", host))
	n.setID()
	n.Latency = -1
	return n, nil
}

type vmessJSON struct {
	PS   string      `json:"ps"`
	Add  string      `json:"add"`
	Port json.Number `json:"port"`
	ID   string      `json:"id"`
	Aid  json.Number `json:"aid"`
	Scy  string      `json:"scy"`
	Net  string      `json:"net"`
	Type string      `json:"type"`
	Host string      `json:"host"`
	Path string      `json:"path"`
	TLS  string      `json:"tls"`
	SNI  string      `json:"sni"`
	ALPN string      `json:"alpn"`
	FP   string      `json:"fp"`
}

func parseVMess(s string) (Node, error) {
	body := s[len("vmess://"):]
	// Формат (a): base64(JSON).
	if b, ok := b64decode(body); ok && json.Valid(b) {
		var v vmessJSON
		if err := json.Unmarshal(b, &v); err != nil {
			return Node{}, err
		}
		port, err := strconv.Atoi(v.Port.String())
		if err != nil || port <= 0 || port > 65535 {
			return Node{}, fmt.Errorf("vmess: bad port %q", v.Port.String())
		}
		n := Node{
			Type: "vmess", Server: v.Add, Port: port, UUID: v.ID,
			Method: v.Scy, Path: v.Path, Host: v.Host,
			SNI: v.SNI, ALPN: v.ALPN, FP: v.FP,
			TLS: strings.EqualFold(v.TLS, "tls"),
		}
		if v.Aid != "" {
			n.AlterID, _ = strconv.Atoi(v.Aid.String())
		}
		switch strings.ToLower(v.Net) {
		case "ws":
			n.Network = "ws"
		case "grpc":
			n.Network = "grpc"
		case "h2", "http":
			n.Network = "http"
		case "httpupgrade":
			n.Network = "httpupgrade"
		}
		if n.Server == "" || n.UUID == "" {
			return Node{}, fmt.Errorf("vmess: missing server or uuid")
		}
		n.Name = v.PS
		if n.Name == "" {
			n.Name = "vmess-" + n.Server
		}
		n.setID()
		n.Latency = -1
		return n, nil
	}
	// Формат (c): обычный URL vmess://uuid@host:port?type=ws&... (встречается в подписках).
	if strings.Contains(body, "@") {
		if u, err := url.Parse(s); err == nil && u.User != nil && u.User.Username() != "" {
			if host, port, herr := hostPort(u.Host); herr == nil {
				q := u.Query()
				n := Node{Type: "vmess", Server: host, Port: port, UUID: u.User.Username()}
				n.Method = q.Get("encryption")
				if v := q.Get("alterId"); v != "" {
					n.AlterID, _ = strconv.Atoi(v)
				}
				applyTLSQuery(&n, q)
				n.Name = fragName(u, "vmess-"+host)
				n.setID()
				n.Latency = -1
				return n, nil
			}
		}
	}
	// Формат (b), Shadowrocket: base64(method:uuid@host:port)?query#name
	rest := body
	frag := ""
	if i := strings.Index(rest, "#"); i >= 0 {
		frag, _ = url.QueryUnescape(rest[i+1:])
		rest = rest[:i]
	}
	query := ""
	if i := strings.Index(rest, "?"); i >= 0 {
		query = rest[i+1:]
		rest = rest[:i]
	}
	b, ok := b64decode(rest)
	if !ok {
		return Node{}, fmt.Errorf("vmess: undecodable payload")
	}
	core := string(b)
	at := strings.LastIndex(core, "@")
	if at < 0 {
		return Node{}, fmt.Errorf("vmess: malformed payload")
	}
	cred, addr := core[:at], core[at+1:]
	host, port, err := hostPort(addr)
	if err != nil {
		return Node{}, err
	}
	method, uuid := "auto", cred
	if c := strings.Index(cred, ":"); c >= 0 {
		method, uuid = cred[:c], cred[c+1:]
	}
	if uuid == "" {
		return Node{}, fmt.Errorf("vmess: missing uuid")
	}
	q, _ := url.ParseQuery(query)
	n := Node{Type: "vmess", Server: host, Port: port, UUID: uuid, Method: method}
	if v := q.Get("obfs"); v != "" {
		switch strings.ToLower(v) {
		case "websocket", "ws":
			n.Network = "ws"
		case "grpc":
			n.Network = "grpc"
		case "h2", "http":
			n.Network = "http"
		}
	}
	n.Path = q.Get("path")
	n.Host = q.Get("obfsParam")
	n.TLS = truthy(q.Get("tls"))
	n.SNI = q.Get("peer")
	if v := q.Get("alterId"); v != "" {
		n.AlterID, _ = strconv.Atoi(v)
	}
	if truthy(q.Get("allowInsecure")) {
		n.Insecure = true
	}
	n.Name = frag
	if n.Name == "" {
		n.Name = q.Get("remarks")
	}
	if n.Name == "" {
		n.Name = "vmess-" + host
	}
	n.setID()
	n.Latency = -1
	return n, nil
}

func parseSS(s string) (Node, error) {
	body := s[len("ss://"):]
	frag := ""
	if i := strings.Index(body, "#"); i >= 0 {
		frag, _ = url.QueryUnescape(body[i+1:])
		body = body[:i]
	}
	query := ""
	if i := strings.Index(body, "?"); i >= 0 {
		query = body[i+1:]
		body = body[:i]
	}
	if q, _ := url.ParseQuery(query); q.Get("plugin") != "" {
		return Node{}, fmt.Errorf("ss: plugin not supported")
	}
	var method, password, addr string
	if at := strings.LastIndex(body, "@"); at >= 0 {
		// SIP002: base64(method:pass)@host:port
		cred := body[:at]
		addr = body[at+1:]
		b, ok := b64decode(cred)
		if !ok {
			// Может быть уже в открытом виде.
			b = []byte(cred)
		}
		c := strings.Index(string(b), ":")
		if c < 0 {
			return Node{}, fmt.Errorf("ss: malformed credentials")
		}
		method, password = string(b)[:c], string(b)[c+1:]
	} else {
		// Legacy: base64(method:pass@host:port)
		b, ok := b64decode(body)
		if !ok {
			return Node{}, fmt.Errorf("ss: undecodable payload")
		}
		core := string(b)
		at := strings.LastIndex(core, "@")
		if at < 0 {
			return Node{}, fmt.Errorf("ss: malformed payload")
		}
		cred := core[:at]
		addr = core[at+1:]
		c := strings.Index(cred, ":")
		if c < 0 {
			return Node{}, fmt.Errorf("ss: malformed credentials")
		}
		method, password = cred[:c], cred[c+1:]
	}
	host, port, err := hostPort(addr)
	if err != nil {
		return Node{}, err
	}
	if method == "" || password == "" {
		return Node{}, fmt.Errorf("ss: missing method or password")
	}
	n := Node{Type: "ss", Server: host, Port: port, Method: method, Password: password}
	n.Name = frag
	if n.Name == "" {
		n.Name = "ss-" + host
	}
	n.setID()
	n.Latency = -1
	return n, nil
}

func parseTrojan(s string) (Node, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Node{}, err
	}
	host, port, err := hostPort(u.Host)
	if err != nil {
		return Node{}, err
	}
	if u.User == nil || u.User.Username() == "" {
		return Node{}, fmt.Errorf("trojan: missing password")
	}
	n := Node{Type: "trojan", Server: host, Port: port, Password: u.User.Username(), TLS: true}
	applyTLSQuery(&n, u.Query())
	n.TLS = true
	n.Name = fragName(u, "trojan-"+host)
	n.setID()
	n.Latency = -1
	return n, nil
}

func parseSocksHTTP(s, typ string) (Node, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Node{}, err
	}
	host := u.Hostname()
	if host == "" {
		return Node{}, fmt.Errorf("%s: missing host", typ)
	}
	port := 0
	if p := u.Port(); p != "" {
		port, err = strconv.Atoi(p)
		if err != nil {
			return Node{}, fmt.Errorf("%s: bad port", typ)
		}
	} else if typ == "http" {
		port = 80
		if strings.HasPrefix(strings.ToLower(s), "https://") {
			port = 443
		}
	} else {
		port = 1080
	}
	if port <= 0 || port > 65535 {
		return Node{}, fmt.Errorf("%s: bad port", typ)
	}
	n := Node{Type: typ, Server: host, Port: port}
	if typ == "http" && strings.HasPrefix(strings.ToLower(s), "https://") {
		n.TLS = true
	}
	if u.User != nil {
		n.User = u.User.Username()
		n.Password, _ = u.User.Password()
	}
	n.Name = fragName(u, fmt.Sprintf("%s-%s", typ, host))
	n.setID()
	n.Latency = -1
	return n, nil
}

func parseSSH(s string) (Node, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Node{}, err
	}
	host := u.Hostname()
	if host == "" {
		return Node{}, fmt.Errorf("ssh: missing host")
	}
	port := 22
	if p := u.Port(); p != "" {
		port, err = strconv.Atoi(p)
		if err != nil || port <= 0 || port > 65535 {
			return Node{}, fmt.Errorf("ssh: bad port")
		}
	}
	n := Node{Type: "ssh", Server: host, Port: port}
	if u.User != nil {
		n.User = u.User.Username()
		n.Password, _ = u.User.Password()
	}
	if n.User == "" {
		return Node{}, fmt.Errorf("ssh: missing user")
	}
	q := u.Query()
	n.PrivKey = q.Get("privateKey")
	if n.PrivKey == "" {
		n.PrivKey = q.Get("private_key")
	}
	if n.Password == "" && n.PrivKey == "" {
		return Node{}, fmt.Errorf("ssh: neither password nor privateKey")
	}
	n.Name = fragName(u, "ssh-"+host)
	n.setID()
	n.Latency = -1
	return n, nil
}

// ParseAWGConf принимает INI AmneziaWG/WireGuard и возвращает узел + очищенный конфиг.
// Строки DNS выкидываются: в UT нет resolvconf.
func ParseAWGConf(name, text string) (Node, string, error) {
	if len(text) > 65536 {
		return Node{}, "", fmt.Errorf("awg: config too large")
	}
	var hasIface, hasPeer, priv, pub bool
	var endpoint string
	var kept []string
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(raw)
		lower := strings.ToLower(t)
		switch {
		case strings.HasPrefix(lower, "[interface]"):
			hasIface = true
		case strings.HasPrefix(lower, "[peer]"):
			hasPeer = true
		case strings.HasPrefix(lower, "privatekey"):
			priv = true
		case strings.HasPrefix(lower, "publickey"):
			pub = true
		case strings.HasPrefix(lower, "endpoint"):
			if _, v, ok := splitKV(t); ok {
				endpoint = v
			}
		case strings.HasPrefix(lower, "dns"):
			continue // resolvconf в UT отсутствует
		}
		kept = append(kept, raw)
	}
	if !(hasIface && hasPeer && priv && pub) {
		return Node{}, "", fmt.Errorf("awg: missing [Interface]/[Peer], PrivateKey or PublicKey")
	}
	n := Node{Type: "awg", Name: name, AWGConf: name}
	if endpoint != "" {
		if h, p, err := hostPort(endpoint); err == nil {
			n.Server, n.Port = h, p
		} else {
			n.Server = endpoint
		}
	}
	n.setID()
	n.Latency = -1
	return n, strings.Join(kept, "\n") + "\n", nil
}
