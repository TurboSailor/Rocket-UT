package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var subClient = &http.Client{Timeout: 20 * time.Second}

func isHTTPURL(s string) bool {
	low := strings.ToLower(s)
	return strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://")
}

// parseSubLink распознаёт ссылку-подписку, а не узел:
//
//	sub://<base64(url)>            — формат Shadowrocket «поделиться подпиской»
//	sub://<url>                    — часть клиентов не кодирует
//	shadowrocket://subscribe?url=… — вариант из deeplink Shadowrocket
//
// Возвращает реальный http(s)-адрес подписки.
func parseSubLink(s string) (string, bool) {
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)

	var payload string
	switch {
	case strings.HasPrefix(low, "sub://"):
		payload = s[len("sub://"):]
	case strings.HasPrefix(low, "shadowrocket://subscribe"):
		u, err := url.Parse(s)
		if err != nil {
			return "", false
		}
		payload = u.Query().Get("url")
	default:
		return "", false
	}

	// Хвост #remark к адресу подписки не относится.
	if i := strings.IndexByte(payload, '#'); i >= 0 {
		payload = payload[:i]
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", false
	}
	if isHTTPURL(payload) {
		return payload, true
	}
	if b, ok := b64decode(payload); ok {
		if d := strings.TrimSpace(string(b)); isHTTPURL(d) {
			return d, true
		}
	}
	return "", false
}

func httpGetText(rawurl string, limit int64) (string, error) {
	if !isHTTPURL(rawurl) {
		return "", fmt.Errorf("only http(s) urls allowed")
	}
	req, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Rocket/1.0")
	resp, err := subClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// parseSubBody разбирает тело подписки: base64-блок, список URI или .conf.
// Возвращает узлы и число нераспознанных строк.
func parseSubBody(body string) ([]Node, int) {
	text := strings.TrimSpace(body)
	if text == "" {
		return nil, 0
	}
	// .conf с секцией [Proxy] — берём узлы оттуда.
	if strings.HasPrefix(text, "[") || strings.Contains(text, "[Proxy]") {
		if c, err := ParseConf(text); err == nil && len(c.Proxies) > 0 {
			return c.Proxies, len(c.Skipped)
		}
	}
	// Целиком base64 — раскрываем.
	if !strings.Contains(text, "://") {
		if b, ok := b64decode(text); ok {
			text = string(b)
		}
	}
	var out []Node
	bad := 0
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		n, err := ParseURI(line)
		if err != nil {
			bad++
			continue
		}
		out = append(out, n)
	}
	return out, bad
}

// fetchSub скачивает подписку и обновляет её узлы в хранилище.
func fetchSub(st *Store, sub Sub) (int, error) {
	body, err := httpGetText(sub.URL, maxSubBody)
	if err != nil {
		st.UpdateSub(sub.ID, func(s *Sub) {
			s.Err = err.Error()
			s.UpdatedAt = time.Now().Unix()
		})
		return 0, err
	}
	nodes, bad := parseSubBody(body)
	if len(nodes) == 0 {
		e := fmt.Sprintf("no nodes parsed (%d unparsable lines)", bad)
		st.UpdateSub(sub.ID, func(s *Sub) {
			s.Err = e
			s.UpdatedAt = time.Now().Unix()
		})
		return 0, fmt.Errorf("%s", e)
	}
	st.ReplaceSubNodes(sub.ID, nodes)
	st.UpdateSub(sub.ID, func(s *Sub) {
		s.Count = len(nodes)
		s.UpdatedAt = time.Now().Unix()
		s.Err = ""
		if bad > 0 {
			s.Err = fmt.Sprintf("%d unparsable lines skipped", bad)
		}
	})
	return len(nodes), nil
}
