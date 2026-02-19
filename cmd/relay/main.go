package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/SWAI-Ltd/Qumbed/internal/mesh"
)

func main() {
	addr := flag.String("addr", ":6121", "listen address")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	srv, err := mesh.RunRelay(ctx, *addr)
	if err != nil {
		slog.Error("failed to start relay", "err", err)
		os.Exit(1)
	}
	<-ctx.Done()
	_ = srv
	slog.Info("relay shutting down")
}
