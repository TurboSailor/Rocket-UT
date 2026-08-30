package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Строки взяты из реального лога sing-box 1.13.21 на устройстве.
const (
	lineIn    = `17:22:23 [sing-box] +0300 2026-08-30 17:22:23 INFO [1741474495 0ms] inbound/tun[tun-in]: inbound connection to 8.47.69.9:443`
	lineOut   = `17:22:23 [sing-box] +0300 2026-08-30 17:22:23 INFO [1741474495 4ms] outbound/socks[proxy]: outbound connection to 8.47.69.9:443`
	lineInUDP = `17:22:24 [sing-box] +0300 2026-08-30 17:22:24 INFO [99 0ms] inbound/tun[tun-in]: inbound packet connection to 1.1.1.1:53`
	lineOut2  = `17:22:43 [sing-box] +0300 2026-08-30 17:22:43 INFO [2416273740 4ms] outbound/direct[direct]: outbound connection to 34.160.111.145:443`
	lineIn2   = `17:22:43 [sing-box] +0300 2026-08-30 17:22:43 INFO [2416273740 0ms] inbound/tun[tun-in]: inbound connection to 34.160.111.145:443`
)

func newTestLog(t *testing.T) *connLog {
	t.Helper()
	// connLogPath() смотрит в StateDir; в тесте файл не нужен.
	return &connLog{byID: map[string]int{}}
}

func TestConnLogFromSingBoxLog(t *testing.T) {
	cl := newTestLog(t)
	cl.FromLog(lineIn)
	cl.FromLog(lineOut)
	cl.FromLog(lineIn2)
	cl.FromLog(lineOut2)

	got := cl.Since(0)
	if len(got) != 2 {
		t.Fatalf("entries = %d, want 2: %+v", len(got), got)
	}
	// Одно соединение = одна запись: inbound и outbound склеиваются по id.
	first := got[0]
	if first.Host != "8.47.69.9" || first.Port != 443 || first.Net != "tcp" {
		t.Errorf("first entry = %+v", first)
	}
	if first.Policy != "proxy" {
		t.Errorf("first policy = %q, want proxy", first.Policy)
	}
	if got[1].Policy != "direct" {
		t.Errorf("second policy = %q, want direct", got[1].Policy)
	}
}

func TestConnLogUDP(t *testing.T) {
	cl := newTestLog(t)
	cl.FromLog(lineInUDP)
	got := cl.Since(0)
	if len(got) != 1 {
		t.Fatalf("entries = %d", len(got))
	}
	if got[0].Net != "udp" || got[0].Port != 53 {
		t.Errorf("entry = %+v", got[0])
	}
}

func TestConnLogIgnoresNoise(t *testing.T) {
	cl := newTestLog(t)
	for _, l := range []string{
		`INFO inbound/tun[tun-in]: started at rocket0`,
		`INFO clash-api: restful api listening at 127.0.0.1:9091`,
		`WARN inbound/tun[tun-in]: enable offload: set udp offload: TUNSETOFFLOAD: invalid argument`,
		`INFO sing-box started (0.06s)`,
		``,
	} {
		cl.FromLog(l)
	}
	if got := cl.Since(0); len(got) != 0 {
		t.Errorf("noise produced entries: %+v", got)
	}
}

// Регрессия: sing-box без --disable-color обёртывает id в ANSI, и разбор ломался.
func TestConnLogStripsANSI(t *testing.T) {
	cl := newTestLog(t)
	cl.FromLog("17:30:01 \x1b[36mINFO\x1b[0m [\x1b[38;5;140m898658940\x1b[0m 0ms] " +
		"inbound/tun[tun-in]: inbound connection to 74.125.205.16:993")
	cl.FromLog("17:30:01 \x1b[36mINFO\x1b[0m [\x1b[38;5;140m898658940\x1b[0m 0ms] " +
		"outbound/direct[direct]: outbound connection to 74.125.205.16:993")
	got := cl.Since(0)
	if len(got) != 1 {
		t.Fatalf("entries = %d, want 1: %+v", len(got), got)
	}
	if got[0].Host != "74.125.205.16" || got[0].Port != 993 || got[0].Policy != "direct" {
		t.Errorf("entry = %+v", got[0])
	}
}

func TestConnLogRingTrim(t *testing.T) {
	dir := t.TempDir()
	f, err := os.OpenFile(filepath.Join(dir, "connlog.jsonl"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cl := &connLog{byID: map[string]int{}, file: f}
	for i := range connRingCap + 50 {
		cl.FromLog(`INFO [` + itoaTest(i) + ` 0ms] inbound/tun[tun-in]: inbound connection to 1.2.3.4:443`)
	}
	if got := len(cl.Since(0)); got != connRingCap {
		t.Errorf("ring size = %d, want %d", got, connRingCap)
	}
	// Индексы должны остаться валидными после обрезки: обновление не паникует.
	cl.FromLog(`INFO [` + itoaTest(connRingCap+49) + ` 4ms] outbound/socks[proxy]: outbound connection to 1.2.3.4:443`)
	last := cl.Since(0)
	if last[len(last)-1].Policy != "proxy" {
		t.Errorf("last entry policy = %q", last[len(last)-1].Policy)
	}
}

func TestConnLogIngestTraffic(t *testing.T) {
	cl := newTestLog(t)
	cl.FromLog(lineIn)
	cl.FromLog(lineOut)
	c := &clashConns{DownloadTotal: 4096, UploadTotal: 1024}
	cc := clashConn{Upload: 100, Download: 900, Rule: "domain_suffix", RulePayload: "ipify.org"}
	cc.Metadata.DestinationPort = "443"
	cc.Metadata.DestinationIP = "8.47.69.9"
	cc.Metadata.Host = "8.47.69.9"
	cc.Metadata.Network = "tcp"
	c.Connections = []clashConn{cc}
	cl.Ingest(c)

	rx, tx := cl.Traffic()
	if rx != 4096 || tx != 1024 {
		t.Errorf("traffic = %d/%d", rx, tx)
	}
	e := cl.Since(0)[0]
	if e.Down != 900 || e.Up != 100 {
		t.Errorf("per-conn bytes = %d/%d", e.Down, e.Up)
	}
	if e.Rule != "domain_suffix ipify.org" {
		t.Errorf("rule = %q", e.Rule)
	}
}

// Записи из лога знают только IP; домен обязан подставиться из Clash API,
// иначе диалог «добавить в правило» может предложить лишь IP-CIDR.
func TestConnLogUpgradesIPToDomain(t *testing.T) {
	cl := newTestLog(t)
	cl.FromLog(lineIn)
	cl.FromLog(lineOut)
	if got := cl.Since(0)[0].Host; got != "8.47.69.9" {
		t.Fatalf("pre-enrichment host = %q", got)
	}
	cc := clashConn{Upload: 1, Download: 2}
	cc.Metadata.DestinationIP = "8.47.69.9"
	cc.Metadata.DestinationPort = "443"
	cc.Metadata.Host = "api.ipify.org"
	cc.Metadata.Network = "tcp"
	cl.Ingest(&clashConns{Connections: []clashConn{cc}})

	e2 := cl.Since(0)[0]
	if e2.Host != "api.ipify.org" {
		t.Errorf("host = %q, want api.ipify.org", e2.Host)
	}
	if e2.IP != "8.47.69.9" {
		t.Errorf("ip = %q, want 8.47.69.9", e2.IP)
	}
}

func itoaTest(i int) string {
	if i == 0 {
		return "0"
	}
	var out []byte
	for i > 0 {
		out = append([]byte{byte('0' + i%10)}, out...)
		i /= 10
	}
	return string(out)
}
