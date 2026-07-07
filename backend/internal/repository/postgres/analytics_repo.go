package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/analytics"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// Sale-bearing order statuses: completed plus partially refunded (their
// remaining revenue still counts; full refunds subtract via order_refunds).
const saleStatuses = `('completed', 'partially_refunded', 'refunded')`

type AnalyticsRepo struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepo(db *pgxpool.Pool) *AnalyticsRepo { return &AnalyticsRepo{db: db} }

func (r *AnalyticsRepo) SalesBetween(ctx context.Context, tenantID string, rng analytics.Range) (int64, int64, error) {
	var sales, orders int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(total), 0), COUNT(*)
		FROM orders
		WHERE tenant_id=$1 AND status IN `+saleStatuses+` AND deleted_at IS NULL
		  AND completed_at >= $2 AND completed_at < $3`,
		tenantID, rng.From, rng.To).Scan(&sales, &orders)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to sum sales: %w", err)
	}
	return sales, orders, nil
}

func (r *AnalyticsRepo) Summary(ctx context.Context, tenantID string, rng analytics.Range) (*analytics.Summary, error) {
	s := &analytics.Summary{}

	var err error
	if s.GrossSales, s.Orders, err = r.SalesBetween(ctx, tenantID, rng); err != nil {
		return nil, err
	}

	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM refunds
		WHERE tenant_id=$1 AND created_at >= $2 AND created_at < $3`,
		tenantID, rng.From, rng.To).Scan(&s.Refunds); err != nil {
		return nil, fmt.Errorf("failed to sum refunds: %w", err)
	}

	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)
		FROM expenses
		WHERE tenant_id=$1 AND deleted_at IS NULL AND expense_date >= $2::date AND expense_date < $3::date`,
		tenantID, rng.From, rng.To).Scan(&s.Expenses); err != nil {
		return nil, fmt.Errorf("failed to sum expenses: %w", err)
	}

	// COGS: recipe deductions booked as 'sale' movements; the movement's
	// unit_cost snapshot wins, falling back to the item's current cost.
	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM((-m.qty_delta) * (CASE WHEN m.unit_cost > 0 THEN m.unit_cost ELSE i.cost_per_unit END)), 0)::bigint
		FROM inventory_movements m JOIN inventory_items i ON i.id = m.item_id
		WHERE m.tenant_id=$1 AND m.movement_type='sale' AND m.created_at >= $2 AND m.created_at < $3`,
		tenantID, rng.From, rng.To).Scan(&s.COGS); err != nil {
		return nil, fmt.Errorf("failed to sum cogs: %w", err)
	}

	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(oi.qty), 0)
		FROM order_items oi JOIN orders o ON o.id = oi.order_id
		WHERE oi.tenant_id=$1 AND o.status IN `+saleStatuses+` AND o.deleted_at IS NULL
		  AND o.completed_at >= $2 AND o.completed_at < $3 AND oi.deleted_at IS NULL`,
		tenantID, rng.From, rng.To).Scan(&s.ItemsSold); err != nil {
		return nil, fmt.Errorf("failed to sum items sold: %w", err)
	}

	s.NetSales = s.GrossSales - s.Refunds
	s.Profit = s.NetSales - s.COGS - s.Expenses
	if s.Orders > 0 {
		s.AOV = s.GrossSales / s.Orders
	}
	return s, nil
}

func (r *AnalyticsRepo) TopProducts(ctx context.Context, tenantID string, rng analytics.Range, limit int) ([]analytics.TopProduct, error) {
	rows, err := r.db.Query(ctx, `
		SELECT oi.name, SUM(oi.qty)::bigint, SUM(oi.line_total)::bigint AS revenue
		FROM order_items oi JOIN orders o ON o.id = oi.order_id
		WHERE oi.tenant_id=$1 AND o.status IN `+saleStatuses+` AND o.deleted_at IS NULL
		  AND o.completed_at >= $2 AND o.completed_at < $3 AND oi.deleted_at IS NULL
		GROUP BY oi.name ORDER BY revenue DESC LIMIT $4`,
		tenantID, rng.From, rng.To, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top products: %w", err)
	}
	defer rows.Close()
	var out []analytics.TopProduct
	for rows.Next() {
		var p analytics.TopProduct
		if err := rows.Scan(&p.Name, &p.Qty, &p.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan top product: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepo) TopCategories(ctx context.Context, tenantID string, rng analytics.Range, limit int) ([]analytics.TopCategory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(c.name, 'Uncategorized'), SUM(oi.qty)::bigint, SUM(oi.line_total)::bigint AS revenue
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		LEFT JOIN products p ON p.id = oi.product_id
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE oi.tenant_id=$1 AND o.status IN `+saleStatuses+` AND o.deleted_at IS NULL
		  AND o.completed_at >= $2 AND o.completed_at < $3 AND oi.deleted_at IS NULL
		GROUP BY c.name ORDER BY revenue DESC LIMIT $4`,
		tenantID, rng.From, rng.To, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top categories: %w", err)
	}
	defer rows.Close()
	var out []analytics.TopCategory
	for rows.Next() {
		var c analytics.TopCategory
		if err := rows.Scan(&c.Name, &c.Qty, &c.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan top category: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepo) TopEmployees(ctx context.Context, tenantID string, rng analytics.Range, limit int) ([]analytics.TopEmployee, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.full_name, COUNT(*)::bigint, SUM(o.total)::bigint AS revenue
		FROM orders o JOIN users u ON u.id = o.cashier_user_id
		WHERE o.tenant_id=$1 AND o.status IN `+saleStatuses+` AND o.deleted_at IS NULL
		  AND o.completed_at >= $2 AND o.completed_at < $3
		GROUP BY u.full_name ORDER BY revenue DESC LIMIT $4`,
		tenantID, rng.From, rng.To, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top employees: %w", err)
	}
	defer rows.Close()
	var out []analytics.TopEmployee
	for rows.Next() {
		var e analytics.TopEmployee
		if err := rows.Scan(&e.Name, &e.Orders, &e.Revenue); err != nil {
			return nil, fmt.Errorf("failed to scan top employee: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepo) Hourly(ctx context.Context, tenantID string, rng analytics.Range, tz string) ([]analytics.HourPoint, error) {
	rows, err := r.db.Query(ctx, `
		SELECT EXTRACT(HOUR FROM completed_at AT TIME ZONE $4)::int AS h,
		       COALESCE(SUM(total), 0)::bigint, COUNT(*)::bigint
		FROM orders
		WHERE tenant_id=$1 AND status IN `+saleStatuses+` AND deleted_at IS NULL
		  AND completed_at >= $2 AND completed_at < $3
		GROUP BY h ORDER BY h`,
		tenantID, rng.From, rng.To, tz)
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly sales: %w", err)
	}
	defer rows.Close()

	byHour := map[int]analytics.HourPoint{}
	for rows.Next() {
		var p analytics.HourPoint
		if err := rows.Scan(&p.Hour, &p.Sales, &p.Orders); err != nil {
			return nil, fmt.Errorf("failed to scan hourly point: %w", err)
		}
		byHour[p.Hour] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Dense 24-bucket series so charts never skip hours.
	out := make([]analytics.HourPoint, 24)
	for h := 0; h < 24; h++ {
		out[h] = analytics.HourPoint{Hour: h}
		if p, ok := byHour[h]; ok {
			out[h] = p
		}
	}
	return out, nil
}

func (r *AnalyticsRepo) Heatmap(ctx context.Context, tenantID string, rng analytics.Range, tz string) ([]analytics.HeatCell, error) {
	rows, err := r.db.Query(ctx, `
		SELECT EXTRACT(DOW FROM completed_at AT TIME ZONE $4)::int,
		       EXTRACT(HOUR FROM completed_at AT TIME ZONE $4)::int,
		       COALESCE(SUM(total), 0)::bigint, COUNT(*)::bigint
		FROM orders
		WHERE tenant_id=$1 AND status IN `+saleStatuses+` AND deleted_at IS NULL
		  AND completed_at >= $2 AND completed_at < $3
		GROUP BY 1, 2 ORDER BY 1, 2`,
		tenantID, rng.From, rng.To, tz)
	if err != nil {
		return nil, fmt.Errorf("failed to query heatmap: %w", err)
	}
	defer rows.Close()
	var out []analytics.HeatCell
	for rows.Next() {
		var c analytics.HeatCell
		if err := rows.Scan(&c.DayOfWeek, &c.Hour, &c.Sales, &c.Orders); err != nil {
			return nil, fmt.Errorf("failed to scan heat cell: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepo) PaymentMix(ctx context.Context, tenantID string, rng analytics.Range) ([]analytics.PaymentSlice, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.method, SUM(p.amount)::bigint, COUNT(*)::bigint
		FROM payments p JOIN orders o ON o.id = p.order_id
		WHERE p.tenant_id=$1 AND o.status IN `+saleStatuses+` AND o.deleted_at IS NULL
		  AND o.completed_at >= $2 AND o.completed_at < $3
		GROUP BY p.method ORDER BY 2 DESC`,
		tenantID, rng.From, rng.To)
	if err != nil {
		return nil, fmt.Errorf("failed to query payment mix: %w", err)
	}
	defer rows.Close()

	var out []analytics.PaymentSlice
	for rows.Next() {
		var s analytics.PaymentSlice
		if err := rows.Scan(&s.Method, &s.Amount, &s.Count); err != nil {
			return nil, fmt.Errorf("failed to scan payment slice: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Cash tenders include change given back — report net cash kept.
	var change int64
	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(change), 0)
		FROM orders
		WHERE tenant_id=$1 AND status IN `+saleStatuses+` AND deleted_at IS NULL
		  AND completed_at >= $2 AND completed_at < $3`,
		tenantID, rng.From, rng.To).Scan(&change); err != nil {
		return nil, fmt.Errorf("failed to sum change: %w", err)
	}
	for i := range out {
		if out[i].Method == "cash" {
			out[i].Amount -= change
		}
	}
	return out, nil
}

// ---- expenses ----

func (r *AnalyticsRepo) CreateExpense(ctx context.Context, tenantID string, e *analytics.Expense) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO expenses (tenant_id, category, description, amount, expense_date, created_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		tenantID, e.Category, e.Description, e.Amount, e.ExpenseDate, e.CreatedBy,
	).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create expense: %w", err)
	}
	return nil
}

func (r *AnalyticsRepo) ListExpenses(ctx context.Context, tenantID string, rng analytics.Range) ([]analytics.Expense, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, category, description, amount, expense_date, created_by, created_at
		FROM expenses
		WHERE tenant_id=$1 AND deleted_at IS NULL AND expense_date >= $2::date AND expense_date < $3::date
		ORDER BY expense_date DESC, created_at DESC LIMIT 200`,
		tenantID, rng.From, rng.To)
	if err != nil {
		return nil, fmt.Errorf("failed to list expenses: %w", err)
	}
	defer rows.Close()
	var out []analytics.Expense
	for rows.Next() {
		var e analytics.Expense
		if err := rows.Scan(&e.ID, &e.Category, &e.Description, &e.Amount, &e.ExpenseDate, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan expense: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepo) UpdateExpense(ctx context.Context, tenantID string, e *analytics.Expense) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE expenses SET category=$3, description=$4, amount=$5, expense_date=$6, updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`,
		tenantID, e.ID, e.Category, e.Description, e.Amount, e.ExpenseDate)
	if err != nil {
		return fmt.Errorf("failed to update expense: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("expense")
	}
	return nil
}

func (r *AnalyticsRepo) SoftDeleteExpense(ctx context.Context, tenantID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE expenses SET deleted_at=now(), updated_at=now()
		WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return fmt.Errorf("failed to delete expense: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("expense")
	}
	return nil
}
