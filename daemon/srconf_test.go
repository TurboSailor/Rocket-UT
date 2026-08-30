package main

import (
	"fmt"
	"strings"
	"testing"
)

// Фрагмент реального конфига (gist wgzhao) + dlisin RULE-SET-строки.
const sampleConf = `# Shadowrocket: 2020-12-13 10:10:56
[General]
bypass-system = true
skip-proxy = 192.168.0.0/16, 10.0.0.0/8, localhost, *.local
bypass-tun = 10.0.0.0/8,100.64.0.0/10,127.0.0.0/8
#dns-server = 119.29.29.29
ipv6 = false

[Proxy]
MySocks = socks5, 10.0.0.1, 1080, user, pass
MyHttp = http, proxy.example.com, 8080
MyVMess = vmess, 1.2.3.4, 443, aes-256-gcm, b831381d-6324-4d53-ad4f-8cda48b30811, transport=ws, path=/ws, tls=1
MyVless = vless, 5.6.7.8, 443, b831381d-6324-4d53-ad4f-8cda48b30811, tls=1, sni=a.com, flow=xtls-rprx-vision
MySS = ss, 9.9.9.9, 8388, aes-256-gcm, secretpass
MyTrojan = trojan, t.example.com, 443, trojanpass
MySSH = ssh, ssh.example.com, 22, root, sshpass
Broken = wireguard, 1.1.1.1, 51820

[Proxy Group]
MyGroup = url-test,MySocks,MyHttp,interval=300,timeout=3,url=https://cp.cloudflare.com/generate_204
BadGroup = url-test,interval=600,policy-regex-filter=*

[Rule]
DOMAIN-SUFFIX,rfi.fr,PROXY
DOMAIN,box.com,PROXY
DOMAIN,qq.com,DIRECT
IP-CIDR,17.0.0.0/8,DIRECT
DOMAIN-KEYWORD,google,PROXY
DOMAIN-SUFFIX,doubleclick.net,REJECT
GEOIP,CN,DIRECT
DST-PORT,443,PROXY
RULE-SET,https://example.com/list.list,PROXY,no-resolve
IP-ASN,4134,DIRECT
USER-AGENT,Mozilla*,PROXY
FINAL,DIRECT

[Host]
localhost = 127.0.0.1

[URL Rewrite]
^http://example.com http://example.org 302
`

func TestParseConfGeneral(t *testing.T) {
	c, err := ParseConf(sampleConf)
	if err != nil {
		t.Fatalf("ParseConf: %v", err)
	}
	if c.General["bypass-system"] != "true" {
		t.Errorf("bypass-system = %q", c.General["bypass-system"])
	}
	if c.General["ipv6"] != "false" {
		t.Errorf("ipv6 = %q", c.General["ipv6"])
	}
	// Закомментированный dns-server не должен попасть в General.
	if _, ok := c.General["dns-server"]; ok {
		t.Error("commented dns-server was parsed")
	}
	if got := c.Hosts["localhost"]; got != "127.0.0.1" {
		t.Errorf("host localhost = %q", got)
	}
}

func TestParseConfRuleOrderAndFinal(t *testing.T) {
	c, err := ParseConf(sampleConf)
	if err != nil {
		t.Fatal(err)
	}
	// FINAL входит в Rules, но first-match-wins проверяем по не-FINAL порядку.
	want := []struct{ typ, arg, policy string }{
		{"DOMAIN-SUFFIX", "rfi.fr", "PROXY"},
		{"DOMAIN", "box.com", "PROXY"},
		{"DOMAIN", "qq.com", "DIRECT"},
		{"IP-CIDR", "17.0.0.0/8", "DIRECT"},
		{"DOMAIN-KEYWORD", "google", "PROXY"},
		{"DOMAIN-SUFFIX", "doubleclick.net", "REJECT"},
		{"GEOIP", "CN", "DIRECT"},
		{"DST-PORT", "443", "PROXY"},
		{"RULE-SET", "https://example.com/list.list", "PROXY"},
	}
	if len(c.Rules) != len(want)+1 {
		t.Fatalf("rules = %d, want %d (+FINAL): %+v", len(c.Rules), len(want)+1, c.Rules)
	}
	for i, w := range want {
		got := c.Rules[i]
		if got.Type != w.typ || got.Arg != w.arg || got.Policy != w.policy {
			t.Errorf("rule[%d] = %+v, want %v", i, got, w)
		}
	}
	if c.Final() != "DIRECT" {
		t.Errorf("Final() = %q, want DIRECT", c.Final())
	}
	// no-resolve сохраняется как опция.
	rs := c.Rules[8]
	if len(rs.Opts) != 1 || rs.Opts[0] != "no-resolve" {
		t.Errorf("rule-set opts = %v", rs.Opts)
	}
}

func TestParseConfSkipsUnsupported(t *testing.T) {
	c, err := ParseConf(sampleConf)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(c.Skipped, "\n")
	for _, want := range []string{"IP-ASN", "USER-AGENT", "wireguard", "policy-regex-filter", "url rewrite"} {
		if !strings.Contains(strings.ToLower(joined), strings.ToLower(want)) {
			t.Errorf("Skipped missing %q; got:\n%s", want, joined)
		}
	}
}

func TestParseProxyLines(t *testing.T) {
	c, err := ParseConf(sampleConf)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Node{}
	for _, p := range c.Proxies {
		byName[p.Name] = p
	}
	if len(byName) != 7 {
		t.Fatalf("proxies = %d, want 7: %v", len(byName), byName)
	}
	tests := []struct {
		name  string
		check func(Node) error
	}{
		{"MySocks", func(n Node) error {
			if n.Type != "socks5" || n.Server != "10.0.0.1" || n.Port != 1080 ||
				n.User != "user" || n.Password != "pass" {
				return errf("socks5 mismatch: %+v", n)
			}
			return nil
		}},
		{"MyHttp", func(n Node) error {
			if n.Type != "http" || n.Port != 8080 {
				return errf("http mismatch: %+v", n)
			}
			return nil
		}},
		{"MyVMess", func(n Node) error {
			if n.Type != "vmess" || n.UUID != "b831381d-6324-4d53-ad4f-8cda48b30811" ||
				n.Method != "aes-256-gcm" || n.Network != "ws" || n.Path != "/ws" || !n.TLS {
				return errf("vmess mismatch: %+v", n)
			}
			return nil
		}},
		{"MyVless", func(n Node) error {
			if n.Type != "vless" || !n.TLS || n.SNI != "a.com" || n.Flow != "xtls-rprx-vision" {
				return errf("vless mismatch: %+v", n)
			}
			return nil
		}},
		{"MySS", func(n Node) error {
			if n.Type != "ss" || n.Method != "aes-256-gcm" || n.Password != "secretpass" {
				return errf("ss mismatch: %+v", n)
			}
			return nil
		}},
		{"MyTrojan", func(n Node) error {
			if n.Type != "trojan" || n.Password != "trojanpass" || !n.TLS {
				return errf("trojan mismatch: %+v", n)
			}
			return nil
		}},
		{"MySSH", func(n Node) error {
			if n.Type != "ssh" || n.User != "root" || n.Password != "sshpass" || n.Port != 22 {
				return errf("ssh mismatch: %+v", n)
			}
			return nil
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, ok := byName[tc.name]
			if !ok {
				t.Fatalf("proxy %s not parsed", tc.name)
			}
			if n.ID == "" {
				t.Error("empty ID")
			}
			if err := tc.check(n); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestParseGroups(t *testing.T) {
	c, err := ParseConf(sampleConf)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Groups) != 1 {
		t.Fatalf("groups = %d, want 1 (BadGroup must be skipped): %+v", len(c.Groups), c.Groups)
	}
	g := c.Groups[0]
	if g.Name != "MyGroup" || g.Type != "url-test" || g.Interval != 300 || g.Timeout != 3 {
		t.Errorf("group = %+v", g)
	}
	if len(g.Members) != 2 || g.Members[0] != "MySocks" || g.Members[1] != "MyHttp" {
		t.Errorf("members = %v", g.Members)
	}
}

func TestInsertRule(t *testing.T) {
	c, err := ParseConf(sampleConf)
	if err != nil {
		t.Fatal(err)
	}
	before := len(c.Raw)
	c.InsertRule(Rule{Type: "DOMAIN-SUFFIX", Arg: "ipify.org", Policy: "PROXY"})
	if len(c.Raw) != before+1 {
		t.Fatalf("Raw grew by %d, want 1", len(c.Raw)-before)
	}
	lines := strings.Split(c.Text(), "\n")
	idx := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "[Rule]" {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("[Rule] section vanished")
	}
	if lines[idx+1] != "DOMAIN-SUFFIX,ipify.org,PROXY" {
		t.Errorf("first rule line = %q", lines[idx+1])
	}
	// Остальной файл не должен пострадать.
	if !strings.Contains(c.Text(), "FINAL,DIRECT") ||
		!strings.Contains(c.Text(), "MySocks = socks5, 10.0.0.1, 1080, user, pass") ||
		!strings.Contains(c.Text(), "[Host]") {
		t.Error("InsertRule damaged the rest of the file")
	}
	// Правило должно быть видно парсеру и стоять первым.
	c2, err := ParseConf(c.Text())
	if err != nil {
		t.Fatal(err)
	}
	if c2.Rules[0].Arg != "ipify.org" {
		t.Errorf("reparsed first rule = %+v", c2.Rules[0])
	}
}

func TestInsertRuleNoSection(t *testing.T) {
	c, err := ParseConf("[General]\nipv6 = false\n")
	if err != nil {
		t.Fatal(err)
	}
	c.InsertRule(Rule{Type: "DOMAIN", Arg: "a.com", Policy: "REJECT"})
	c2, err := ParseConf(c.Text())
	if err != nil {
		t.Fatal(err)
	}
	if len(c2.Rules) != 1 || c2.Rules[0].Policy != "REJECT" {
		t.Errorf("rules = %+v\ntext:\n%s", c2.Rules, c.Text())
	}
}

func TestParseConfEmpty(t *testing.T) {
	if _, err := ParseConf("   \n\n"); err == nil {
		t.Error("expected error on empty config")
	}
}

func errf(format string, a ...any) error { return fmt.Errorf(format, a...) }
