package main

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Записи журнала берутся из лога sing-box: опрос Clash API раз в секунду
// пропускает короткие соединения, а лог фиксирует каждое.
var (
	// sing-box запускается с --disable-color, но раскраска возможна при ручном
	// запуске бинаря, а ANSI-последовательности ломают разбор id соединения.
	reANSI = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

	reInbound = regexp.MustCompile(
		`\[(\d+) [^\]]*\] inbound/\S+: inbound (packet )?connection to (\S+)`)
	reOutbound = regexp.MustCompile(
		`\[(\d+) [^\]]*\] outbound/\w+\[([^\]]+)\]: outbound (packet )?connection to (\S+)`)
	reRejected = regexp.MustCompile(`\[(\d+) [^\]]*\].*(rejected|blocked)`)
)

// connLog — кольцевой лог соединений: источник для UI и для «добавить в правило».
type connLog struct {
	mu   sync.Mutex
	ring []ConnEntry
	byID map[string]int
	rx   int64
	tx   int64
	file *os.File
}

func newConnLog() *connLog {
	cl := &connLog{byID: map[string]int{}}
	cl.trimFile()
	f, err := os.OpenFile(connLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		logf("connlog open: %v", err)
	} else {
		cl.file = f
	}
	return cl
}

// trimFile обрезает файл до connLogKeep последних строк при старте.
func (cl *connLog) trimFile() {
	f, err := os.Open(connLogPath())
	if err != nil {
		return
	}
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		lines = append(lines, sc.Text())
		if len(lines) > connLogKeep*2 {
			lines = lines[len(lines)-connLogKeep:]
		}
	}
	f.Close()
	if len(lines) <= connLogKeep {
		return
	}
	lines = lines[len(lines)-connLogKeep:]
	if err := writeSecret(connLogPath(), strings.Join(lines, "\n")+"\n"); err != nil {
		logf("connlog trim: %v", err)
	}
}

func (cl *connLog) Traffic() (rx, tx int64) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	return cl.rx, cl.tx
}

// FromLog разбирает строку лога sing-box и обновляет журнал.
func (cl *connLog) FromLog(line string) {
	if strings.IndexByte(line, 0x1b) >= 0 {
		line = reANSI.ReplaceAllString(line, "")
	}
	if m := reInbound.FindStringSubmatch(line); m != nil {
		id, isUDP, addr := m[1], m[2] != "", m[3]
		host, port := splitAddr(addr)
		net_ := "tcp"
		if isUDP {
			net_ = "udp"
		}
		cl.upsert(id, func(e *ConnEntry) {
			e.Host, e.Port, e.Net = host, port, net_
			if ip := net.ParseIP(host); ip != nil {
				e.IP = host
			}
			e.Open = true
		})
		return
	}
	if m := reOutbound.FindStringSubmatch(line); m != nil {
		id, tag, addr := m[1], m[2], m[4]
		host, port := splitAddr(addr)
		cl.upsert(id, func(e *ConnEntry) {
			e.Policy = tag
			if e.Host == "" {
				e.Host, e.Port = host, port
			}
			if ip := net.ParseIP(host); ip != nil && e.IP == "" {
				e.IP = host
			}
		})
		return
	}
	if m := reRejected.FindStringSubmatch(line); m != nil {
		cl.upsert(m[1], func(e *ConnEntry) { e.Policy = "reject"; e.Open = false })
	}
}

func splitAddr(addr string) (string, int) {
	h, p, err := net.SplitHostPort(strings.TrimSuffix(addr, ":"))
	if err != nil {
		return addr, 0
	}
	port, _ := strconv.Atoi(p)
	return h, port
}

// upsert создаёт или обновляет запись по идентификатору соединения sing-box.
func (cl *connLog) upsert(id string, fn func(*ConnEntry)) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if i, ok := cl.byID[id]; ok && i < len(cl.ring) {
		fn(&cl.ring[i])
		return
	}
	e := ConnEntry{Time: time.Now().Unix(), Open: true}
	fn(&e)
	cl.ring = append(cl.ring, e)
	if len(cl.ring) > connRingCap {
		drop := len(cl.ring) - connRingCap
		for _, gone := range cl.ring[:drop] {
			cl.appendFileLocked(gone)
		}
		cl.ring = cl.ring[drop:]
		// Индексы сдвинулись — перестраиваем карту.
		for k, v := range cl.byID {
			if v-drop < 0 {
				delete(cl.byID, k)
				continue
			}
			cl.byID[k] = v - drop
		}
	}
	cl.byID[id] = len(cl.ring) - 1
}

// Ingest обновляет счётчики трафика и обогащает записи данными Clash API.
// Лог sing-box печатает только IP назначения, а domain приходит отсюда —
// без него «добавить в правило» предлагал бы лишь IP-CIDR.
func (cl *connLog) Ingest(c *clashConns) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.rx, cl.tx = c.DownloadTotal, c.UploadTotal
	type key struct {
		addr string
		port int
	}
	live := map[key]clashConn{}
	for _, cc := range c.Connections {
		port, _ := strconv.Atoi(cc.Metadata.DestinationPort)
		// Индексируем и по IP, и по домену: запись из лога знает только IP.
		if ip := cc.Metadata.DestinationIP; ip != "" {
			live[key{ip, port}] = cc
		}
		if h := cc.Metadata.Host; h != "" {
			live[key{h, port}] = cc
		}
	}
	for i := range cl.ring {
		e := &cl.ring[i]
		cc, ok := live[key{e.Host, e.Port}]
		if !ok && e.IP != "" {
			cc, ok = live[key{e.IP, e.Port}]
		}
		if !ok {
			continue
		}
		e.Up, e.Down = cc.Upload, cc.Download
		if e.Rule == "" && cc.Rule != "" {
			e.Rule = cc.Rule
			if cc.RulePayload != "" {
				e.Rule += " " + cc.RulePayload
			}
		}
		if e.IP == "" {
			e.IP = cc.Metadata.DestinationIP
		}
		// Домен полезнее IP: показываем его и предлагаем в правило.
		if h := cc.Metadata.Host; h != "" && net.ParseIP(e.Host) != nil {
			e.Host = h
		}
		if cc.Metadata.Network != "" {
			e.Net = cc.Metadata.Network
		}
	}
}

func (cl *connLog) appendFileLocked(e ConnEntry) {
	if cl.file == nil {
		return
	}
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	if _, err := cl.file.Write(append(b, '\n')); err != nil {
		logf("connlog write: %v", err)
	}
}

// Since отдаёт записи не старше ts (0 = все).
func (cl *connLog) Since(ts int64) []ConnEntry {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	out := make([]ConnEntry, 0, len(cl.ring))
	for _, e := range cl.ring {
		if e.Time >= ts {
			out = append(out, e)
		}
	}
	return out
}

func (cl *connLog) Reset() {
	cl.mu.Lock()
	for _, e := range cl.ring {
		cl.appendFileLocked(e)
	}
	cl.ring = nil
	cl.byID = map[string]int{}
	cl.rx, cl.tx = 0, 0
	cl.mu.Unlock()
}
