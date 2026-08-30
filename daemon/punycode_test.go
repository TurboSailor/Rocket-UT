package main

import "testing"

func TestPunyDomain(t *testing.T) {
	tests := []struct{ in, want string }{
		// Векторы проверены против общеизвестных значений IDN.
		{".рф", ".xn--p1ai"},
		{"рф", "xn--p1ai"},
		{"яндекс.рф", "xn--d1acpjx3f.xn--p1ai"},
		{"почта.рф", "xn--80a1acny.xn--p1ai"},
		{"münchen.de", "xn--mnchen-3ya.de"},
		{"example.com", "example.com"},
		{".ru", ".ru"},
		{"", ""},
		// Смешанные метки: ASCII-часть не трогаем.
		{"shop.яндекс.ru", "shop.xn--d1acpjx3f.ru"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := punyDomain(tc.in); got != tc.want {
				t.Errorf("punyDomain(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestPunyDomainIdempotent(t *testing.T) {
	for _, s := range []string{".xn--p1ai", "xn--d1acpjx3f.xn--p1ai", "example.com"} {
		if got := punyDomain(s); got != s {
			t.Errorf("punyDomain(%q) = %q, want unchanged", s, got)
		}
	}
}
