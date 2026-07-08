// Package main is the background worker: asynq consumers for email
// delivery and notifications, plus per-tenant daily-summary crons.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/config"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/mailer"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
	"github.com/jasperleoncito/pos-system/backend/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
		Analytics:     postgres.NewAnalyticsRepo(db),
		Mailer: mailer.New(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.User,
			cfg.SMTP.Password, cfg.SMTP.From, cfg.SMTP.FromName),
		AppName: cfg.App.Name,
		AppURL:  cfg.HTTP.AppURL,
		Logger:  logger,
	}

	redisOpt := asynq.RedisClientOpt{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password}

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:    5,
		ShutdownTimeout: 10 * time.Second,
		Logger:          asynqLogger{logger},
	})
	scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{Logger: asynqLogger{logger}})

	if err := worker.RegisterDailySummaries(ctx, scheduler, handlers.Tenants, logger); err != nil {
		logger.Warn("daily summary registration failed", "error", err)
	}

	go func() {
		if err := scheduler.Run(); err != nil {
			logger.Error("scheduler stopped", "error", err)
		}
	}()

	logger.Info("worker starting", "env", cfg.App.Env, "redis", cfg.Redis.Addr)
	if err := server.Start(handlers.Mux()); err != nil {
		logger.Error("worker failed to start", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()
	scheduler.Shutdown()
	server.Shutdown()
	logger.Info("worker stopped")
}

// asynqLogger adapts slog to asynq's logging interface.
type asynqLogger struct{ l *slog.Logger }

func (a asynqLogger) Debug(args ...any) { a.l.Debug(fmt.Sprint(args...)) }
func (a asynqLogger) Info(args ...any)  { a.l.Info(fmt.Sprint(args...)) }
func (a asynqLogger) Warn(args ...any)  { a.l.Warn(fmt.Sprint(args...)) }
func (a asynqLogger) Error(args ...any) { a.l.Error(fmt.Sprint(args...)) }
func (a asynqLogger) Fatal(args ...any) {
	a.l.Error(fmt.Sprint(args...))
	os.Exit(1)
}
