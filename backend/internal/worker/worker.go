// Package worker hosts the asynq job handlers: transactional email
// delivery, manager notifications, and per-tenant daily summaries.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/analytics"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/notification"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/rbac"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/mailer"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/queue"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
)

// EmailSender delivers mail (SMTP in production, Mailpit in dev).
type EmailSender interface {
	Send(to, subject, htmlBody string) error
}

// Handlers owns the job implementations and their dependencies.
type Handlers struct {
	Tenants       tenant.Repository
	Memberships   tenant.MembershipRepository
	Users         *postgres.UserRepo
	Notifications notification.Repository
	Analytics     analytics.Repository
	Mailer        EmailSender
	AppName       string
	AppURL        string
	Logger        *slog.Logger
}

// brandedEmail renders a notification in the shared legit-looking layout.
func (h *Handlers) brandedEmail(title, intro, extraHTML string) string {
	return mailer.Render(mailer.Email{
		AppName: h.AppName, Title: title, Intro: intro, BodyHTML: extraHTML,
		ButtonText: "Open " + h.AppName, ButtonURL: h.AppURL,
		FooterNote: "You receive these alerts because you manage this business. Adjust them under Notifications in the app.",
	})
}

// Mux registers every task handler.
func (h *Handlers) Mux() *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TaskEmailSend, h.HandleEmail)
	mux.HandleFunc(queue.TaskLowStock, h.HandleLowStock)
	mux.HandleFunc(queue.TaskAttendanceAlert, h.HandleAttendanceAlert)
	mux.HandleFunc(queue.TaskDailySummary, h.HandleDailySummary)
	return mux
}

func (h *Handlers) HandleEmail(ctx context.Context, t *asynq.Task) error {
	var p queue.EmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("bad email payload: %w", err)
	}
	if err := h.Mailer.Send(p.To, p.Subject, p.HTML); err != nil {
		return fmt.Errorf("smtp delivery failed: %w", err)
	}
	h.Logger.Info("email delivered", "to", p.To, "subject", p.Subject)
	return nil
}

// managerRecipients resolves owner+manager members of a tenant.
func (h *Handlers) managerRecipients(ctx context.Context, tenantID string) ([]tenant.Membership, error) {
	members, err := h.Memberships.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var out []tenant.Membership
	for _, m := range members {
		if m.Role == string(rbac.RoleOwner) || m.Role == string(rbac.RoleManager) {
			out = append(out, m)
		}
	}
	return out, nil
}

// notifyManagers creates in-app rows and emails opted-in managers.
func (h *Handlers) notifyManagers(ctx context.Context, tenantID string, n *notification.Notification,
	emailPref func(*notification.Prefs) bool, emailSubject, emailBody string) error {
	recipients, err := h.managerRecipients(ctx, tenantID)
	if err != nil {
		return err
	}
	userIDs := make([]string, len(recipients))
	for i, m := range recipients {
		userIDs[i] = m.UserID
	}
	if err := h.Notifications.CreateForUsers(ctx, tenantID, userIDs, n); err != nil {
		return err
	}

	for _, m := range recipients {
		prefs, err := h.Notifications.GetPrefs(ctx, tenantID, m.UserID)
		if err != nil || !emailPref(prefs) {
			continue
		}
		user, err := h.Users.GetByID(ctx, m.UserID)
		if err != nil {
			continue
		}
		if err := h.Mailer.Send(user.Email, emailSubject, emailBody); err != nil {
			h.Logger.Warn("notification email failed", "to", user.Email, "error", err)
		}
	}
	return nil
}

func (h *Handlers) HandleLowStock(ctx context.Context, t *asynq.Task) error {
	var p queue.LowStockPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("bad low-stock payload: %w", err)
	}
	label := "is running low"
	if p.AlertType == "out_of_stock" {
		label = "is out of stock"
	}
	title := fmt.Sprintf("%s %s", p.ItemName, label)
	body := fmt.Sprintf("Current stock: %g. Review inventory and reorder if needed.", p.Stock)
	return h.notifyManagers(ctx, p.TenantID,
		&notification.Notification{Type: notification.TypeLowStock, Title: title, Body: body, Link: "/inventory"},
		func(pr *notification.Prefs) bool { return pr.EmailLowStock },
		"Stock alert: "+title,
		h.brandedEmail(title, body, ""))
}

func (h *Handlers) HandleAttendanceAlert(ctx context.Context, t *asynq.Task) error {
	var p queue.AttendanceAlertPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("bad attendance payload: %w", err)
	}
	title := fmt.Sprintf("%s clocked in %d minutes late", p.EmployeeName, p.LateMinutes)
	body := fmt.Sprintf("Clock-in at %s.", p.ClockInLocal)
	return h.notifyManagers(ctx, p.TenantID,
		&notification.Notification{Type: notification.TypeAttendance, Title: title, Body: body, Link: "/attendance"},
		func(pr *notification.Prefs) bool { return pr.EmailAttendance },
		"Attendance alert: "+title,
		h.brandedEmail(title, body, ""))
}

func (h *Handlers) HandleDailySummary(ctx context.Context, t *asynq.Task) error {
	var p queue.DailySummaryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("bad daily-summary payload: %w", err)
	}
	ten, err := h.Tenants.GetByID(ctx, p.TenantID)
	if err != nil {
		return err
	}
	loc, err := time.LoadLocation(ten.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	summary, err := h.Analytics.Summary(ctx, p.TenantID, analytics.Range{From: dayStart, To: now})
	if err != nil {
		return err
	}

	peso := func(c int64) string { return fmt.Sprintf("₱%.2f", float64(c)/100) }
	title := fmt.Sprintf("Daily summary — %s in sales", peso(summary.GrossSales))
	body := fmt.Sprintf("%d orders · profit %s · expenses %s · refunds %s",
		summary.Orders, peso(summary.Profit), peso(summary.Expenses), peso(summary.Refunds))
	details := fmt.Sprintf(
		"<ul style=\"margin:0;padding-left:20px;\"><li>Sales: %s (%d orders)</li><li>Profit: %s</li><li>COGS: %s</li><li>Expenses: %s</li><li>Refunds: %s</li></ul>",
		peso(summary.GrossSales), summary.Orders,
		peso(summary.Profit), peso(summary.COGS), peso(summary.Expenses), peso(summary.Refunds))
	html := h.brandedEmail(
		fmt.Sprintf("%s — %s", ten.Name, now.Format("Jan 2, 2006")),
		"Here's how the day went:", details)

	return h.notifyManagers(ctx, p.TenantID,
		&notification.Notification{Type: notification.TypeDailySummary, Title: title, Body: body, Link: "/dashboard"},
		func(pr *notification.Prefs) bool { return pr.EmailDailySummary },
		fmt.Sprintf("%s daily summary — %s", ten.Name, now.Format("Jan 2")), html)
}

// RegisterDailySummaries adds one scheduler entry per tenant, firing at
// 21:00 in each tenant's own timezone.
func RegisterDailySummaries(ctx context.Context, scheduler *asynq.Scheduler, tenants tenant.Repository, logger *slog.Logger) error {
	list, _, err := tenants.List(ctx, 1000, 0)
	if err != nil {
		return err
	}
	for _, t := range list {
		payload, _ := json.Marshal(queue.DailySummaryPayload{TenantID: t.ID})
		spec := fmt.Sprintf("CRON_TZ=%s 0 21 * * *", t.Timezone)
		if _, err := scheduler.Register(spec, asynq.NewTask(queue.TaskDailySummary, payload)); err != nil {
			logger.Warn("failed to register daily summary", "tenant", t.Slug, "error", err)
			continue
		}
		logger.Info("daily summary scheduled", "tenant", t.Slug, "spec", spec)
	}
	return nil
}
