package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// binDir: бинари из click, иначе локальная копия (отладка без переустановки пакета).
func binDir() string {
	if st, err := os.Stat(ClickBin); err == nil && st.IsDir() {
		if _, err := os.Stat(filepath.Join(ClickBin, "sing-box")); err == nil {
			return ClickBin
		}
	}
	return LocalBin
}

func tool(name string) string { return filepath.Join(binDir(), name) }

// proc — один управляемый дочерний процесс с кольцевым буфером stderr.
type proc struct {
	mu   sync.Mutex
	cmd  *exec.Cmd
	name string
	tail []string
	done chan struct{}
}

func (p *proc) addLine(s string) {
	p.mu.Lock()
	p.tail = append(p.tail, s)
	if len(p.tail) > 80 {
		p.tail = p.tail[len(p.tail)-80:]
	}
	p.mu.Unlock()
}

func (p *proc) Tail(n int) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n > len(p.tail) {
		n = len(p.tail)
	}
	return strings.Join(p.tail[len(p.tail)-n:], "\n")
}

func (p *proc) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cmd != nil && p.cmd.Process != nil && p.done != nil && !isClosed(p.done)
}

func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// start запускает процесс, перекачивая stdout/stderr в лог демона и в tail.
func (p *proc) start(bin string, args []string, env []string, onLine func(string)) error {
	p.mu.Lock()
	if p.cmd != nil && p.cmd.Process != nil && p.done != nil && !isClosed(p.done) {
		p.mu.Unlock()
		return fmt.Errorf("%s already running", p.name)
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		p.mu.Unlock()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		p.mu.Unlock()
		return err
	}
	if err := cmd.Start(); err != nil {
		p.mu.Unlock()
		return err
	}
	done := make(chan struct{})
	p.cmd, p.done, p.tail = cmd, done, nil
	p.mu.Unlock()

	pump := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 64*1024), 512*1024)
		for sc.Scan() {
			line := sc.Text()
			p.addLine(line)
			logf("[%s] %s", p.name, line)
			if onLine != nil {
				onLine(line)
			}
		}
	}
	go pump(stdout)
	go pump(stderr)
	go func() {
		err := cmd.Wait()
		if err != nil {
			logf("[%s] exited: %v", p.name, err)
		} else {
			logf("[%s] exited", p.name)
		}
		close(done)
	}()
	return nil
}

// stop гасит процесс: SIGTERM, затем SIGKILL по таймауту.
func (p *proc) stop() {
	p.mu.Lock()
	cmd, done := p.cmd, p.done
	p.cmd = nil
	p.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}
}
