package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/billing"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/notification"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/mailer"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/queue"
)

// dueNoticeWindow is how far ahead of the due date the renewal notice
// goes out (user requirement: 3 days).
const dueNoticeWindow = 72 * time.Hour

// HandleBillingSweep runs hourly: (a) sends the 3-day renewal notice to
// owners, (b) deactivates subscriptions past their due date. Both passes
// are idempotent (due_notice_sent_at guard / conditional UPDATE), so
// asynq retries are safe.
func (h *Handlers) HandleBillingSweep(ctx context.Context, _ *asynq.Task) error {
	settings, err := h.Billing.GetPlatformSettings(ctx)
	if err != nil {
		return fmt.Errorf("billing sweep: load prices: %w", err)
	}
	peso := func(c int64) string { return fmt.Sprintf("₱%.2f", float64(c)/100) }

	// (a) Renewal notices.
	due, err := h.Billing.ListDueForNotice(ctx, dueNoticeWindow)
	if err != nil {
		return fmt.Errorf("billing sweep: list due: %w", err)
	}
	for _, d := range due {
		amount := settings.PriceFor(d.Plan)
		dueDate := d.PeriodEnd.In(tenantLocation(d.Timezone)).Format("January 2, 2006")
		title := fmt.Sprintf("Subscription payment due %s", dueDate)
		body := fmt.Sprintf("Your %s plan for %s renews on %s. Pay %s (or switch plans) before then to keep the business active.",
			d.Plan, d.TenantName, dueDate, peso(amount))

		if err := h.notifyOwnerBilling(ctx, d, title, body,
			fmt.Sprintf("Payment due %s — %s", dueDate, d.TenantName)); err != nil {
			h.Logger.Warn("billing notice failed", "tenant", d.TenantID, "error", err)
			continue
		}
		if err := h.Billing.SetNoticeSent(ctx, d.SubscriptionID, time.Now()); err != nil {
			h.Logger.Warn("failed to mark notice sent", "subscription", d.SubscriptionID, "error", err)
		}
		h.Logger.Info("renewal notice sent", "tenant", d.TenantID, "due", d.PeriodEnd)
	}

	// (b) Past-due deactivation.
	overdue, err := h.Billing.DeactivateOverdue(ctx)
	if err != nil {
		return fmt.Errorf("billing sweep: deactivate overdue: %w", err)
	}
	for _, d := range overdue {
		title := fmt.Sprintf("%s is now inactive", d.TenantName)
		body := "The subscription payment wasn't received by the due date. Sign in and pay to reactivate — your data is safe."
		if err := h.notifyOwnerBilling(ctx, d, title, body,
			fmt.Sprintf("Action needed — %s deactivated", d.TenantName)); err != nil {
			h.Logger.Warn("deactivation notice failed", "tenant", d.TenantID, "error", err)
		}
		h.Logger.Info("subscription deactivated", "tenant", d.TenantID, "was_due", d.PeriodEnd)
	}
	return nil
}

// notifyOwnerBilling creates the owner's in-app row and ALWAYS emails —
// billing notices have no opt-out preference.
func (h *Handlers) notifyOwnerBilling(ctx context.Context, d billing.DueSubscription, title, body, subject string) error {
	if err := h.Notifications.CreateForUsers(ctx, d.TenantID, []string{d.OwnerUserID},
		&notification.Notification{Type: notification.TypeBilling, Title: title, Body: body, Link: "/settings/billing"}); err != nil {
		return err
	}
	html := mailer.Render(mailer.Email{
		AppName: h.AppName, Title: title,
		Intro:      fmt.Sprintf("Hi %s, %s", d.OwnerName, body),
		ButtonText: "Open billing", ButtonURL: h.AppURL,
		FooterNote: "You receive billing notices because you own this business.",
	})
	return h.Mailer.Send(d.OwnerEmail, subject, html)
}

func tenantLocation(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}

// RegisterBillingSweep adds the single hourly sweep entry.
func RegisterBillingSweep(scheduler *asynq.Scheduler, logger *slog.Logger) {
	if _, err := scheduler.Register("0 * * * *", asynq.NewTask(queue.TaskBillingSweep, nil)); err != nil {
		logger.Warn("failed to register billing sweep", "error", err)
		return
	}
	logger.Info("billing sweep scheduled", "spec", "0 * * * *")
}
