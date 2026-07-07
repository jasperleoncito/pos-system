// Package main seeds demo data (Teresa's Eatery). Seeders are added as
// modules come online: tenants/users in the auth phase, the full menu in
// the catalog phase.
package main

import (
	"log/slog"
	"os"

	"github.com/jasperleoncito/pos-system/backend/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Info("seeder ready — no seeders registered yet", "env", cfg.App.Env)
}
