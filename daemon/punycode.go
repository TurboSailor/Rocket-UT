package main

import "strings"

// Punycode (RFC 3492) для IDN-доменов вроде `.рф`.
// sing-box сопоставляет уже закодированные имена, которые приходят из DNS/SNI,
// поэтому правило `DOMAIN-SUFFIX,.рф` без преобразования не сработало бы никогда.
// Своя реализация вместо golang.org/x/net/idna: демон держим на stdlib.

const (
	punyBase        = 36
	punyTMin        = 1
	punyTMax        = 26
	punySkew        = 38
	punyDamp        = 700
	punyInitialBias = 72
	punyInitialN    = 128
	punyDelimiter   = '-'
)

func punyDigit(d int) byte {
	if d < 26 {
		return byte('a' + d)
	}
	return byte('0' + d - 26)
}

func punyAdapt(delta, numPoints int, firstTime bool) int {
	if firstTime {
		delta /= punyDamp
	} else {
		delta /= 2
	}
	delta += delta / numPoints
	k := 0
	for delta > ((punyBase-punyTMin)*punyTMax)/2 {
		delta /= punyBase - punyTMin
		k += punyBase
	}
	return k + (punyBase-punyTMin+1)*delta/(delta+punySkew)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// punyLabel кодирует одну метку домена. Возвращает false, если кодирование не нужно.
func punyLabel(label string) (string, bool) {
	if label == "" || isASCII(label) {
		return label, false
	}
	runes := []rune(label)
	var out strings.Builder
	out.WriteString("xn--")

	basic := 0
	for _, r := range runes {
		if r < punyInitialN {
			out.WriteRune(r)
			basic++
		}
	}
	if basic > 0 {
		out.WriteByte(punyDelimiter)
	}

	n, bias, delta := punyInitialN, punyInitialBias, 0
	handled := basic
	for handled < len(runes) {
		// Наименьший кодпоинт >= n среди оставшихся.
		m := 0x7fffffff
		for _, r := range runes {
			if int(r) >= n && int(r) < m {
				m = int(r)
			}
		}
		if m == 0x7fffffff {
			return label, false
		}
		delta += (m - n) * (handled + 1)
		n = m
		for _, r := range runes {
			c := int(r)
			switch {
			case c < n:
				delta++
			case c == n:
				q := delta
				for k := punyBase; ; k += punyBase {
					t := k - bias
					if t < punyTMin {
						t = punyTMin
					} else if t > punyTMax {
						t = punyTMax
					}
					if q < t {
						break
					}
					out.WriteByte(punyDigit(t + (q-t)%(punyBase-t)))
					q = (q - t) / (punyBase - t)
				}
				out.WriteByte(punyDigit(q))
				bias = punyAdapt(delta, handled+1, handled == basic)
				delta = 0
				handled++
			}
		}
		delta++
		n++
	}
	return out.String(), true
}

// punyDomain кодирует все не-ASCII метки, сохраняя ведущую точку суффикса.
func punyDomain(s string) string {
	if isASCII(s) {
		return s
	}
	lead := ""
	if strings.HasPrefix(s, ".") {
		lead, s = ".", s[1:]
	}
	parts := strings.Split(s, ".")
	for i, p := range parts {
		if enc, ok := punyLabel(p); ok {
			parts[i] = enc
		}
	}
	return lead + strings.Join(parts, ".")
}
