package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// Штатный шаблон, поставляемый в click как default.conf.
// Он содержит две особенности, на которых парсер раньше молча терял правила:
// комментарии `//` и не-ASCII домен `.рф`.
const templateConf = `# Shadowrocket: 2026-08-30 18:20:34
[General]
bypass-system = true
skip-proxy = 127.0.0.1, 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, localhost, *.local, captive.apple.com, *.ru
bypass-tun = 10.0.0.0/8, 100.64.0.0/10, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.0.0.0/24, 192.0.2.0/24, 192.88.99.0/24, 192.168.0.0/16, 198.18.0.0/15, 198.51.100.0/24, 203.0.113.0/24, 224.0.0.0/4, 255.255.255.255/32
dns-server = system, 1.1.1.1, 8.8.8.8
update-url = https://raw.githubusercontent.com/vladdevops/shadowrocket-cfg/main/template-ios-ru-domain.conf interval=60 strict=true

[Rule]
DOMAIN-KEYWORD,rutracker,PROXY
DOMAIN-KEYWORD,binance,DIRECT
DOMAIN-KEYWORD,bnbstatic.com,DIRECT
DOMAIN-KEYWORD,yandex,DIRECT
DOMAIN-SUFFIX,.ru,DIRECT
DOMAIN-SUFFIX,.com.ru,DIRECT
DOMAIN-SUFFIX,.exnet.su,DIRECT
DOMAIN-SUFFIX,.ru.net,DIRECT
DOMAIN-SUFFIX,.pp.ru,DIRECT
DOMAIN-SUFFIX,.net.ru,DIRECT
DOMAIN-SUFFIX,.org.ru,DIRECT
DOMAIN-SUFFIX,.рф,DIRECT
GEOIP,RU,DIRECT
// Proxy Com

// KEYWORD

// Proxy Ru
FINAL,PROXY

[Host]
localhost = 127.0.0.1

[URL Rewrite]
^http://(www.)?yandex.ru https://www.ya.ru 302
`

func TestTemplateConfParses(t *testing.T) {
	c, err := ParseConf(templateConf)
	if err != nil {
		t.Fatal(err)
	}
	// 13 правил + FINAL. Строки `// ...` — комментарии, не правила.
	if len(c.Rules) != 14 {
		t.Fatalf("rules = %d, want 14: %+v", len(c.Rules), c.Rules)
	}
	if c.Final() != "PROXY" {
		t.Errorf("Final() = %q, want PROXY", c.Final())
	}
	// Комментарии `//` не должны попадать в Skipped как ошибки.
	for _, s := range c.Skipped {
		if strings.Contains(s, "Proxy Com") || strings.Contains(s, "KEYWORD") {
			t.Errorf("`//` comment treated as rule: %s", s)
		}
	}
	// Единственное ожидаемое предупреждение — [URL Rewrite].
	if len(c.Skipped) != 1 || !strings.Contains(strings.ToLower(c.Skipped[0]), "url rewrite") {
		t.Errorf("Skipped = %v, want only [URL Rewrite]", c.Skipped)
	}
	if c.General["dns-server"] != "system, 1.1.1.1, 8.8.8.8" {
		t.Errorf("dns-server = %q", c.General["dns-server"])
	}
}

// Регрессия: `.рф` обязан стать punycode, иначе правило никогда не сработает.
func TestTemplateIDNRuleBecomesPunycode(t *testing.T) {
	c, err := ParseConf(templateConf)
	if err != nil {
		t.Fatal(err)
	}
	n, err := ParseURI("socks5://127.0.0.1:1080#t")
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildSingbox("config", &n, c, newRulesetCache())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(b.Config["route"])
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "xn--p1ai") {
		t.Error("`.рф` did not reach the config as punycode")
	}
	if strings.Contains(s, "рф") {
		t.Error("raw non-ASCII domain leaked into the config")
	}
	// FINAL,PROXY должен стать финальным исходом.
	if b.Config["route"].(jmap)["final"] != tagProxy {
		t.Errorf("final = %v, want %s", b.Config["route"].(jmap)["final"], tagProxy)
	}
}

func TestTemplateBuildsValidForSingBox(t *testing.T) {
	bin := findSingBoxBinary()
	if bin == "" {
		t.Skip("sing-box binary not found; run `make bins`")
	}
	c, err := ParseConf(templateConf)
	if err != nil {
		t.Fatal(err)
	}
	for _, uri := range []string{
		"socks5://127.0.0.1:1080#t",
		"vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?security=tls&sni=a.com#v",
	} {
		n, err := ParseURI(uri)
		if err != nil {
			t.Fatal(err)
		}
		b, err := buildSingbox("config", &n, c, newRulesetCache())
		if err != nil {
			t.Fatal(err)
		}
		checkWithSingBox(t, bin, b.Config)
	}
	// AWG-узел на том же шаблоне: правила должны вести в awg-out.
	awg := Node{Type: "awg", Name: "home", AWGConf: "home"}
	awg.setID()
	b, err := buildSingbox("config", &awg, c, newRulesetCache())
	if err != nil {
		t.Fatal(err)
	}
	if b.Config["route"].(jmap)["final"] != tagAwg {
		t.Errorf("final = %v, want %s", b.Config["route"].(jmap)["final"], tagAwg)
	}
	checkWithSingBox(t, bin, b.Config)
}
