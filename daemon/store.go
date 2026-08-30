package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Store — потокобезопасное состояние демона на диске.
type Store struct {
	mu    sync.Mutex
	state State
	nodes []Node
	subs  []Sub
}

func statePath() string   { return filepath.Join(StateDir, "state.json") }
func nodesPath() string   { return filepath.Join(StateDir, "nodes.json") }
func subsPath() string    { return filepath.Join(StateDir, "subs.json") }
func confsDir() string    { return filepath.Join(StateDir, "confs") }
func awgDir() string      { return filepath.Join(StateDir, "awg") }
func rulesetDir() string  { return filepath.Join(StateDir, "rulesets") }
func sbConfPath() string  { return filepath.Join(StateDir, "singbox.json") }
func connLogPath() string { return filepath.Join(StateDir, "connlog.jsonl") }

// writeSecret — атомарная запись 0600 (перенос write_secret из awgd.py).
func writeSecret(path, text string) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.WriteString(text); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err = f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err = os.Chmod(tmp, 0o600); err != nil {
		os.Remove(tmp)
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Chmod(path, 0o600)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeSecret(path, string(b))
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// NewStore готовит каталоги и загружает состояние.
func NewStore() (*Store, error) {
	for _, d := range []string{StateDir, confsDir(), awgDir(), rulesetDir(), LocalBin} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(AwgSockDir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{state: State{Mode: "config"}}
	if err := readJSON(statePath(), &s.state); err != nil && !errors.Is(err, os.ErrNotExist) {
		logf("state.json unreadable: %v", err)
	}
	if s.state.Mode == "" {
		s.state.Mode = "config"
	}
	// up/err — транзиентные: после рестарта стек не поднят, а старая ошибка неактуальна.
	s.state.Up = false
	s.state.Err = ""
	if err := readJSON(nodesPath(), &s.nodes); err != nil && !errors.Is(err, os.ErrNotExist) {
		logf("nodes.json unreadable: %v", err)
	}
	if err := readJSON(subsPath(), &s.subs); err != nil && !errors.Is(err, os.ErrNotExist) {
		logf("subs.json unreadable: %v", err)
	}
	return s, nil
}

func (s *Store) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Store) SetState(fn func(*State)) State {
	s.mu.Lock()
	fn(&s.state)
	st := s.state
	s.mu.Unlock()
	if err := writeJSON(statePath(), st); err != nil {
		logf("save state: %v", err)
	}
	return st
}

func (s *Store) Nodes() []Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Node, len(s.nodes))
	copy(out, s.nodes)
	return out
}

func (s *Store) Node(id string) (Node, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range s.nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

func (s *Store) saveNodesLocked() {
	if err := writeJSON(nodesPath(), s.nodes); err != nil {
		logf("save nodes: %v", err)
	}
}

// AddNodes добавляет узлы, дедуплицируя по ID. Возвращает число реально добавленных.
func (s *Store) AddNodes(list []Node) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	have := map[string]int{}
	for i, n := range s.nodes {
		have[n.ID] = i
	}
	added := 0
	for _, n := range list {
		if i, ok := have[n.ID]; ok {
			n.Latency = s.nodes[i].Latency
			s.nodes[i] = n
			continue
		}
		if n.Latency == 0 {
			n.Latency = -1
		}
		s.nodes = append(s.nodes, n)
		have[n.ID] = len(s.nodes) - 1
		added++
	}
	s.saveNodesLocked()
	return added
}

func (s *Store) DeleteNode(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, n := range s.nodes {
		if n.ID == id {
			s.nodes = append(s.nodes[:i], s.nodes[i+1:]...)
			s.saveNodesLocked()
			return true
		}
	}
	return false
}

func (s *Store) SetLatency(id string, ms int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.nodes {
		if s.nodes[i].ID == id {
			s.nodes[i].Latency = ms
			s.saveNodesLocked()
			return
		}
	}
}

// ReplaceSubNodes заменяет узлы подписки целиком, сохраняя измеренные задержки.
func (s *Store) ReplaceSubNodes(subID string, list []Node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lat := map[string]int{}
	kept := s.nodes[:0:0]
	for _, n := range s.nodes {
		if n.SubID == subID {
			lat[n.ID] = n.Latency
			continue
		}
		kept = append(kept, n)
	}
	for _, n := range list {
		n.SubID = subID
		if v, ok := lat[n.ID]; ok {
			n.Latency = v
		} else if n.Latency == 0 {
			n.Latency = -1
		}
		kept = append(kept, n)
	}
	s.nodes = kept
	s.saveNodesLocked()
}

func (s *Store) Subs() []Sub {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Sub, len(s.subs))
	copy(out, s.subs)
	return out
}

func (s *Store) saveSubsLocked() {
	if err := writeJSON(subsPath(), s.subs); err != nil {
		logf("save subs: %v", err)
	}
}

func (s *Store) AddSub(sub Sub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.subs {
		if s.subs[i].ID == sub.ID {
			s.subs[i].Name = sub.Name
			s.subs[i].URL = sub.URL
			s.saveSubsLocked()
			return
		}
	}
	s.subs = append(s.subs, sub)
	s.saveSubsLocked()
}

func (s *Store) UpdateSub(id string, fn func(*Sub)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.subs {
		if s.subs[i].ID == id {
			fn(&s.subs[i])
			s.saveSubsLocked()
			return true
		}
	}
	return false
}

func (s *Store) DeleteSub(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sub := range s.subs {
		if sub.ID == id {
			s.subs = append(s.subs[:i], s.subs[i+1:]...)
			kept := s.nodes[:0:0]
			for _, n := range s.nodes {
				if n.SubID != id {
					kept = append(kept, n)
				}
			}
			s.nodes = kept
			s.saveSubsLocked()
			s.saveNodesLocked()
			return true
		}
	}
	return false
}

// --- .conf-файлы ---

func confPath(name string) string { return filepath.Join(confsDir(), name+".conf") }

func confNames() []string {
	ents, err := os.ReadDir(confsDir())
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		out = append(out, strings.TrimSuffix(e.Name(), ".conf"))
	}
	sort.Strings(out)
	return out
}

// defaultConfText отдаёт шаблон правил: сначала default.conf из click-пакета
// (штатный RU-шаблон), иначе встроенный минимальный конфиг.
func defaultConfText() string {
	for _, p := range []string{
		filepath.Join(ClickRoot, "default.conf"),
		filepath.Join(StateDir, "default.conf"),
	} {
		if b, err := os.ReadFile(p); err == nil && len(strings.TrimSpace(string(b))) > 0 {
			if _, perr := ParseConf(string(b)); perr == nil {
				return string(b)
			}
			logf("bundled %s is not a valid config, ignoring", p)
		}
	}
	return defaultConf
}

// seedDefaultConf на чистой установке кладёт штатный шаблон и делает его активным.
func (s *Store) seedDefaultConf() {
	if len(confNames()) > 0 {
		return
	}
	text := defaultConfText()
	if err := writeSecret(confPath("default"), text); err != nil {
		logf("seed default conf: %v", err)
		return
	}
	logf("seeded default.conf (%d bytes)", len(text))
	s.SetState(func(st *State) {
		if st.ConfName == "" {
			st.ConfName = "default"
		}
	})
}

func readConf(name string) (string, error) {
	b, err := os.ReadFile(confPath(name))
	return string(b), err
}

func awgConfPath(name string) string { return filepath.Join(awgDir(), name+".conf") }
