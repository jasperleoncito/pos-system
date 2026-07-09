// Command sweep runs the subscription billing sweep once and exits:
// it sends 3-day renewal notices and deactivates past-due businesses.
// The worker runs this hourly on a schedule; this binary lets ops (or
// tests) trigger it on demand, mirroring cmd/seed.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/config"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/mailer"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
	"github.com/jasperleoncito/pos-system/backend/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	handlers := &worker.Handlers{
		Tenants:       postgres.NewTenantRepo(db),
		Memberships:   postgres.NewMembershipRepo(db),
		Users:         postgres.NewUserRepo(db),
		Notifications: postgres.NewNotificationRepo(db),
		Billing:       postgres.NewBillingRepo(db),
		Mailer: mailer.New(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.User,
			cfg.SMTP.Password, cfg.SMTP.From, cfg.SMTP.FromName),
		AppName: cfg.App.Name,
		AppURL:  cfg.HTTP.AppURL,
		Logger:  logger,
	}

	if err := handlers.HandleBillingSweep(ctx, nil); err != nil {
		logger.Error("billing sweep failed", "error", err)
		os.Exit(1)
	}
	logger.Info("billing sweep complete")
}
