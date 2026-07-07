package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportRepo runs report queries and returns generic rows keyed by the
// query's column names — the report service supplies column metadata.
type ReportRepo struct {
	db *pgxpool.Pool
}

func NewReportRepo(db *pgxpool.Pool) *ReportRepo { return &ReportRepo{db: db} }

// queryRows maps every result row to column-name → value.
func (r *ReportRepo) queryRows(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("report query failed: %w", err)
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	var out []map[string]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("report row read failed: %w", err)
		}
		row := make(map[string]any, len(fields))
		for i, f := range fields {
			row[f.Name] = values[i]
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ReportRepo) Sales(ctx context.Context, tenantID string, from, to time.Time, tz string) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT o.order_number::text AS number,
		       to_char(o.completed_at AT TIME ZONE $4, 'YYYY-MM-DD HH24:MI') AS completed,
		       u.full_name AS cashier, o.order_type AS type, o.status,
		       o.subtotal::bigint AS subtotal, o.discount_total::bigint AS discount,
		       o.tax_total::bigint AS tax, o.total::bigint AS total
		FROM orders o JOIN users u ON u.id = o.cashier_user_id
		WHERE o.tenant_id=$1 AND o.status IN `+saleStatuses+` AND o.deleted_at IS NULL
		  AND o.completed_at >= $2 AND o.completed_at < $3
		ORDER BY o.completed_at`, tenantID, from, to, tz)
}

func (r *ReportRepo) Inventory(ctx context.Context, tenantID string) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT i.name AS item, i.type, u.abbreviation AS unit,
		       i.current_stock::float8 AS stock, i.reorder_level::float8 AS reorder,
		       i.cost_per_unit::bigint AS unit_cost,
		       (i.current_stock * i.cost_per_unit)::bigint AS stock_value,
		       CASE WHEN i.current_stock <= 0 THEN 'out of stock'
		            WHEN i.current_stock <= i.reorder_level THEN 'low' ELSE 'ok' END AS status
		FROM inventory_items i JOIN units u ON u.id = i.unit_id
		WHERE i.tenant_id=$1 AND i.deleted_at IS NULL
		ORDER BY i.name`, tenantID)
}

func (r *ReportRepo) Employees(ctx context.Context, tenantID string) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT full_name AS employee, position, salary_type,
		       salary_rate::bigint AS rate,
		       COALESCE(to_char(hire_date, 'YYYY-MM-DD'), '') AS hired,
		       CASE WHEN is_active THEN 'active' ELSE 'inactive' END AS status
		FROM employees
		WHERE tenant_id=$1 AND deleted_at IS NULL
		ORDER BY full_name`, tenantID)
}

func (r *ReportRepo) Attendance(ctx context.Context, tenantID string, from, to time.Time, tz string) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT e.full_name AS employee,
		       to_char(a.clock_in AT TIME ZONE $4, 'YYYY-MM-DD') AS date,
		       to_char(a.clock_in AT TIME ZONE $4, 'HH24:MI') AS clock_in,
		       COALESCE(to_char(a.clock_out AT TIME ZONE $4, 'HH24:MI'), '') AS clock_out,
		       a.late_minutes::bigint AS late_min, a.overtime_minutes::bigint AS ot_min,
		       a.break_minutes::bigint AS break_min,
		       CASE WHEN a.clock_out IS NULL THEN 0
		            ELSE GREATEST(0, (EXTRACT(EPOCH FROM a.clock_out - a.clock_in)/60 - a.break_minutes))::bigint END AS worked_min,
		       a.status
		FROM attendance_records a JOIN employees e ON e.id = a.employee_id
		WHERE a.tenant_id=$1 AND a.clock_in >= $2 AND a.clock_in < $3
		ORDER BY a.clock_in DESC`, tenantID, from, to, tz)
}

// DailyProfit merges per-day gross, refunds, COGS, and expenses.
func (r *ReportRepo) DailyProfit(ctx context.Context, tenantID string, from, to time.Time, tz string) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		WITH days AS (
			SELECT generate_series($2::timestamptz AT TIME ZONE $4, ($3::timestamptz AT TIME ZONE $4) - interval '1 day', '1 day')::date AS day
		), sales AS (
			SELECT (completed_at AT TIME ZONE $4)::date AS day, SUM(total)::bigint AS gross
			FROM orders WHERE tenant_id=$1 AND status IN `+saleStatuses+` AND deleted_at IS NULL
			  AND completed_at >= $2 AND completed_at < $3 GROUP BY 1
		), refunds AS (
			SELECT (created_at AT TIME ZONE $4)::date AS day, SUM(amount)::bigint AS refunded
			FROM refunds WHERE tenant_id=$1 AND created_at >= $2 AND created_at < $3 GROUP BY 1
		), cogs AS (
			SELECT (m.created_at AT TIME ZONE $4)::date AS day,
			       SUM((-m.qty_delta) * (CASE WHEN m.unit_cost > 0 THEN m.unit_cost ELSE i.cost_per_unit END))::bigint AS cogs
			FROM inventory_movements m JOIN inventory_items i ON i.id = m.item_id
			WHERE m.tenant_id=$1 AND m.movement_type='sale' AND m.created_at >= $2 AND m.created_at < $3 GROUP BY 1
		), exp AS (
			SELECT expense_date AS day, SUM(amount)::bigint AS expenses
			FROM expenses WHERE tenant_id=$1 AND deleted_at IS NULL
			  AND expense_date >= ($2::timestamptz AT TIME ZONE $4)::date
			  AND expense_date < ($3::timestamptz AT TIME ZONE $4)::date GROUP BY 1
		)
		SELECT to_char(d.day, 'YYYY-MM-DD') AS date,
		       COALESCE(s.gross, 0) AS gross,
		       COALESCE(r.refunded, 0) AS refunds,
		       COALESCE(c.cogs, 0) AS cogs,
		       COALESCE(e.expenses, 0) AS expenses,
		       COALESCE(s.gross, 0) - COALESCE(r.refunded, 0) - COALESCE(c.cogs, 0) - COALESCE(e.expenses, 0) AS profit
		FROM days d
		LEFT JOIN sales s ON s.day = d.day
		LEFT JOIN refunds r ON r.day = d.day
		LEFT JOIN cogs c ON c.day = d.day
		LEFT JOIN exp e ON e.day = d.day
		ORDER BY d.day`, tenantID, from, to, tz)
}

func (r *ReportRepo) DailyTax(ctx context.Context, tenantID string, from, to time.Time, tz string) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT to_char((completed_at AT TIME ZONE $4)::date, 'YYYY-MM-DD') AS date,
		       COUNT(*)::bigint AS orders,
		       SUM(total)::bigint AS gross_sales,
		       SUM(total - tax_total)::bigint AS net_of_tax,
		       SUM(tax_total)::bigint AS tax_collected
		FROM orders
		WHERE tenant_id=$1 AND status IN `+saleStatuses+` AND deleted_at IS NULL
		  AND completed_at >= $2 AND completed_at < $3
		GROUP BY 1 ORDER BY 1`, tenantID, from, to, tz)
}

func (r *ReportRepo) Receipts(ctx context.Context, tenantID string, from, to time.Time, tz string) ([]map[string]any, error) {
	return r.queryRows(ctx, `
		SELECT o.order_number::text AS number, o.id AS order_id,
		       to_char(o.completed_at AT TIME ZONE $4, 'YYYY-MM-DD HH24:MI') AS completed,
		       o.total::bigint AS total, o.tendered::bigint AS tendered, o.change::bigint AS change,
		       COALESCE((SELECT string_agg(DISTINCT p.method, ', ') FROM payments p WHERE p.order_id = o.id), '') AS methods
		FROM orders o
		WHERE o.tenant_id=$1 AND o.status IN `+saleStatuses+` AND o.deleted_at IS NULL
		  AND o.completed_at >= $2 AND o.completed_at < $3
		ORDER BY o.completed_at DESC`, tenantID, from, to, tz)
}
