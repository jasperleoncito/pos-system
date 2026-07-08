package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/customer"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
)

// seedCustomers adds a few loyalty members so the POS attach flow and
// points redemption are demonstrable out of the box. Idempotent.
func seedCustomers(ctx context.Context, db *pgxpool.Pool, tenantID string, logger *slog.Logger) error {
	repo := postgres.NewCustomerRepo(db)

	existing, err := repo.List(ctx, tenantID, "")
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		logger.Info("customers already seeded — skipping", "customers", len(existing))
		return nil
	}

	bday := func(m time.Month, d int) *time.Time {
		t := time.Date(1990, m, d, 0, 0, 0, 0, time.UTC)
		return &t
	}
	demo := []customer.Customer{
		{FullName: "Juan Dela Cruz", Phone: "09171234567", Email: "juan@example.ph", Birthday: bday(time.March, 14), IsActive: true},
		{FullName: "Maria Santos", Phone: "09182345678", Email: "maria@example.ph", Birthday: bday(time.July, 22), IsActive: true},
		{FullName: "Pedro Reyes", Phone: "09193456789", IsActive: true},
	}
	for i := range demo {
		if err := repo.Create(ctx, tenantID, &demo[i]); err != nil {
			return err
		}
	}
	logger.Info("customers seeded", "customers", len(demo))
	return nil
}
