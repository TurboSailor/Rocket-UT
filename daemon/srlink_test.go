package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

// Shadowrocket упаковывает `user:pass@host:port` в base64 внутри socks:// и http://.
// Раньше парсер принимал такую ссылку молча: в Server попадал сам base64,
// креды оставались пустыми, и соединение не работало. Креды здесь синтетические.
func TestParseSocksHTTPEncodedAuthority(t *testing.T) {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

	t.Run("socks with encoded credentials", func(t *testing.T) {
		link := "socks://" + enc("e8221ebeec4a3d18:99b4209ea6e7e27fa6632a7a7c7d31d1@203.0.113.7:1080") +
			"?udp=1&method=auto"
		n, err := ParseURI(link)
		if err != nil {
			t.Fatal(err)
		}
		if n.Type != "socks5" {
			t.Errorf("type = %q", n.Type)
		}
		if n.Server != "203.0.113.7" || n.Port != 1080 {
			t.Errorf("addr = %s:%d, want 203.0.113.7:1080", n.Server, n.Port)
		}
		if n.User != "e8221ebeec4a3d18" || n.Password != "99b4209ea6e7e27fa6632a7a7c7d31d1" {
			t.Errorf("credentials lost: user=%q pass=%q", n.User, n.Password)
		}
		// Регрессия: base64 не должен оказаться именем хоста.
		if strings.Contains(n.Server, "=") || len(n.Server) > 40 {
			t.Errorf("base64 leaked into Server: %q", n.Server)
		}
	})

	t.Run("http with encoded credentials and tfo", func(t *testing.T) {
		link := "http://" + strings.TrimRight(enc("v3j7vhuXo5:rKloFMEHDZ@proxy.example.com:26326"), "=") +
			"?method=connect&tfo=1&chain=622D590A-AF8B-4156-A997-7B046A854176"
		n, err := ParseURI(link)
		if err != nil {
			t.Fatal(err)
		}
		if n.Type != "http" {
			t.Errorf("type = %q", n.Type)
		}
		if n.Server != "proxy.example.com" || n.Port != 26326 {
			t.Errorf("addr = %s:%d", n.Server, n.Port)
		}
		if n.User != "v3j7vhuXo5" || n.Password != "rKloFMEHDZ" {
			t.Errorf("credentials lost: user=%q pass=%q", n.User, n.Password)
		}
		if !n.TFO {
			t.Error("tfo=1 was ignored")
		}
		if n.TLS {
			t.Error("plain http must not enable TLS")
		}
	})

	t.Run("socks5h encoded", func(t *testing.T) {
		n, err := ParseURI("socks5h://" + enc("u:p@198.51.100.9:9050"))
		if err != nil {
			t.Fatal(err)
		}
		if n.Server != "198.51.100.9" || n.Port != 9050 || n.User != "u" {
			t.Errorf("node = %+v", n)
		}
	})
}

// Обычные, не закодированные ссылки должны продолжать работать как раньше:
// декодирование включается только для «голого» токена.
func TestParseSocksHTTPPlainStillWorks(t *testing.T) {
	tests := []struct {
		uri            string
		server         string
		port           int
		user, password string
		tls            bool
	}{
		{"socks5://10.0.0.1:1080", "10.0.0.1", 1080, "", "", false},
		{"socks5://u:p@10.0.0.1:1080#x", "10.0.0.1", 1080, "u", "p", false},
		{"socks5://localhost:1080", "localhost", 1080, "", "", false},
		{"socks5://localhost", "localhost", 1080, "", "", false},
		{"http://u:p@proxy.example.com:8080", "proxy.example.com", 8080, "u", "p", false},
		{"http://proxy.example.com", "proxy.example.com", 80, "", "", false},
		{"https://proxy.example.com", "proxy.example.com", 443, "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.uri, func(t *testing.T) {
			n, err := ParseURI(tc.uri)
			if err != nil {
				t.Fatal(err)
			}
			if n.Server != tc.server || n.Port != tc.port ||
				n.User != tc.user || n.Password != tc.password || n.TLS != tc.tls {
				t.Errorf("node = {server=%q port=%d user=%q pass=%q tls=%v}, want {%q %d %q %q %v}",
					n.Server, n.Port, n.User, n.Password, n.TLS,
					tc.server, tc.port, tc.user, tc.password, tc.tls)
			}
		})
	}
}

func TestDecodeSRAuthority(t *testing.T) {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	ok := []struct{ in, want string }{
		{enc("u:p@1.2.3.4:1080"), "u:p@1.2.3.4:1080"},
		{enc("host.example.com:443"), "host.example.com:443"},
		{strings.TrimRight(enc("u:p@1.2.3.4:1080"), "="), "u:p@1.2.3.4:1080"},
	}
	for _, tc := range ok {
		if got, isEnc := decodeSRAuthority(tc.in); !isEnc || got != tc.want {
			t.Errorf("decodeSRAuthority(%q) = (%q,%v), want (%q,true)", tc.in, got, isEnc, tc.want)
		}
	}
	// Обычные authority и мусор трогать нельзя.
	for _, s := range []string{
		"", "proxy.example.com", "proxy.example.com:8080", "u:p@1.2.3.4:1080",
		"localhost", "10.0.0.1", enc("not-an-address"), enc("host-without-port"),
	} {
		if got, isEnc := decodeSRAuthority(s); isEnc {
			t.Errorf("decodeSRAuthority(%q) wrongly decoded to %q", s, got)
		}
	}
}

// Сгенерированный конфиг с такими узлами должен приниматься самим sing-box.
func TestEncodedLinksBuildValidConfig(t *testing.T) {
	bin := findSingBoxBinary()
	if bin == "" {
		t.Skip("sing-box binary not found; run `make bins`")
	}
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	conf, err := ParseConf(templateConf)
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range []string{
		"socks://" + enc("u:p@203.0.113.7:1080") + "?udp=1&method=auto",
		"http://" + enc("u:p@proxy.example.com:26326") + "?method=connect&tfo=1",
	} {
		n, err := ParseURI(link)
		if err != nil {
			t.Fatal(err)
		}
		b, err := buildSingbox("config", &n, conf, newRulesetCache())
		if err != nil {
			t.Fatal(err)
		}
		checkWithSingBox(t, bin, b.Config)
	}
}
