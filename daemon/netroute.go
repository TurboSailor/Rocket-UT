package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Работа с iproute2. nftables на этом ядре нет (CONFIG_NF_TABLES не задан),
// поэтому всё делается через ip rule / ip route + fwmark.

func run(timeout time.Duration, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "PATH=/usr/sbin:/sbin:/usr/bin:/bin")
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func ip(args ...string) (string, error) { return run(10*time.Second, "ip", args...) }

// ipQuiet выполняет команду, игнорируя ошибку (для идемпотентных del).
func ipQuiet(args ...string) { _, _ = ip(args...) }

func ifaceExists(name string) bool {
	_, err := os.Stat("/sys/class/net/" + name)
	return err == nil
}

func markHex(m int) string { return "0x" + strconv.FormatInt(int64(m), 16) }

// awgRoutingUp настраивает policy routing для AWG-выхода.
// Приоритеты 8800/8801 меньше iproute2_rule_index=9000 у TUN, поэтому
// проверяются раньше правил sing-box.
func awgRoutingUp() error {
	tbl := strconv.Itoa(AwgTable)
	if _, err := ip("route", "replace", "default", "dev", AwgName, "table", tbl); err != nil {
		return fmt.Errorf("awg route: %w", err)
	}
	awgRoutingDownRules()
	if _, err := ip("rule", "add", "fwmark", markHex(AwgMark), "lookup", tbl,
		"priority", strconv.Itoa(AwgRulePrio)); err != nil {
		return fmt.Errorf("awg rule: %w", err)
	}
	// Сокет самого amneziawg-go помечен EscapeMark: уводим его в main,
	// иначе UDP к эндпоинту уйдёт в туннель и получится петля.
	if _, err := ip("rule", "add", "fwmark", markHex(EscapeMark), "lookup", "main",
		"priority", strconv.Itoa(EscapePrio)); err != nil {
		return fmt.Errorf("escape rule: %w", err)
	}
	return nil
}

func awgRoutingDownRules() {
	ipQuiet("rule", "del", "priority", strconv.Itoa(AwgRulePrio))
	ipQuiet("rule", "del", "priority", strconv.Itoa(EscapePrio))
}

func awgRoutingDown() {
	awgRoutingDownRules()
	ipQuiet("route", "flush", "table", strconv.Itoa(AwgTable))
}

// awgIfaceUp поднимает адрес и MTU на интерфейсе awg0.
func awgIfaceUp(addrs []string, mtu int) error {
	if _, err := ip("link", "set", AwgName, "up"); err != nil {
		return fmt.Errorf("link up: %w", err)
	}
	if mtu <= 0 {
		mtu = TunMTU
	}
	if _, err := ip("link", "set", AwgName, "mtu", strconv.Itoa(mtu)); err != nil {
		logf("awg mtu: %v", err)
	}
	for _, a := range addrs {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, err := ip("addr", "add", a, "dev", AwgName); err != nil {
			// Адрес мог остаться с прошлого запуска — не фатально.
			logf("awg addr %s: %v", a, err)
		}
	}
	return nil
}

// awgConfParts вытаскивает Address/MTU из INI и отдаёт конфиг для `awg setconf`
// (setconf понимает только ключи uapi-уровня; Address/MTU/DNS/Table применяем сами).
func awgConfParts(text string) (setconf string, addrs []string, mtu int) {
	var kept []string
	section := ""
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if s, ok := sectionName(t); ok {
			section = s
			kept = append(kept, t)
			continue
		}
		k, v, ok := splitKV(t)
		if !ok {
			continue
		}
		switch strings.ToLower(k) {
		case "address":
			addrs = append(addrs, splitCSV(v)...)
			continue
		case "mtu":
			if i, err := strconv.Atoi(v); err == nil {
				mtu = i
			}
			continue
		case "dns", "table", "preup", "postup", "predown", "postdown", "saveconfig":
			continue
		case "fwmark":
			continue // марку ставим сами
		}
		kept = append(kept, t)
		_ = section
	}
	// Помечаем исходящий сокет, чтобы escape-правило вывело его мимо TUN.
	out := make([]string, 0, len(kept)+1)
	for _, l := range kept {
		out = append(out, l)
		if s, ok := sectionName(l); ok && s == "interface" {
			out = append(out, "FwMark = "+markHex(EscapeMark))
		}
	}
	return strings.Join(out, "\n") + "\n", addrs, mtu
}
