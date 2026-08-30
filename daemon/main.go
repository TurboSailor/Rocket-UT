package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func logf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
}

func main() {
	st, err := NewStore()
	if err != nil {
		logf("fatal: state init: %v", err)
		os.Exit(1)
	}
	en := NewEngine(st)
	api := &API{st: st, en: en}

	// Чистая установка получает штатный шаблон правил.
	st.seedDefaultConf()

	// Стартовая генерация — в фоне: RULE-SET качаются по сети и могут занять
	// десятки секунд, а порт должен слушаться сразу, иначе UI не подключится.
	go func() {
		if err := en.generate(); err != nil {
			logf("initial config: %v", err)
		}
	}()
	go en.pollLoop()

	srv := &http.Server{
		Addr:              APIAddr,
		Handler:           api.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sig
		logf("signal %v: shutting down", s)
		en.Down()
		_ = srv.Close()
	}()

	logf("rocketd: listening on %s (bin dir %s)", APIAddr, binDir())
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logf("fatal: %v", err)
		os.Exit(1)
	}
}
