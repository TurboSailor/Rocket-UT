package main

import (
	"encoding/base64"
	"testing"
)

// Реальная ссылка из Shadowrocket, на которой импорт падал с «scheme sub not supported».
const realSubLink = "sub://aHR0cHM6Ly9zdWJzLmJpbm9tLmRldi9hOF9ldW1tWW5vM0tmejRv"

func TestParseSubLink(t *testing.T) {
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	raw := base64.RawURLEncoding.EncodeToString([]byte("https://example.com/s/tok"))

	ok := []struct {
		in   string
		want string
	}{
		{realSubLink, "https://subs.binom.dev/a8_eummYno3Kfz4o"},
		{"sub://" + b64("https://example.com/sub?token=1"), "https://example.com/sub?token=1"},
		// base64 без паддинга и url-safe алфавит
		{"sub://" + raw, "https://example.com/s/tok"},
		// часть клиентов не кодирует адрес
		{"sub://https://example.com/plain", "https://example.com/plain"},
		// хвост #remark к адресу не относится
		{"sub://" + b64("https://example.com/sub") + "#My%20sub", "https://example.com/sub"},
		{"SUB://" + b64("http://example.com/sub"), "http://example.com/sub"},
		{"shadowrocket://subscribe?url=" + b64("https://example.com/sr"), "https://example.com/sr"},
		{"shadowrocket://subscribe?url=https%3A%2F%2Fexample.com%2Fenc", "https://example.com/enc"},
	}
	for _, tc := range ok {
		t.Run(tc.in, func(t *testing.T) {
			got, isSub := parseSubLink(tc.in)
			if !isSub {
				t.Fatalf("not recognised as a subscription link")
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}

	bad := []string{
		"",
		"sub://",
		"sub://###",
		// base64, который декодируется не в http(s)
		"sub://" + b64("vless://uuid@1.2.3.4:443"),
		"https://example.com/sub",
		"vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443",
		"shadowrocket://subscribe",
		"shadowrocket://subscribe?url=",
	}
	for _, s := range bad {
		t.Run("bad:"+s, func(t *testing.T) {
			if got, isSub := parseSubLink(s); isSub {
				t.Errorf("%q wrongly treated as a subscription -> %q", s, got)
			}
		})
	}
}

// Ссылка-подписка не должна выглядеть как узел ни для ParseURI, ни для parseSubBody:
// иначе она молча попала бы в список узлов мусором.
func TestSubLinkIsNotANode(t *testing.T) {
	if _, err := ParseURI(realSubLink); err == nil {
		t.Error("ParseURI accepted a subscription link as a node")
	}
	nodes, bad := parseSubBody(realSubLink)
	if len(nodes) != 0 {
		t.Errorf("parseSubBody produced %d nodes from a sub link", len(nodes))
	}
	if bad == 0 {
		t.Error("sub link should be counted as unparsable by parseSubBody")
	}
}

// Тело подписки бывает и списком ссылок, и .conf — проверяем, что раскрытый
// адрес затем обрабатывается обычным путём.
func TestSubLinkResolvedThenParsed(t *testing.T) {
	link, ok := parseSubLink(realSubLink)
	if !ok {
		t.Fatal("link not resolved")
	}
	if !isHTTPURL(link) {
		t.Fatalf("resolved link is not http(s): %q", link)
	}
	body := base64.StdEncoding.EncodeToString(
		[]byte("socks5://10.0.0.1:1080#a\nvless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?type=tcp#b\n"))
	nodes, badLines := parseSubBody(body)
	if len(nodes) != 2 || badLines != 0 {
		t.Fatalf("nodes=%d bad=%d", len(nodes), badLines)
	}
}
