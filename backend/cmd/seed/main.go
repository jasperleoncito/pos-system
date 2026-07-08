// Package main runs the full demo seeder manually:
//
//	docker compose exec backend /app/seed
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/config"
	"github.com/jasperleoncito/pos-system/backend/internal/seed"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := seed.Run(ctx, db, logger); err != nil {
		logger.Error("seed failed", "error", err)
		os.Exit(1)
	}
	logger.Info("seed completed")
}
