package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/storage"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/export"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
)

// Report types offered by the reports center.
var reportTypes = map[string]struct {
	title    string
	rangeful bool // uses from/to
	columns  []export.Column
	totals   []string // money/number keys to sum for the footer
}{
	"sales": {
		title: "Sales Report", rangeful: true,
		columns: []export.Column{
			{Key: "number", Label: "Order #", Kind: export.KindText},
			{Key: "completed", Label: "Completed", Kind: export.KindText},
			{Key: "cashier", Label: "Cashier", Kind: export.KindText},
			{Key: "type", Label: "Type", Kind: export.KindText},
			{Key: "status", Label: "Status", Kind: export.KindText},
			{Key: "subtotal", Label: "Subtotal", Kind: export.KindMoney},
			{Key: "discount", Label: "Discount", Kind: export.KindMoney},
			{Key: "tax", Label: "Tax", Kind: export.KindMoney},
			{Key: "total", Label: "Total", Kind: export.KindMoney},
		},
		totals: []string{"subtotal", "discount", "tax", "total"},
	},
	"inventory": {
		title: "Inventory Report", rangeful: false,
		columns: []export.Column{
			{Key: "item", Label: "Item", Kind: export.KindText},
			{Key: "type", Label: "Type", Kind: export.KindText},
			{Key: "unit", Label: "Unit", Kind: export.KindText},
			{Key: "stock", Label: "Stock", Kind: export.KindNum},
			{Key: "reorder", Label: "Reorder at", Kind: export.KindNum},
			{Key: "unit_cost", Label: "Unit cost", Kind: export.KindMoney},
			{Key: "stock_value", Label: "Stock value", Kind: export.KindMoney},
			{Key: "status", Label: "Status", Kind: export.KindText},
		},
		totals: []string{"stock_value"},
	},
	"employees": {
		title: "Employee Directory", rangeful: false,
		columns: []export.Column{
			{Key: "employee", Label: "Employee", Kind: export.KindText},
			{Key: "position", Label: "Position", Kind: export.KindText},
			{Key: "salary_type", Label: "Salary type", Kind: export.KindText},
			{Key: "rate", Label: "Rate", Kind: export.KindMoney},
			{Key: "hired", Label: "Hired", Kind: export.KindText},
			{Key: "status", Label: "Status", Kind: export.KindText},
		},
	},
	"attendance": {
		title: "Attendance Report", rangeful: true,
		columns: []export.Column{
			{Key: "employee", Label: "Employee", Kind: export.KindText},
			{Key: "date", Label: "Date", Kind: export.KindText},
			{Key: "clock_in", Label: "In", Kind: export.KindText},
			{Key: "clock_out", Label: "Out", Kind: export.KindText},
			{Key: "worked_min", Label: "Worked (min)", Kind: export.KindNum},
			{Key: "late_min", Label: "Late (min)", Kind: export.KindNum},
			{Key: "ot_min", Label: "OT (min)", Kind: export.KindNum},
			{Key: "break_min", Label: "Break (min)", Kind: export.KindNum},
			{Key: "status", Label: "Status", Kind: export.KindText},
		},
		totals: []string{"worked_min", "late_min", "ot_min", "break_min"},
	},
	"profit": {
		title: "Profit Report", rangeful: true,
		columns: []export.Column{
			{Key: "date", Label: "Date", Kind: export.KindText},
			{Key: "gross", Label: "Gross sales", Kind: export.KindMoney},
			{Key: "refunds", Label: "Refunds", Kind: export.KindMoney},
			{Key: "cogs", Label: "COGS", Kind: export.KindMoney},
			{Key: "expenses", Label: "Expenses", Kind: export.KindMoney},
			{Key: "profit", Label: "Profit", Kind: export.KindMoney},
		},
		totals: []string{"gross", "refunds", "cogs", "expenses", "profit"},
	},
	"tax": {
		title: "Tax Report", rangeful: true,
		columns: []export.Column{
			{Key: "date", Label: "Date", Kind: export.KindText},
			{Key: "orders", Label: "Orders", Kind: export.KindNum},
			{Key: "gross_sales", Label: "Gross sales", Kind: export.KindMoney},
			{Key: "net_of_tax", Label: "Net of tax", Kind: export.KindMoney},
			{Key: "tax_collected", Label: "Tax collected", Kind: export.KindMoney},
		},
		totals: []string{"orders", "gross_sales", "net_of_tax", "tax_collected"},
	},
	"receipts": {
		title: "Receipts", rangeful: true,
		columns: []export.Column{
			{Key: "number", Label: "Order #", Kind: export.KindText},
			{Key: "completed", Label: "Completed", Kind: export.KindText},
			{Key: "methods", Label: "Paid via", Kind: export.KindText},
			{Key: "total", Label: "Total", Kind: export.KindMoney},
			{Key: "tendered", Label: "Tendered", Kind: export.KindMoney},
			{Key: "change", Label: "Change", Kind: export.KindMoney},
		},
		totals: []string{"total"},
	},
}

// ReportTypes lists the valid report type keys.
func ReportTypes() []string {
	return []string{"sales", "inventory", "employees", "attendance", "profit", "tax", "receipts"}
}

// ReportService builds export.Documents for the reports center.
type ReportService struct {
	repo      *postgres.ReportRepo
	analytics *AnalyticsService // range parsing in tenant timezone
	tenants   tenant.Repository
	settings  tenant.SettingsRepository
	store     storage.ObjectStorage
	logger    *slog.Logger
}

func NewReportService(repo *postgres.ReportRepo, analytics *AnalyticsService,
	tenants tenant.Repository, settings tenant.SettingsRepository,
	store storage.ObjectStorage, logger *slog.Logger) *ReportService {
	return &ReportService{repo: repo, analytics: analytics, tenants: tenants,
		settings: settings, store: store, logger: logger}
}

// Build assembles the report document; withLogo loads the tenant's PNG
// favicon for PDF headers.
func (s *ReportService) Build(ctx context.Context, tenantID, reportType, fromStr, toStr string, withLogo bool) (*export.Document, error) {
	meta, ok := reportTypes[reportType]
	if !ok {
		return nil, apperror.Validation("unknown report type")
	}

	rng, tz, err := s.analytics.ParseRange(ctx, tenantID, fromStr, toStr)
	if err != nil {
		return nil, err
	}

	var rows []map[string]any
	switch reportType {
	case "sales":
		rows, err = s.repo.Sales(ctx, tenantID, rng.From, rng.To, tz)
	case "inventory":
		rows, err = s.repo.Inventory(ctx, tenantID)
	case "employees":
		rows, err = s.repo.Employees(ctx, tenantID)
	case "attendance":
		rows, err = s.repo.Attendance(ctx, tenantID, rng.From, rng.To, tz)
	case "profit":
		rows, err = s.repo.DailyProfit(ctx, tenantID, rng.From, rng.To, tz)
	case "tax":
		rows, err = s.repo.DailyTax(ctx, tenantID, rng.From, rng.To, tz)
	case "receipts":
		rows, err = s.repo.Receipts(ctx, tenantID, rng.From, rng.To, tz)
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	subtitle := t.Name
	if meta.rangeful {
		subtitle = fmt.Sprintf("%s · %s to %s", t.Name,
			rng.From.Format("Jan 2, 2006"), rng.To.AddDate(0, 0, -1).Format("Jan 2, 2006"))
	} else {
		subtitle = fmt.Sprintf("%s · as of %s", t.Name, time.Now().Format("Jan 2, 2006"))
	}

	doc := &export.Document{
		Title:    meta.title,
		Subtitle: subtitle,
		Columns:  meta.columns,
		Rows:     rows,
	}
	if len(meta.totals) > 0 && len(rows) > 0 {
		totals := map[string]any{doc.Columns[0].Key: "TOTAL"}
		for _, key := range meta.totals {
			var sum int64
			for _, row := range rows {
				if v, ok := row[key]; ok {
					switch n := v.(type) {
					case int64:
						sum += n
					case int:
						sum += int64(n)
					case float64:
						sum += int64(n)
					}
				}
			}
			totals[key] = sum
		}
		doc.Totals = totals
	}

	if withLogo {
		if settings, err := s.settings.GetByTenant(ctx, tenantID); err == nil {
			if key, ok := settings.FaviconKeys["180"]; ok && key != "" {
				if png, err := s.store.Get(ctx, key); err == nil {
					doc.LogoPNG = png
				}
			}
		}
	}
	return doc, nil
}
