// Package main is the background worker entrypoint. Job handlers are
// registered here as modules come online (emails, alerts, summaries).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jasperleoncito/pos-system/backend/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Info("worker starting", "env", cfg.App.Env, "redis", cfg.Redis.Addr)

	// asynq server registration lands in the notifications phase; until
	// then the worker idles so the compose stack stays green.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	logger.Info("worker stopped")
}
