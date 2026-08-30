package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// Форматы и мусор ниже взяты из реальных публичных списков, на которых
// приложение падало: dlisin/shadowrocket-config, misha-tgshv, blackmatrix7.

func TestBareRuleFormats(t *testing.T) {
	tests := []struct {
		in       string
		wantType string
		wantArg  string
		wantOK   bool
	}{
		{".apple.com", "DOMAIN-SUFFIX", "apple.com", true},
		{"example.com", "DOMAIN-SUFFIX", "example.com", true},
		{"104.20.39.144", "IP-CIDR", "104.20.39.144/32", true},
		{"10.0.0.0/8", "IP-CIDR", "10.0.0.0/8", true},
		{"2001:db8::/32", "IP-CIDR", "2001:db8::/32", true},
		{"2606:4700::1111", "IP-CIDR", "2606:4700::1111/128", true},
		{"", "", "", false},
		{"not a domain", "", "", false},
		{"localhost", "", "", false},
		{"999.999.999.999/8", "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := bareRule(tc.in, "PROXY")
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (got %+v)", ok, tc.wantOK, got)
			}
			if !ok {
				return
			}
			if got.Type != tc.wantType || got.Arg != tc.wantArg || got.Policy != "PROXY" {
				t.Errorf("got %+v, want %s,%s", got, tc.wantType, tc.wantArg)
			}
		})
	}
}

func TestDstPortRangeAndDirtyChars(t *testing.T) {
	tests := []struct {
		arg  string
		want string
	}{
		{"3478", `{"port":[3478]}`},
		{"596-599", `{"port_range":["596:599"]}`},
		// en-dash + zero-width space, как в voice_ports.list
		{"19302\u200b\u201319309", `{"port_range":["19302:19309"]}`},
		{" 443 ", `{"port":[443]}`},
	}
	for _, tc := range tests {
		t.Run(tc.arg, func(t *testing.T) {
			got, err := ruleToSB(Rule{Type: "DST-PORT", Arg: tc.arg})
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(got)
			if string(b) != tc.want {
				t.Errorf("got %s, want %s", b, tc.want)
			}
		})
	}
	for _, bad := range []string{"0", "70000", "abc", "599-596", "-5"} {
		if _, err := ruleToSB(Rule{Type: "DST-PORT", Arg: bad}); err == nil {
			t.Errorf("expected error for port %q", bad)
		}
	}
}

// Регрессия: DoH/DoT по имени хоста без domain_resolver валили старт sing-box
// с "missing domain resolver for domain server address".
func TestDNSHostnameServersGetResolver(t *testing.T) {
	conf, err := ParseConf(`[General]
ipv6 = false
dns-server = https://security.cloudflare-dns.com/dns-query,tls://one.one.one.one:853,1.1.1.1

[Rule]
FINAL,DIRECT
`)
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildSingbox("config", nil, conf, newRulesetCache())
	if err != nil {
		t.Fatal(err)
	}
	servers := b.Config["dns"].(jmap)["servers"].([]jmap)
	var bootstrap string
	for _, s := range servers {
		if srv, ok := s["server"].(string); ok && srv == "1.1.1.1" {
			bootstrap = s["tag"].(string)
		}
	}
	if bootstrap == "" {
		t.Fatal("no literal-IP bootstrap server present")
	}
	found := 0
	for _, s := range servers {
		srv, _ := s["server"].(string)
		if srv == "security.cloudflare-dns.com" || srv == "one.one.one.one" {
			found++
			if s["domain_resolver"] != bootstrap {
				t.Errorf("server %q domain_resolver = %v, want %q", srv, s["domain_resolver"], bootstrap)
			}
		}
		if srv == "1.1.1.1" {
			if _, ok := s["domain_resolver"]; ok {
				t.Error("literal-IP server must not carry domain_resolver")
			}
		}
	}
	if found != 2 {
		t.Errorf("hostname servers found = %d, want 2", found)
	}
}

func TestDNSBootstrapAddedWhenAllHostnames(t *testing.T) {
	conf, err := ParseConf("[General]\ndns-server = https://dns.google/dns-query\n\n[Rule]\nFINAL,DIRECT\n")
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildSingbox("config", nil, conf, newRulesetCache())
	if err != nil {
		t.Fatal(err)
	}
	servers := b.Config["dns"].(jmap)["servers"].([]jmap)
	if len(servers) < 2 {
		t.Fatalf("bootstrap server was not added: %+v", servers)
	}
	if servers[0]["server"] != "1.1.1.1" {
		t.Errorf("first server = %+v, want literal-IP bootstrap", servers[0])
	}
	for _, s := range servers {
		if s["server"] == "dns.google" && s["domain_resolver"] != servers[0]["tag"] {
			t.Errorf("dns.google resolver = %v", s["domain_resolver"])
		}
	}
}

// Настоящий конфиг dlisin целиком: он не должен терять правила и должен
// давать конфиг, который принимает сам sing-box.
func TestRealConfigBuildsValid(t *testing.T) {
	bin := findSingBoxBinary()
	if bin == "" {
		t.Skip("sing-box binary not found; run `make bins`")
	}
	conf, err := ParseConf(realDlisinConf)
	if err != nil {
		t.Fatal(err)
	}
	if len(conf.Rules) == 0 {
		t.Fatal("no rules parsed")
	}
	if conf.General["dns-server"] == "" {
		t.Fatal("dns-server not parsed")
	}
	n, err := ParseURI("socks5://127.0.0.1:1080#t")
	if err != nil {
		t.Fatal(err)
	}
	// Сеть в тестах не используется: RULE-SET без кэша попадёт в Skipped,
	// а конфиг всё равно обязан остаться валидным.
	b, err := buildSingbox("config", &n, conf, newRulesetCache())
	if err != nil {
		t.Fatal(err)
	}
	checkWithSingBox(t, bin, b.Config)

	dns := b.Config["dns"].(jmap)
	if dns["strategy"] != "ipv4_only" {
		t.Errorf("ipv6=false must give ipv4_only, got %v", dns["strategy"])
	}
	// tun-excluded-routes должен доехать до inbound.
	tun := b.Config["inbounds"].([]jmap)[0]
	excl, _ := tun["route_exclude_address"].([]string)
	if len(excl) == 0 {
		t.Error("tun-excluded-routes lost")
	}
	if !strings.Contains(strings.Join(excl, ","), "100.64.0.0/10") {
		t.Errorf("route_exclude_address = %v", excl)
	}
}

// Фрагмент реального конфига dlisin/shadowrocket-config.
const realDlisinConf = `[General]
bypass-system = true
ipv6 = false
prefer-ipv6 = false
private-ip-answer = true
dns-direct-system = false
dns-server = tls://1.1.1.1:853,https://security.cloudflare-dns.com/dns-query,1.1.1.2,8.8.8.8
fallback-dns-server = tls://77.88.8.88:853,system
hijack-dns = :53
skip-proxy = 192.168.0.0/16,10.0.0.0/8,172.16.0.0/12,localhost,*.local,captive.apple.com
tun-excluded-routes = 10.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,224.0.0.0/4
always-real-ip = *
icmp-auto-reply = false
udp-policy-not-supported-behaviour = REJECT
update-url = https://raw.githubusercontent.com/dlisin/shadowrocket-config/master/shadowrocket.conf

[Proxy Group]
#AUTO = url-test,interval=600,timeout=5,url=https://cp.cloudflare.com/generate_204,policy-regex-filter=*

[Rule]
RULE-SET,https://raw.githubusercontent.com/dlisin/shadowrocket-config/master/reject.list,REJECT
RULE-SET,https://raw.githubusercontent.com/dlisin/shadowrocket-config/master/proxy.list,PROXY
DOMAIN-SUFFIX,openai.com,PROXY
IP-CIDR,17.0.0.0/8,DIRECT
GEOIP,RU,DIRECT
FINAL,DIRECT

[Host]
localhost = 127.0.0.1
`
