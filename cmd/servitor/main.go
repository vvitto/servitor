// Command servitor records the messages of a Telegram chat into SQLite.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vitalijnizankovskij/servitor/internal/config"
	"github.com/vitalijnizankovskij/servitor/internal/store"
	"github.com/vitalijnizankovskij/servitor/internal/telegram"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "servitor:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DSN())
	if err != nil {
		return err
	}
	defer db.Close()
	log.Info("database ready", "path", cfg.DBPath)

	svc, err := telegram.New(ctx, telegram.Options{
		Token: cfg.BotToken,
		DB:    db,
		Log:   log,
		Debug: cfg.Debug,
	})
	if err != nil {
		return err
	}

	log.Info("polling for updates")
	svc.Start(ctx)
	log.Info("shutting down")
	return nil
}
