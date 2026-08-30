package main

import (
	"strings"
	"testing"
)

// Вывод снят с bundled zxing на устройстве: `text:` — последнее поле,
// после него идёт payload целиком, включая переводы строк.
const zxingAWGOutput = `text: [Interface]
PrivateKey = aGVsbG93b3JsZGhlbGxvd29ybGRoZWxsb3dvcmxkMTI=
Address = 10.8.0.2/32
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

// Регрессия: обрезка по первому '\n' превращала конфиг AmneziaWG в одну строку
// «[Interface]» и делала импорт по QR неработоспособным.
func TestParseZxingOutputMultiline(t *testing.T) {
	got, err := parseZxingOutput(zxingAWGOutput)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "[Interface]") {
		t.Errorf("payload starts with %q", got[:min(20, len(got))])
	}
	for _, need := range []string{
		"[Peer]", "PrivateKey = ", "PublicKey = ", "Endpoint = vpn.example.com:51820",
		"Jc = 4", "Jmin = 40", "Jmax = 70", "S1 = 50", "S2 = 100",
		"H1 = 1234567", "H4 = 4567890", "AllowedIPs = 0.0.0.0/0",
	} {
		if !strings.Contains(got, need) {
			t.Errorf("payload lost %q", need)
		}
	}
	if nl := strings.Count(got, "\n"); nl < 18 {
		t.Errorf("payload has %d newlines, want >= 18 (truncated?)", nl)
	}
	// Распознанное должно годиться как конфиг AmneziaWG.
	// ParseAWGConf вместо importText: последний пишет в /var/lib/rocketd.
	n, clean, err := ParseAWGConf("qrawg", got)
	if err != nil {
		t.Fatalf("ParseAWGConf: %v", err)
	}
	if n.Type != "awg" || n.AWGConf != "qrawg" {
		t.Errorf("node = %+v", n)
	}
	if n.Server != "vpn.example.com" || n.Port != 51820 {
		t.Errorf("endpoint lost: %+v", n)
	}
	// Обфускация обязана дожить до конфига интерфейса.
	setconf, addrs, mtu := awgConfParts(clean)
	if mtu != 1280 || len(addrs) != 1 || addrs[0] != "10.8.0.2/32" {
		t.Errorf("mtu=%d addrs=%v", mtu, addrs)
	}
	for _, k := range []string{"Jc = 4", "Jmin = 40", "Jmax = 70", "S1 = 50",
		"S2 = 100", "H1 = 1234567", "H4 = 4567890"} {
		if !strings.Contains(setconf, k) {
			t.Errorf("obfuscation %q lost after QR import", k)
		}
	}
}

func TestParseZxingOutputSingleLine(t *testing.T) {
	uri := "vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?type=ws#qrnode"
	got, err := parseZxingOutput("text: " + uri + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if got != uri {
		t.Errorf("got %q, want %q", got, uri)
	}
	if _, err := ParseURI(got); err != nil {
		t.Errorf("decoded URI does not parse: %v", err)
	}
}

func TestParseZxingOutputErrors(t *testing.T) {
	for _, out := range []string{
		"",
		"cannot load /home/phablet/Pictures/x.png",
		"text:",
		"text:   \n\n",
	} {
		t.Run(out, func(t *testing.T) {
			if _, err := parseZxingOutput(out); err == nil {
				t.Errorf("expected error for %q", out)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
