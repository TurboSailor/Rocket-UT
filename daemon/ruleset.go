package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const rulesetTTL = 24 * time.Hour

// rulesetCache раскрывает RULE-SET в набор обычных правил с кэшем на диске.
type rulesetCache struct {
	client *http.Client
}

func newRulesetCache() *rulesetCache {
	return &rulesetCache{client: &http.Client{Timeout: 30 * time.Second}}
}

func rulesetFile(url string) string {
	h := sha1.Sum([]byte(url))
	return filepath.Join(rulesetDir(), hex.EncodeToString(h[:])+".list")
}

// expand возвращает правила из RULE-SET с политикой исходного правила.
func (rc *rulesetCache) expand(r Rule) ([]Rule, error) {
	arg := strings.TrimSpace(r.Arg)
	switch strings.ToUpper(arg) {
	case "SYSTEM", "LAN":
		// Приватные адреса — единственный вменяемый эквивалент встроенных наборов.
		return []Rule{{Type: "IP-CIDR", Arg: "10.0.0.0/8", Policy: r.Policy},
			{Type: "IP-CIDR", Arg: "172.16.0.0/12", Policy: r.Policy},
			{Type: "IP-CIDR", Arg: "192.168.0.0/16", Policy: r.Policy},
			{Type: "IP-CIDR", Arg: "127.0.0.0/8", Policy: r.Policy}}, nil
	}
	if !strings.HasPrefix(arg, "http://") && !strings.HasPrefix(arg, "https://") {
		return nil, fmt.Errorf("unsupported rule-set %q", arg)
	}
	text, err := rc.fetch(arg)
	if err != nil {
		return nil, err
	}
	var out []Rule
	for _, raw := range strings.Split(text, "\n") {
		line := stripComment(raw)
		if line == "" {
			continue
		}
		f := splitCSV(line)
		if len(f) >= 2 {
			typ := strings.ToUpper(f[0])
			if !ruleTypeOK[typ] || typ == "RULE-SET" || typ == "FINAL" {
				continue
			}
			// В .list-файлах политики нет — берём из исходного RULE-SET.
			out = append(out, Rule{Type: typ, Arg: f[1], Policy: r.Policy})
			continue
		}
		// Публичные списки часто состоят из «голых» строк: `.apple.com`,
		// `example.com`, `1.2.3.4`, `10.0.0.0/8` — без типа правила.
		if br, ok := bareRule(f[0], r.Policy); ok {
			out = append(out, br)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no usable rules")
	}
	return out, nil
}

// bareRule распознаёт строку списка без префикса типа.
func bareRule(s, policy string) (Rule, bool) {
	s = cleanArg(s)
	if s == "" || strings.ContainsAny(s, " \t,") {
		return Rule{}, false
	}
	if strings.Contains(s, "/") {
		if _, _, err := net.ParseCIDR(s); err == nil {
			return Rule{Type: "IP-CIDR", Arg: s, Policy: policy}, true
		}
		return Rule{}, false
	}
	if ip := net.ParseIP(s); ip != nil {
		suffix := "/32"
		if ip.To4() == nil {
			suffix = "/128"
		}
		return Rule{Type: "IP-CIDR", Arg: s + suffix, Policy: policy}, true
	}
	// Ведущая точка у Shadowrocket означает совпадение по суффиксу.
	if strings.HasPrefix(s, ".") {
		s = s[1:]
	}
	if s == "" || !strings.Contains(s, ".") || strings.HasPrefix(s, "-") {
		return Rule{}, false
	}
	return Rule{Type: "DOMAIN-SUFFIX", Arg: s, Policy: policy}, true
}

// fetch отдаёт содержимое набора: свежий кэш, иначе сеть, иначе устаревший кэш.
func (rc *rulesetCache) fetch(url string) (string, error) {
	path := rulesetFile(url)
	if st, err := os.Stat(path); err == nil && time.Since(st.ModTime()) < rulesetTTL {
		if b, err := os.ReadFile(path); err == nil {
			return string(b), nil
		}
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Rocket/1.0")
	resp, err := rc.client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			b, rerr := io.ReadAll(io.LimitReader(resp.Body, maxSubBody))
			if rerr == nil {
				if werr := writeSecret(path, string(b)); werr != nil {
					logf("ruleset cache write: %v", werr)
				}
				return string(b), nil
			}
			err = rerr
		} else {
			err = fmt.Errorf("http %d", resp.StatusCode)
		}
	}
	// Сеть недоступна — используем устаревший кэш, если он есть.
	if b, cerr := os.ReadFile(path); cerr == nil {
		logf("ruleset %s: using stale cache (%v)", url, err)
		return string(b), nil
	}
	return "", err
}
