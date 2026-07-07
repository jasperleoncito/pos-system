package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/analytics"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	redisrepo "github.com/jasperleoncito/pos-system/backend/internal/repository/redis"
)

// analyticsCacheTTL keeps dashboard queries warm without staleness pain;
// completions/refunds/voids invalidate the tenant's prefix anyway.
const analyticsCacheTTL = 3 * time.Minute

// AnalyticsService aggregates sales for the dashboard and owns expenses.
type AnalyticsService struct {
	repo    analytics.Repository
	tenants tenant.Repository
	cache   *redisrepo.Cache
	auditor *AuditService
	logger  *slog.Logger
}

func NewAnalyticsService(repo analytics.Repository, tenants tenant.Repository, cache *redisrepo.Cache, auditor *AuditService, logger *slog.Logger) *AnalyticsService {
	return &AnalyticsService{repo: repo, tenants: tenants, cache: cache, auditor: auditor, logger: logger}
}

func (s *AnalyticsService) cacheKey(tenantID, endpoint string, parts ...string) string {
	key := "analytics:" + tenantID + ":" + endpoint
	for _, p := range parts {
		key += ":" + p
	}
	return key
}

// InvalidateTenant clears cached analytics after sales-affecting events.
func (s *AnalyticsService) InvalidateTenant(ctx context.Context, tenantID string) {
	s.cache.DeletePrefix(ctx, "analytics:"+tenantID+":")
}

// tenantLocation resolves the tenant's timezone (fallback UTC).
func (s *AnalyticsService) tenantLocation(ctx context.Context, tenantID string) (*time.Location, string, error) {
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, "", err
	}
	loc, err := time.LoadLocation(t.Timezone)
	if err != nil {
		return time.UTC, "UTC", nil
	}
	return loc, t.Timezone, nil
}

// ParseRange turns from/to date strings (YYYY-MM-DD, tenant-local, both
// inclusive) into a half-open UTC instant window. Defaults to the last 7 days.
func (s *AnalyticsService) ParseRange(ctx context.Context, tenantID, fromStr, toStr string) (analytics.Range, string, error) {
	loc, tz, err := s.tenantLocation(ctx, tenantID)
	if err != nil {
		return analytics.Range{}, "", err
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	from := today.AddDate(0, 0, -6)
	to := today.AddDate(0, 0, 1)
	if fromStr != "" {
		if d, err := time.ParseInLocation("2006-01-02", fromStr, loc); err == nil {
			from = d
		}
	}
	if toStr != "" {
		if d, err := time.ParseInLocation("2006-01-02", toStr, loc); err == nil {
			to = d.AddDate(0, 0, 1) // inclusive upper day
		}
	}
	if !from.Before(to) {
		return analytics.Range{}, "", apperror.Validation("the from date must be on or before the to date")
	}
	return analytics.Range{From: from, To: to}, tz, nil
}

// Overview returns the four stat cards with previous-period comparisons.
func (s *AnalyticsService) Overview(ctx context.Context, tenantID string) ([]analytics.PeriodStat, error) {
	key := s.cacheKey(tenantID, "overview")
	var cached []analytics.PeriodStat
	if s.cache.GetJSON(ctx, key, &cached) {
		return cached, nil
	}

	loc, _, err := s.tenantLocation(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// Monday-start week.
	weekday := (int(today.Weekday()) + 6) % 7
	weekStart := today.AddDate(0, 0, -weekday)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)

	periods := []struct {
		label      string
		cur, prev  analytics.Range
	}{
		{"today", analytics.Range{From: today, To: today.AddDate(0, 0, 1)},
			analytics.Range{From: today.AddDate(0, 0, -1), To: today}},
		{"wtd", analytics.Range{From: weekStart, To: now},
			analytics.Range{From: weekStart.AddDate(0, 0, -7), To: now.AddDate(0, 0, -7)}},
		{"mtd", analytics.Range{From: monthStart, To: now},
			analytics.Range{From: monthStart.AddDate(0, -1, 0), To: now.AddDate(0, -1, 0)}},
		{"ytd", analytics.Range{From: yearStart, To: now},
			analytics.Range{From: yearStart.AddDate(-1, 0, 0), To: now.AddDate(-1, 0, 0)}},
	}

	stats := make([]analytics.PeriodStat, 0, len(periods))
	for _, p := range periods {
		sales, orders, err := s.repo.SalesBetween(ctx, tenantID, p.cur)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		prevSales, prevOrders, err := s.repo.SalesBetween(ctx, tenantID, p.prev)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		stats = append(stats, analytics.PeriodStat{
			Label: p.label, Sales: sales, Orders: orders,
			PrevSales: prevSales, PrevOrders: prevOrders,
		})
	}
	s.cache.SetJSON(ctx, key, stats, analyticsCacheTTL)
	return stats, nil
}

// Dashboard bundles everything the dashboard needs for one range.
type Dashboard struct {
	Summary       *analytics.Summary       `json:"summary"`
	TopProducts   []analytics.TopProduct   `json:"top_products"`
	TopCategories []analytics.TopCategory  `json:"top_categories"`
	TopEmployees  []analytics.TopEmployee  `json:"top_employees"`
	Hourly        []analytics.HourPoint    `json:"hourly"`
	Heatmap       []analytics.HeatCell     `json:"heatmap"`
	PaymentMix    []analytics.PaymentSlice `json:"payment_mix"`
}

func (s *AnalyticsService) GetDashboard(ctx context.Context, tenantID, fromStr, toStr string) (*Dashboard, error) {
	rng, tz, err := s.ParseRange(ctx, tenantID, fromStr, toStr)
	if err != nil {
		return nil, err
	}
	key := s.cacheKey(tenantID, "dashboard",
		rng.From.UTC().Format("20060102"), rng.To.UTC().Format("20060102"))
	var cached Dashboard
	if s.cache.GetJSON(ctx, key, &cached) {
		return &cached, nil
	}

	d := &Dashboard{}
	if d.Summary, err = s.repo.Summary(ctx, tenantID, rng); err != nil {
		return nil, apperror.Internal(err)
	}
	if d.TopProducts, err = s.repo.TopProducts(ctx, tenantID, rng, 10); err != nil {
		return nil, apperror.Internal(err)
	}
	if d.TopCategories, err = s.repo.TopCategories(ctx, tenantID, rng, 8); err != nil {
		return nil, apperror.Internal(err)
	}
	if d.TopEmployees, err = s.repo.TopEmployees(ctx, tenantID, rng, 8); err != nil {
		return nil, apperror.Internal(err)
	}
	if d.Hourly, err = s.repo.Hourly(ctx, tenantID, rng, tz); err != nil {
		return nil, apperror.Internal(err)
	}
	if d.Heatmap, err = s.repo.Heatmap(ctx, tenantID, rng, tz); err != nil {
		return nil, apperror.Internal(err)
	}
	if d.PaymentMix, err = s.repo.PaymentMix(ctx, tenantID, rng); err != nil {
		return nil, apperror.Internal(err)
	}

	s.cache.SetJSON(ctx, key, d, analyticsCacheTTL)
	return d, nil
}

// ---- expenses ----

func (s *AnalyticsService) CreateExpense(ctx context.Context, tenantID, userID string, e *analytics.Expense) (*analytics.Expense, error) {
	if e.Description == "" {
		return nil, apperror.Validation("a description is required")
	}
	e.CreatedBy = userID
	if err := s.repo.CreateExpense(ctx, tenantID, e); err != nil {
		return nil, apperror.Internal(err)
	}
	s.InvalidateTenant(ctx, tenantID)
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "expense.created",
		EntityType: "expense", EntityID: e.ID,
		After: map[string]any{"amount": e.Amount, "category": e.Category}})
	return e, nil
}

func (s *AnalyticsService) ListExpenses(ctx context.Context, tenantID, fromStr, toStr string) ([]analytics.Expense, error) {
	rng, _, err := s.ParseRange(ctx, tenantID, fromStr, toStr)
	if err != nil {
		return nil, err
	}
	return s.repo.ListExpenses(ctx, tenantID, rng)
}

func (s *AnalyticsService) UpdateExpense(ctx context.Context, tenantID, userID string, e *analytics.Expense) (*analytics.Expense, error) {
	if err := s.repo.UpdateExpense(ctx, tenantID, e); err != nil {
		return nil, err
	}
	s.InvalidateTenant(ctx, tenantID)
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "expense.updated",
		EntityType: "expense", EntityID: e.ID, After: map[string]any{"amount": e.Amount}})
	return e, nil
}

func (s *AnalyticsService) DeleteExpense(ctx context.Context, tenantID, userID, id string) error {
	if err := s.repo.SoftDeleteExpense(ctx, tenantID, id); err != nil {
		return err
	}
	s.InvalidateTenant(ctx, tenantID)
	s.auditor.Record(audit.Log{TenantID: tenantID, UserID: userID, Action: "expense.deleted",
		EntityType: "expense", EntityID: id})
	return nil
}
