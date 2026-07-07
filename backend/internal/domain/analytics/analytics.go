// Package analytics defines dashboard aggregates and operating expenses.
// All money is integer centavos; buckets use the tenant's timezone.
package analytics

import (
	"context"
	"time"
)

// Summary is the headline block for a date range.
type Summary struct {
	GrossSales  int64   `json:"gross_sales"`
	Orders      int64   `json:"orders"`
	AOV         int64   `json:"aov"`
	Refunds     int64   `json:"refunds"`
	Expenses    int64   `json:"expenses"`
	COGS        int64   `json:"cogs"`
	Profit      int64   `json:"profit"` // gross − refunds − cogs − expenses
	NetSales    int64   `json:"net_sales"`
	ItemsSold   int64   `json:"items_sold"`
}

// PeriodStat is one stat card: current window vs the previous one.
type PeriodStat struct {
	Label      string `json:"label"` // today | wtd | mtd | ytd
	Sales      int64  `json:"sales"`
	Orders     int64  `json:"orders"`
	PrevSales  int64  `json:"prev_sales"`
	PrevOrders int64  `json:"prev_orders"`
}

type TopProduct struct {
	Name    string `json:"name"`
	Qty     int64  `json:"qty"`
	Revenue int64  `json:"revenue"`
}

type TopCategory struct {
	Name    string `json:"name"`
	Qty     int64  `json:"qty"`
	Revenue int64  `json:"revenue"`
}

type TopEmployee struct {
	Name    string `json:"name"`
	Orders  int64  `json:"orders"`
	Revenue int64  `json:"revenue"`
}

type HourPoint struct {
	Hour   int   `json:"hour"` // 0–23 tenant-local
	Sales  int64 `json:"sales"`
	Orders int64 `json:"orders"`
}

type HeatCell struct {
	DayOfWeek int   `json:"day_of_week"` // 0 = Sunday, tenant-local
	Hour      int   `json:"hour"`
	Sales     int64 `json:"sales"`
	Orders    int64 `json:"orders"`
}

type PaymentSlice struct {
	Method string `json:"method"`
	Amount int64  `json:"amount"`
	Count  int64  `json:"count"`
}

type Expense struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Amount      int64     `json:"amount"`
	ExpenseDate time.Time `json:"expense_date"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// Range is a half-open window [From, To).
type Range struct {
	From time.Time
	To   time.Time
}

type Repository interface {
	SalesBetween(ctx context.Context, tenantID string, r Range) (sales int64, orders int64, err error)
	Summary(ctx context.Context, tenantID string, r Range) (*Summary, error)
	TopProducts(ctx context.Context, tenantID string, r Range, limit int) ([]TopProduct, error)
	TopCategories(ctx context.Context, tenantID string, r Range, limit int) ([]TopCategory, error)
	TopEmployees(ctx context.Context, tenantID string, r Range, limit int) ([]TopEmployee, error)
	Hourly(ctx context.Context, tenantID string, r Range, tz string) ([]HourPoint, error)
	Heatmap(ctx context.Context, tenantID string, r Range, tz string) ([]HeatCell, error)
	PaymentMix(ctx context.Context, tenantID string, r Range) ([]PaymentSlice, error)

	CreateExpense(ctx context.Context, tenantID string, e *Expense) error
	ListExpenses(ctx context.Context, tenantID string, r Range) ([]Expense, error)
	UpdateExpense(ctx context.Context, tenantID string, e *Expense) error
	SoftDeleteExpense(ctx context.Context, tenantID, id string) error
}
