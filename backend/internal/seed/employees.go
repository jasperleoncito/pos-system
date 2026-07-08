package seed

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/employee"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
)

// seedEmployees adds staff profiles linked to the demo role accounts plus
// Mon–Sat weekly schedules so the clock page works out of the box. Idempotent.
func seedEmployees(ctx context.Context, db *pgxpool.Pool, tenantID string, logger *slog.Logger) error {
	repo := postgres.NewEmployeeRepo(db)
	users := postgres.NewUserRepo(db)

	existing, err := repo.List(ctx, tenantID, "")
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		logger.Info("employees already seeded — skipping", "employees", len(existing))
		return nil
	}

	hired := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	staff := []struct {
		email      string // linked login, "" for none
		name       string
		position   string
		salaryType string
		rate       int64 // centavos
	}{
		{"manager@teresas.ph", "Marco Manager", "Store Manager", employee.SalaryMonthly, 2500000},
		{"cashier@teresas.ph", "Cathy Cashier", "Cashier", employee.SalaryDaily, 65000},
		{"kitchen@teresas.ph", "Ken Kitchen", "Cook", employee.SalaryDaily, 70000},
		{"employee@teresas.ph", "Ella Employee", "Service Crew", employee.SalaryDaily, 61000},
		{"", "Sam Server", "Service Crew", employee.SalaryHourly, 8500},
	}

	// Mon (1) – Sat (6), 09:00–17:00, 10 minutes grace.
	var week []employee.ScheduleDay
	for dow := 1; dow <= 6; dow++ {
		week = append(week, employee.ScheduleDay{
			DayOfWeek: dow, StartTime: "09:00", EndTime: "17:00", GraceMinutes: 10,
		})
	}

	for _, def := range staff {
		e := &employee.Employee{
			FullName: def.name, Position: def.position, Email: def.email,
			SalaryType: def.salaryType, SalaryRate: def.rate,
			HireDate: &hired, IsActive: true,
		}
		if def.email != "" {
			u, err := users.GetByEmail(ctx, def.email)
			if err != nil {
				logger.Warn("seed employee user not found — creating unlinked", "email", def.email)
			} else {
				e.UserID = &u.ID
			}
		}
		if err := repo.Create(ctx, tenantID, e); err != nil {
			return err
		}
		if err := repo.ReplaceSchedule(ctx, tenantID, e.ID, week); err != nil {
			return err
		}
	}

	logger.Info("employees seeded", "employees", len(staff))
	return nil
}
