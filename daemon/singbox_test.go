package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRuleToSB(t *testing.T) {
	tests := []struct {
		rule Rule
		want string
	}{
		{Rule{Type: "DOMAIN", Arg: "a.com"}, `{"domain":["a.com"]}`},
		{Rule{Type: "DOMAIN-SUFFIX", Arg: "b.com"}, `{"domain_suffix":["b.com"]}`},
		{Rule{Type: "DOMAIN-KEYWORD", Arg: "kw"}, `{"domain_keyword":["kw"]}`},
		{Rule{Type: "IP-CIDR", Arg: "17.0.0.0/8"}, `{"ip_cidr":["17.0.0.0/8"]}`},
		{Rule{Type: "IP-CIDR", Arg: "1.2.3.4"}, `{"ip_cidr":["1.2.3.4/32"]}`},
		{Rule{Type: "IP-CIDR6", Arg: "2001:db8::/32"}, `{"ip_cidr":["2001:db8::/32"]}`},
		{Rule{Type: "SRC-IP-CIDR", Arg: "192.168.0.0/16"}, `{"source_ip_cidr":["192.168.0.0/16"]}`},
		{Rule{Type: "DST-PORT", Arg: "443"}, `{"port":[443]}`},
		{Rule{Type: "GEOIP", Arg: "CN"}, `{"rule_set":["geoip-cn"]}`},
	}
	for _, tc := range tests {
		t.Run(tc.rule.Type+","+tc.rule.Arg, func(t *testing.T) {
			got, err := ruleToSB(tc.rule)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(got)
			if string(b) != tc.want {
				t.Errorf("got %s, want %s", b, tc.want)
			}
		})
	}
}

func TestRuleToSBUnsupported(t *testing.T) {
	for _, r := range []Rule{
		{Type: "IP-ASN", Arg: "4134"},
		{Type: "USER-AGENT", Arg: "Mozilla*"},
		{Type: "IP-CIDR", Arg: "not-an-ip"},
		{Type: "DST-PORT", Arg: "99999"},
	} {
		if _, err := ruleToSB(r); err == nil {
			t.Errorf("expected error for %+v", r)
		}
	}
}

func TestPolicyTag(t *testing.T) {
	groups := map[string]bool{"MyGroup": true}
	tests := []struct {
		policy     string
		wantTag    string
		wantReject bool
	}{
		{"DIRECT", tagDirect, false},
		{"PROXY", tagProxy, false},
		{"REJECT", "", true},
		{"REJECT-TINYGIF", "", true},
		{"MyGroup", "group-MyGroup", false},
		{"Unknown", tagProxy, false},
	}
	for _, tc := range tests {
		t.Run(tc.policy, func(t *testing.T) {
			tag, rej := policyTag(tc.policy, groups, tagProxy)
			if tag != tc.wantTag || rej != tc.wantReject {
				t.Errorf("got (%q,%v), want (%q,%v)", tag, rej, tc.wantTag, tc.wantReject)
			}
		})
	}
}

func TestMergeRules(t *testing.T) {
	in := []jmap{
		{"domain_suffix": []string{"a.com"}, "outbound": "proxy"},
		{"domain_suffix": []string{"b.com"}, "outbound": "proxy"},
		{"domain_suffix": []string{"c.com"}, "outbound": "direct"},
		{"ip_cidr": []string{"1.0.0.0/8"}, "outbound": "direct"},
	}
	out := mergeRules(in)
	if len(out) != 3 {
		t.Fatalf("merged into %d rules, want 3: %+v", len(out), out)
	}
	got := out[0]["domain_suffix"].([]string)
	if len(got) != 2 || got[0] != "a.com" || got[1] != "b.com" {
		t.Errorf("first rule = %v", got)
	}
}

func buildFor(t *testing.T, mode string, n *Node, confText string) jmap {
	t.Helper()
	var c *Conf
	if confText != "" {
		var err error
		c, err = ParseConf(confText)
		if err != nil {
			t.Fatal(err)
		}
	}
	b, err := buildSingbox(mode, n, c, newRulesetCache())
	if err != nil {
		t.Fatalf("buildSingbox: %v", err)
	}
	return b.Config
}

func findOutbound(cfg jmap, tag string) jmap {
	for _, o := range cfg["outbounds"].([]jmap) {
		if o["tag"] == tag {
			return o
		}
	}
	return nil
}

// Регрессия на отсутствие nftables: auto_redirect/strict_route обязаны быть false.
func TestBuildTunNoNftables(t *testing.T) {
	cfg := buildFor(t, "config", nil, sampleConf)
	tun := cfg["inbounds"].([]jmap)[0]
	if tun["auto_redirect"] != false || tun["strict_route"] != false {
		t.Errorf("auto_redirect/strict_route must be false (kernel has no NF_TABLES): %+v", tun)
	}
	if tun["auto_route"] != true {
		t.Error("auto_route must be true")
	}
	if tun["iproute2_rule_index"] != TunRuleIndex || tun["iproute2_table_index"] != TunTable {
		t.Errorf("iproute2 indices = %v/%v", tun["iproute2_table_index"], tun["iproute2_rule_index"])
	}
	// Legacy-поля, удалённые в sing-box 1.13, не должны появляться.
	for _, k := range []string{"sniff", "sniff_override_destination", "domain_strategy"} {
		if _, ok := tun[k]; ok {
			t.Errorf("legacy inbound field %q present", k)
		}
	}
	// bypass-tun из [General] должен стать route_exclude_address.
	excl, ok := tun["route_exclude_address"].([]string)
	if !ok || len(excl) == 0 {
		t.Errorf("route_exclude_address missing: %+v", tun["route_exclude_address"])
	}
}

func TestBuildRoutingMarkOnlyOnAwg(t *testing.T) {
	vless, err := ParseURI("vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?security=tls&sni=a.com#v")
	if err != nil {
		t.Fatal(err)
	}
	cfg := buildFor(t, "config", &vless, sampleConf)
	for _, o := range cfg["outbounds"].([]jmap) {
		if _, ok := o["routing_mark"]; ok {
			t.Errorf("outbound %v must not carry routing_mark", o["tag"])
		}
	}

	awg := Node{Type: "awg", Name: "home", AWGConf: "home", Server: "vpn.example.com", Port: 51820}
	awg.setID()
	cfg = buildFor(t, "config", &awg, sampleConf)
	out := findOutbound(cfg, tagAwg)
	if out == nil {
		t.Fatal("awg-out outbound missing")
	}
	// bind_interface обязателен: без него auto_detect_interface привязывает
	// сокет к wlan0 и трафик обходит awg0 (проверено на устройстве).
	if out["routing_mark"] != AwgMark || out["type"] != "direct" ||
		out["bind_interface"] != AwgName {
		t.Errorf("awg outbound = %+v", out)
	}
	if findOutbound(cfg, tagProxy) != nil {
		t.Error("awg node must not produce a proxy outbound")
	}
	// PROXY-правила должны указывать на awg-out.
	found := false
	for _, r := range cfg["route"].(jmap)["rules"].([]jmap) {
		if r["outbound"] == tagAwg {
			found = true
		}
	}
	if !found {
		t.Error("no rule routes to awg-out")
	}
}

func TestBuildModes(t *testing.T) {
	n, err := ParseURI("socks5://10.0.0.1:1080#s")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("proxy sends everything to node", func(t *testing.T) {
		cfg := buildFor(t, "proxy", &n, sampleConf)
		if cfg["route"].(jmap)["final"] != tagProxy {
			t.Errorf("final = %v", cfg["route"].(jmap)["final"])
		}
		// В режиме proxy правила .conf игнорируются: остаются только sniff/hijack-dns.
		if got := len(cfg["route"].(jmap)["rules"].([]jmap)); got != 2 {
			t.Errorf("rules = %d, want 2", got)
		}
	})
	t.Run("direct ignores node", func(t *testing.T) {
		cfg := buildFor(t, "direct", &n, sampleConf)
		if cfg["route"].(jmap)["final"] != tagDirect {
			t.Errorf("final = %v", cfg["route"].(jmap)["final"])
		}
	})
	t.Run("config applies rules", func(t *testing.T) {
		cfg := buildFor(t, "config", &n, sampleConf)
		rules := cfg["route"].(jmap)["rules"].([]jmap)
		if len(rules) <= 2 {
			t.Fatalf("rules not applied: %+v", rules)
		}
		if rules[0]["action"] != "sniff" || rules[1]["action"] != "hijack-dns" {
			t.Errorf("first rules must be sniff+hijack-dns: %+v", rules[:2])
		}
		var sawReject bool
		for _, r := range rules {
			if r["action"] == "reject" {
				sawReject = true
			}
		}
		if !sawReject {
			t.Error("REJECT policy did not produce a reject action")
		}
	})
}

func TestBuildHostsAndDNS(t *testing.T) {
	cfg := buildFor(t, "config", nil, sampleConf)
	dns := cfg["dns"].(jmap)
	if dns["strategy"] != "ipv4_only" {
		t.Errorf("ipv6=false must force ipv4_only, got %v", dns["strategy"])
	}
	servers := dns["servers"].([]jmap)
	if servers[0]["type"] != "hosts" {
		t.Fatalf("[Host] must produce a hosts DNS server: %+v", servers[0])
	}
	pre := servers[0]["predefined"].(jmap)
	if got := pre["localhost"].([]string); len(got) != 1 || got[0] != "127.0.0.1" {
		t.Errorf("predefined localhost = %v", pre["localhost"])
	}
}

func TestBuildSkipProxyBeforeRules(t *testing.T) {
	n, _ := ParseURI("socks5://10.0.0.1:1080#s")
	cfg := buildFor(t, "config", &n, sampleConf)
	rules := cfg["route"].(jmap)["rules"].([]jmap)
	// skip-proxy должен стоять сразу после sniff/hijack-dns и вести в direct.
	third := rules[2]
	if third["outbound"] != tagDirect {
		t.Errorf("skip-proxy must come first and be direct: %+v", third)
	}
}

// Главная проверка: сгенерированный конфиг принимается настоящим sing-box.
func TestGeneratedConfigValidatesWithSingBox(t *testing.T) {
	bin := findSingBoxBinary()
	if bin == "" {
		t.Skip("sing-box binary for this host not found; run `make bins`")
	}
	nodes := map[string]string{
		"vless-reality": "vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?security=reality&sni=a.com&fp=chrome&pbk=AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8&sid=ab&type=ws&path=%2Fws#r",
		"vless-tls":     "vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?security=tls&sni=a.com&flow=xtls-rprx-vision#v",
		"vmess":         "vmess://" + b64std("auto:b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:8443") + "?obfs=websocket&path=/ws&tls=1&peer=s.com",
		"ss":            "ss://" + b64std("aes-256-gcm:pw") + "@1.2.3.4:8388#s",
		"trojan":        "trojan://pw@t.example.com:443?sni=s.com#t",
		"socks5":        "socks5://u:p@10.0.0.1:1080#s5",
		"http":          "http://u:p@proxy.example.com:8080#h",
		"ssh":           "ssh://root:pw@1.2.3.4:22#sh",
	}
	for name, uri := range nodes {
		t.Run(name, func(t *testing.T) {
			n, err := ParseURI(uri)
			if err != nil {
				t.Fatal(err)
			}
			cfg := buildFor(t, "config", &n, sampleConf)
			checkWithSingBox(t, bin, cfg)
		})
	}
	t.Run("awg", func(t *testing.T) {
		awg := Node{Type: "awg", Name: "home", AWGConf: "home"}
		awg.setID()
		cfg := buildFor(t, "config", &awg, sampleConf)
		checkWithSingBox(t, bin, cfg)
	})
	t.Run("no node", func(t *testing.T) {
		checkWithSingBox(t, bin, buildFor(t, "config", nil, sampleConf))
	})
}

// checkWithSingBox прогоняет конфиг через `sing-box check`.
// routing_mark/bind_interface поддерживаются только на Linux — на darwin снимаем.
func checkWithSingBox(t *testing.T, bin string, cfg jmap) {
	t.Helper()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var clone map[string]any
	if err := json.Unmarshal(raw, &clone); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" {
		for _, o := range clone["outbounds"].([]any) {
			delete(o.(map[string]any), "routing_mark")
			delete(o.(map[string]any), "bind_interface")
		}
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "singbox.json")
	b, _ := json.MarshalIndent(clone, "", " ")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(bin, "check", "-c", p).CombinedOutput()
	if err != nil {
		t.Fatalf("sing-box check failed: %v\n%s\nconfig:\n%s", err, out, b)
	}
}

func findSingBoxBinary() string {
	names := []string{"sing-box-darwin", "sing-box"}
	if runtime.GOOS != "darwin" {
		names = []string{"sing-box"}
	}
	for _, n := range names {
		p := filepath.Join("..", "vendor-bin", n)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
		}
	}
	return ""
}

func b64std(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
