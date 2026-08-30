package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Engine управляет стеком: sing-box + при необходимости amneziawg-go и policy routing.
type Engine struct {
	mu      sync.Mutex
	st      *Store
	sb      *proc
	awg     *proc
	clash   *clashClient
	log     *connLog
	rs      *rulesetCache
	skipped []string
}

func NewEngine(st *Store) *Engine {
	return &Engine{
		st:    st,
		sb:    &proc{name: "sing-box"},
		awg:   &proc{name: "amneziawg-go"},
		clash: newClashClient(),
		log:   newConnLog(),
		rs:    newRulesetCache(),
	}
}

// activeConf читает и разбирает активный .conf.
func (e *Engine) activeConf() (*Conf, error) {
	st := e.st.State()
	if st.ConfName == "" {
		return ParseConf(defaultConfText())
	}
	text, err := readConf(st.ConfName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ParseConf(defaultConfText())
		}
		return nil, err
	}
	return ParseConf(text)
}

// Skipped отдаёт предупреждения последней генерации конфига.
func (e *Engine) Skipped() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]string, len(e.skipped))
	copy(out, e.skipped)
	return out
}

// generate строит singbox.json из текущего состояния и пишет его атомарно.
func (e *Engine) generate() error {
	st := e.st.State()
	var active *Node
	if st.NodeID != "" {
		if n, ok := e.st.Node(st.NodeID); ok {
			active = &n
		}
	}
	conf, err := e.activeConf()
	if err != nil {
		return fmt.Errorf("conf: %w", err)
	}
	b, err := buildSingbox(st.Mode, active, conf, e.rs)
	if err != nil {
		return err
	}
	text, err := marshalConfig(b.Config)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.skipped = append(append([]string{}, conf.Skipped...), b.Skipped...)
	e.mu.Unlock()
	return writeSecret(sbConfPath(), text)
}

// Up поднимает стек: AWG (если узел awg) + sing-box.
// Идемпотентен: повторный вызов перезапускает стек, а не падает.
func (e *Engine) Up() error {
	e.sb.stop()
	e.awgDown()
	st := e.st.State()
	if st.NodeID == "" && st.Mode != "direct" {
		return fmt.Errorf("no active node")
	}
	var active *Node
	if st.NodeID != "" {
		if n, ok := e.st.Node(st.NodeID); ok {
			active = &n
		} else {
			return fmt.Errorf("active node missing")
		}
	}
	if err := e.generate(); err != nil {
		return err
	}
	if _, err := os.Stat(tool("sing-box")); err != nil {
		return fmt.Errorf("sing-box binary not found in %s", binDir())
	}

	if active != nil && active.Type == "awg" {
		if err := e.awgUp(*active); err != nil {
			e.awgDown()
			return err
		}
	}
	if err := e.sbUp(); err != nil {
		e.awgDown()
		return err
	}
	e.st.SetState(func(s *State) { s.Up = true; s.Err = "" })
	return nil
}

func (e *Engine) sbUp() error {
	env := append(os.Environ(), "PATH="+binDir()+":/usr/sbin:/sbin:/usr/bin:/bin")
	if err := e.sb.start(tool("sing-box"),
		// --disable-color: иначе ANSI-последовательности ломают разбор лога.
		[]string{"run", "--disable-color", "-c", sbConfPath(), "-D", StateDir},
		env, e.log.FromLog); err != nil {
		return fmt.Errorf("sing-box start: %w", err)
	}
	// Готовность = ответ Clash API.
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if e.clash.Alive() {
			return nil
		}
		if !e.sb.Running() {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	tail := e.sb.Tail(12)
	e.sb.stop()
	if len(tail) > 300 {
		tail = tail[len(tail)-300:]
	}
	return fmt.Errorf("sing-box did not become ready: %s", tail)
}

// awgUp поднимает amneziawg-go, применяет конфиг и настраивает policy routing.
func (e *Engine) awgUp(n Node) error {
	if n.AWGConf == "" {
		return fmt.Errorf("awg node without config")
	}
	raw, err := os.ReadFile(awgConfPath(n.AWGConf))
	if err != nil {
		return fmt.Errorf("awg conf: %w", err)
	}
	setconf, addrs, mtu := awgConfParts(string(raw))
	if err := os.MkdirAll(AwgSockDir, 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(tool("amneziawg-go")); err != nil {
		return fmt.Errorf("amneziawg-go binary not found in %s", binDir())
	}
	if ifaceExists(AwgName) {
		e.awgDown()
	}
	env := append(os.Environ(),
		"WG_PROCESS_FOREGROUND=1",
		"PATH="+binDir()+":/usr/sbin:/sbin:/usr/bin:/bin")
	if err := e.awg.start(tool("amneziawg-go"), []string{AwgName}, env, nil); err != nil {
		return fmt.Errorf("amneziawg-go start: %w", err)
	}
	// Ждём появления интерфейса.
	deadline := time.Now().Add(10 * time.Second)
	for !ifaceExists(AwgName) {
		if time.Now().After(deadline) {
			return fmt.Errorf("awg0 did not appear: %s", e.awg.Tail(8))
		}
		time.Sleep(150 * time.Millisecond)
	}

	tmp := awgConfPath(n.AWGConf) + ".uapi"
	if err := writeSecret(tmp, setconf); err != nil {
		return err
	}
	defer os.Remove(tmp)
	if out, err := run(20*time.Second, tool("awg"), "setconf", AwgName, tmp); err != nil {
		return fmt.Errorf("awg setconf: %v: %s", err, out)
	}
	if err := awgIfaceUp(addrs, mtu); err != nil {
		return err
	}
	if err := awgRoutingUp(); err != nil {
		return err
	}
	return nil
}

func (e *Engine) awgDown() {
	awgRoutingDown()
	e.awg.stop()
	if ifaceExists(AwgName) {
		ipQuiet("link", "del", AwgName)
	}
}

// Down гасит стек в обратном порядке.
func (e *Engine) Down() {
	e.sb.stop()
	e.awgDown()
	e.log.Reset()
	// Ошибка предыдущего подъёма после гашения неактуальна.
	e.st.SetState(func(s *State) { s.Up = false; s.Err = "" })
}

// Reload перегенерирует конфиг и перезапускает sing-box.
// Наблюдение на устройстве: sing-box 1.13.21 НЕ перечитывает файл конфига сам,
// поэтому единственный надёжный путь применить правило — рестарт процесса.
// AWG-интерфейс при этом не трогаем: он не зависит от конфига sing-box.
func (e *Engine) Reload() error {
	if err := e.generate(); err != nil {
		return err
	}
	if !e.st.State().Up {
		return nil
	}
	e.sb.stop()
	return e.sbUp()
}

// Restack применяет смену узла/режима/конфига: при поднятом стеке — полный цикл,
// потому что AWG-часть и outbound-и меняются вместе с узлом.
func (e *Engine) Restack() error {
	if !e.st.State().Up {
		return e.generate()
	}
	e.Down()
	if err := e.Up(); err != nil {
		e.st.SetState(func(s *State) { s.Err = err.Error() })
		return err
	}
	return nil
}

// pollLoop раз в секунду тянет соединения из Clash API.
func (e *Engine) pollLoop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		if !e.st.State().Up {
			continue
		}
		c, err := e.clash.Conns()
		if err != nil {
			continue
		}
		e.log.Ingest(c)
	}
}

// awgStatus отдаёт handshake/transfer активного AWG-интерфейса.
func (e *Engine) awgStatus() map[string]any {
	if !ifaceExists(AwgName) {
		return nil
	}
	out := map[string]any{"iface": AwgName}
	if s, err := run(5*time.Second, tool("awg"), "show", AwgName, "latest-handshakes"); err == nil {
		for _, line := range strings.Split(s, "\n") {
			f := strings.Fields(line)
			if len(f) == 2 {
				var ts int64
				fmt.Sscanf(f[1], "%d", &ts)
				if ts > 0 {
					out["handshake_age"] = time.Now().Unix() - ts
				}
			}
		}
	}
	if s, err := run(5*time.Second, tool("awg"), "show", AwgName, "transfer"); err == nil {
		for _, line := range strings.Split(s, "\n") {
			f := strings.Fields(line)
			if len(f) == 3 {
				var rx, tx int64
				fmt.Sscanf(f[1], "%d", &rx)
				fmt.Sscanf(f[2], "%d", &tx)
				out["awg_rx"], out["awg_tx"] = rx, tx
			}
		}
	}
	return out
}
