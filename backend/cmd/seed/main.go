// Package main seeds demo data. Idempotent: safe to run repeatedly.
//
// Seeds a platform super admin plus the demo tenant "Teresa's Eatery"
// with one user per role. All seed accounts share SEED_PASSWORD
// (default "password123").
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/config"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/auth"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/rbac"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/password"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
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

	if err := run(ctx, db, logger); err != nil {
		logger.Error("seed failed", "error", err)
		os.Exit(1)
	}
	logger.Info("seed completed")
}

func run(ctx context.Context, db *pgxpool.Pool, logger *slog.Logger) error {
	users := postgres.NewUserRepo(db)
	tenants := postgres.NewTenantRepo(db)
	settings := postgres.NewTenantSettingsRepo(db)
	memberships := postgres.NewMembershipRepo(db)

	seedPassword := os.Getenv("SEED_PASSWORD")
	if seedPassword == "" {
		seedPassword = "password123"
	}
	hash, err := password.Hash(seedPassword)
	if err != nil {
		return err
	}

	now := time.Now()

	// ---- super admin ----
	if _, err := ensureUser(ctx, users, &auth.User{
		Email: "superadmin@pos.local", PasswordHash: hash, FullName: "Platform Super Admin",
		IsSuperAdmin: true, EmailVerifiedAt: &now, Status: "active",
	}, logger); err != nil {
		return err
	}

	// ---- Teresa's Eatery ----
	owner, err := ensureUser(ctx, users, &auth.User{
		Email: "owner@teresas.ph", PasswordHash: hash, FullName: "Teresa Leoncito",
		EmailVerifiedAt: &now, Status: "active",
	}, logger)
	if err != nil {
		return err
	}

	teresa, err := tenants.GetBySlug(ctx, "teresas-eatery")
	if err != nil {
		var appErr *apperror.Error
		if !errors.As(err, &appErr) || appErr.Kind != apperror.KindNotFound {
			return err
		}
		teresa = &tenant.Tenant{
			Name: "Teresa's Eatery", Slug: "teresas-eatery", OwnerUserID: owner.ID,
			Status: "active", Currency: "PHP", Timezone: "Asia/Manila",
		}
		if err := tenants.Create(ctx, teresa); err != nil {
			return err
		}
		if err := settings.Create(ctx, &tenant.Settings{
			TenantID:       teresa.ID,
			PrimaryColor:   "#16A34A", // Teresa's menu green
			SecondaryColor: "#F97316", // menu heading orange
			AccentColor:    "#CA8A04",
			ReceiptHeader:  "Teresa's Eatery",
			ReceiptFooter:  "Thank you! Please come again.",
			Address:        "Philippines",
		}); err != nil {
			return err
		}
		logger.Info("created tenant", "name", teresa.Name, "id", teresa.ID)
	}

	// ---- per-role users ----
	roleUsers := []struct {
		email string
		name  string
		role  rbac.Role
	}{
		{"owner@teresas.ph", "Teresa Leoncito", rbac.RoleOwner},
		{"manager@teresas.ph", "Marco Manager", rbac.RoleManager},
		{"cashier@teresas.ph", "Cathy Cashier", rbac.RoleCashier},
		{"kitchen@teresas.ph", "Ken Kitchen", rbac.RoleKitchen},
		{"employee@teresas.ph", "Ella Employee", rbac.RoleEmployee},
	}

	for _, ru := range roleUsers {
		u, err := ensureUser(ctx, users, &auth.User{
			Email: ru.email, PasswordHash: hash, FullName: ru.name,
			EmailVerifiedAt: &now, Status: "active",
		}, logger)
		if err != nil {
			return err
		}
		if _, err := memberships.Get(ctx, teresa.ID, u.ID); err == nil {
			continue // membership already seeded
		}
		if err := memberships.Create(ctx, &tenant.Membership{
			TenantID: teresa.ID, UserID: u.ID, Role: string(ru.role),
		}); err != nil {
			return err
		}
		logger.Info("added member", "email", ru.email, "role", ru.role)
	}

	if err := seedMenu(ctx, db, teresa.ID, logger); err != nil {
		return err
	}
	if err := seedInventory(ctx, db, teresa.ID, logger); err != nil {
		return err
	}
	return seedEmployees(ctx, db, teresa.ID, logger)
}

// ensureUser fetches by email or creates when missing.
func ensureUser(ctx context.Context, users *postgres.UserRepo, u *auth.User, logger *slog.Logger) (*auth.User, error) {
	existing, err := users.GetByEmail(ctx, u.Email)
	if err == nil {
		return existing, nil
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindNotFound {
		return nil, err
	}
	if err := users.Create(ctx, u); err != nil {
		return nil, err
	}
	logger.Info("created user", "email", u.Email)
	return u, nil
}
