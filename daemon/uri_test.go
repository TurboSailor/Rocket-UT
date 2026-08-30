package main

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func TestParseURIVless(t *testing.T) {
	n, err := ParseURI("vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443" +
		"?encryption=none&security=reality&sni=a.com&fp=chrome&pbk=PUBKEY&sid=ab" +
		"&type=ws&path=%2Fws&host=h.com&flow=xtls-rprx-vision#my node")
	if err != nil {
		t.Fatal(err)
	}
	if n.Type != "vless" || n.Server != "1.2.3.4" || n.Port != 443 {
		t.Errorf("addr mismatch: %+v", n)
	}
	if n.UUID != "b831381d-6324-4d53-ad4f-8cda48b30811" {
		t.Errorf("uuid = %q", n.UUID)
	}
	if !n.TLS || n.Reality.PublicKey != "PUBKEY" || n.Reality.ShortID != "ab" {
		t.Errorf("reality mismatch: %+v", n)
	}
	if n.Network != "ws" || n.Path != "/ws" || n.Host != "h.com" {
		t.Errorf("transport mismatch: %+v", n)
	}
	if n.SNI != "a.com" || n.FP != "chrome" || n.Flow != "xtls-rprx-vision" {
		t.Errorf("tls params mismatch: %+v", n)
	}
	if n.Name != "my node" {
		t.Errorf("name = %q", n.Name)
	}
}

func TestParseURIVmessJSON(t *testing.T) {
	j := `{"v":"2","ps":"jsonnode","add":"1.2.3.4","port":"443",
	       "id":"b831381d-6324-4d53-ad4f-8cda48b30811","aid":"4","scy":"auto",
	       "net":"ws","host":"h.com","path":"/p","tls":"tls","sni":"s.com"}`
	n, err := ParseURI("vmess://" + base64.StdEncoding.EncodeToString([]byte(j)))
	if err != nil {
		t.Fatal(err)
	}
	if n.Type != "vmess" || n.Server != "1.2.3.4" || n.Port != 443 || n.AlterID != 4 {
		t.Errorf("mismatch: %+v", n)
	}
	if n.Network != "ws" || n.Path != "/p" || n.Host != "h.com" || !n.TLS || n.SNI != "s.com" {
		t.Errorf("transport/tls mismatch: %+v", n)
	}
	if n.Name != "jsonnode" {
		t.Errorf("name = %q", n.Name)
	}
}

func TestParseURIVmessShadowrocket(t *testing.T) {
	core := base64.StdEncoding.EncodeToString(
		[]byte("auto:b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:8443"))
	n, err := ParseURI("vmess://" + core +
		"?remarks=srnode&obfs=websocket&path=/ws&obfsParam=h.com&tls=1&peer=s.com&alterId=2")
	if err != nil {
		t.Fatal(err)
	}
	if n.Type != "vmess" || n.Server != "1.2.3.4" || n.Port != 8443 {
		t.Errorf("addr mismatch: %+v", n)
	}
	if n.UUID != "b831381d-6324-4d53-ad4f-8cda48b30811" || n.Method != "auto" {
		t.Errorf("cred mismatch: %+v", n)
	}
	if n.Network != "ws" || n.Path != "/ws" || n.Host != "h.com" || !n.TLS ||
		n.SNI != "s.com" || n.AlterID != 2 {
		t.Errorf("params mismatch: %+v", n)
	}
	if n.Name != "srnode" {
		t.Errorf("name = %q", n.Name)
	}
}

// Регрессия: подписки встречаются с vmess в виде обычного URL, без base64.
func TestParseURIVmessPlainURL(t *testing.T) {
	n, err := ParseURI("vmess://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:8444" +
		"?type=ws&path=%2Fw&security=tls&sni=s.com&alterId=3#plain")
	if err != nil {
		t.Fatal(err)
	}
	if n.Type != "vmess" || n.Server != "1.2.3.4" || n.Port != 8444 {
		t.Errorf("addr mismatch: %+v", n)
	}
	if n.UUID != "b831381d-6324-4d53-ad4f-8cda48b30811" || n.AlterID != 3 {
		t.Errorf("cred mismatch: %+v", n)
	}
	if n.Network != "ws" || n.Path != "/w" || !n.TLS || n.SNI != "s.com" {
		t.Errorf("transport mismatch: %+v", n)
	}
	if n.Name != "plain" {
		t.Errorf("name = %q", n.Name)
	}
}
func TestParseURISS(t *testing.T) {
	t.Run("sip002", func(t *testing.T) {
		cred := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:pw"))
		n, err := ParseURI("ss://" + cred + "@1.2.3.4:8388#ssnode")
		if err != nil {
			t.Fatal(err)
		}
		if n.Method != "aes-256-gcm" || n.Password != "pw" || n.Port != 8388 {
			t.Errorf("mismatch: %+v", n)
		}
	})
	t.Run("legacy", func(t *testing.T) {
		all := base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:pw2@5.6.7.8:1234"))
		n, err := ParseURI("ss://" + all + "#legacy")
		if err != nil {
			t.Fatal(err)
		}
		if n.Method != "aes-128-gcm" || n.Password != "pw2" || n.Server != "5.6.7.8" || n.Port != 1234 {
			t.Errorf("mismatch: %+v", n)
		}
	})
	t.Run("plugin rejected", func(t *testing.T) {
		cred := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:pw"))
		if _, err := ParseURI("ss://" + cred + "@1.2.3.4:8388?plugin=obfs-local#x"); err == nil {
			t.Error("expected plugin to be rejected")
		}
	})
}

func TestParseURIOthers(t *testing.T) {
	tests := []struct {
		name  string
		uri   string
		check func(Node) error
	}{
		{"trojan", "trojan://pw@t.example.com:443?sni=s.com&fp=chrome#tj", func(n Node) error {
			if n.Type != "trojan" || n.Password != "pw" || !n.TLS || n.SNI != "s.com" {
				return errf("mismatch: %+v", n)
			}
			return nil
		}},
		{"socks5", "socks5://u:p@10.0.0.1:1080#s5", func(n Node) error {
			if n.Type != "socks5" || n.User != "u" || n.Password != "p" || n.Port != 1080 {
				return errf("mismatch: %+v", n)
			}
			return nil
		}},
		{"socks5 no auth", "socks5://10.0.0.1:1080", func(n Node) error {
			if n.Type != "socks5" || n.User != "" || n.Port != 1080 {
				return errf("mismatch: %+v", n)
			}
			return nil
		}},
		{"socks default port", "socks5://10.0.0.1", func(n Node) error {
			if n.Port != 1080 {
				return errf("port = %d, want 1080", n.Port)
			}
			return nil
		}},
		{"http", "http://u:p@proxy.example.com:8080#h", func(n Node) error {
			if n.Type != "http" || n.Port != 8080 || n.User != "u" || n.TLS {
				return errf("mismatch: %+v", n)
			}
			return nil
		}},
		{"https", "https://proxy.example.com", func(n Node) error {
			if n.Type != "http" || n.Port != 443 || !n.TLS {
				return errf("mismatch: %+v", n)
			}
			return nil
		}},
		{"ssh password", "ssh://root:pw@1.2.3.4:2222#sh", func(n Node) error {
			if n.Type != "ssh" || n.User != "root" || n.Password != "pw" || n.Port != 2222 {
				return errf("mismatch: %+v", n)
			}
			return nil
		}},
		{"ssh default port", "ssh://root:pw@1.2.3.4", func(n Node) error {
			if n.Port != 22 {
				return errf("port = %d, want 22", n.Port)
			}
			return nil
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, err := ParseURI(tc.uri)
			if err != nil {
				t.Fatal(err)
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

func TestParseURISSHPrivateKey(t *testing.T) {
	pem := "-----BEGIN OPENSSH PRIVATE KEY-----\nabc+def/==\n-----END OPENSSH PRIVATE KEY-----\n"
	n, err := ParseURI("ssh://root@1.2.3.4:22?privateKey=" + url.QueryEscape(pem) + "#k")
	if err != nil {
		t.Fatal(err)
	}
	if n.PrivKey != pem {
		t.Errorf("private key roundtrip failed:\n%q", n.PrivKey)
	}
}

func TestParseURIErrors(t *testing.T) {
	bad := []string{
		"", "not a uri", "ftp://host:21", "vless://1.2.3.4:443",
		"vmess://%%%", "ss://@@@", "trojan://host", "ssh://1.2.3.4:22",
		"vless://uuid@1.2.3.4:99999", "socks5://u:p@host:abc",
	}
	for _, s := range bad {
		t.Run(s, func(t *testing.T) {
			if _, err := ParseURI(s); err == nil {
				t.Errorf("expected error for %q", s)
			}
		})
	}
}

const awgSample = `[Interface]
PrivateKey = aGVsbG93b3JsZGhlbGxvd29ybGRoZWxsb3dvcmxkMTI=
Address = 10.8.0.2/32
DNS = 1.1.1.1
MTU = 1280
Jc = 4
Jmin = 40
Jmax = 70
S1 = 50
S2 = 100
H1 = 1234567
H2 = 2345678
H3 = 3456789
H4 = 4567890

[Peer]
PublicKey = cHVibGlja2V5cHVibGlja2V5cHVibGlja2V5MTIzNA=
AllowedIPs = 0.0.0.0/0
Endpoint = vpn.example.com:51820
PersistentKeepalive = 25
`

func TestParseAWGConf(t *testing.T) {
	n, clean, err := ParseAWGConf("home", awgSample)
	if err != nil {
		t.Fatal(err)
	}
	if n.Type != "awg" || n.AWGConf != "home" {
		t.Errorf("node = %+v", n)
	}
	if n.Server != "vpn.example.com" || n.Port != 51820 {
		t.Errorf("endpoint mismatch: %+v", n)
	}
	// DNS выкидывается: в UT нет resolvconf.
	if strings.Contains(clean, "DNS") {
		t.Error("DNS line survived sanitization")
	}
	// Обфускация AmneziaWG обязана сохраниться.
	for _, k := range []string{"Jc", "Jmin", "Jmax", "S1", "S2", "H1", "H4"} {
		if !strings.Contains(clean, k+" =") {
			t.Errorf("obfuscation key %s lost", k)
		}
	}
}

func TestParseAWGConfInvalid(t *testing.T) {
	if _, _, err := ParseAWGConf("x", "[Interface]\nPrivateKey = a\n"); err == nil {
		t.Error("expected error without [Peer]/PublicKey")
	}
}

func TestAwgConfParts(t *testing.T) {
	setconf, addrs, mtu := awgConfParts(awgSample)
	if mtu != 1280 {
		t.Errorf("mtu = %d, want 1280", mtu)
	}
	if len(addrs) != 1 || addrs[0] != "10.8.0.2/32" {
		t.Errorf("addrs = %v", addrs)
	}
	// awg setconf не понимает Address/MTU/DNS — их не должно быть в uapi-конфиге.
	for _, k := range []string{"Address", "MTU", "DNS"} {
		if strings.Contains(setconf, k+" =") {
			t.Errorf("%s must not reach awg setconf:\n%s", k, setconf)
		}
	}
	// Марка обязательна: без неё UDP-сокет уйдёт в туннель и получится петля.
	if !strings.Contains(setconf, "FwMark = "+markHex(EscapeMark)) {
		t.Errorf("FwMark missing:\n%s", setconf)
	}
	if !strings.Contains(setconf, "Jc = 4") || !strings.Contains(setconf, "Endpoint =") {
		t.Errorf("setconf lost required keys:\n%s", setconf)
	}
}

func TestParseSubBody(t *testing.T) {
	t.Run("base64 list", func(t *testing.T) {
		list := "socks5://10.0.0.1:1080#a\nssh://root:pw@1.2.3.4:22#b\n"
		nodes, bad := parseSubBody(base64.StdEncoding.EncodeToString([]byte(list)))
		if len(nodes) != 2 || bad != 0 {
			t.Fatalf("nodes = %d, bad = %d", len(nodes), bad)
		}
	})
	t.Run("plain list with junk", func(t *testing.T) {
		nodes, bad := parseSubBody("socks5://10.0.0.1:1080\n# comment\ngarbage\n")
		if len(nodes) != 1 {
			t.Errorf("nodes = %d, want 1", len(nodes))
		}
		if bad != 1 {
			t.Errorf("bad = %d, want 1", bad)
		}
	})
	t.Run("conf body", func(t *testing.T) {
		nodes, _ := parseSubBody(sampleConf)
		if len(nodes) != 7 {
			t.Errorf("nodes = %d, want 7", len(nodes))
		}
	})
}
