package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/notification"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type NotificationRepo struct {
	db *pgxpool.Pool
}

func NewNotificationRepo(db *pgxpool.Pool) *NotificationRepo { return &NotificationRepo{db: db} }

func (r *NotificationRepo) CreateForUsers(ctx context.Context, tenantID string, userIDs []string, n *notification.Notification) error {
	if len(userIDs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, userID := range userIDs {
		batch.Queue(`
			INSERT INTO notifications (tenant_id, user_id, type, title, body, link)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			tenantID, userID, n.Type, n.Title, n.Body, n.Link)
	}
	results := r.db.SendBatch(ctx, batch)
	defer results.Close()
	for range userIDs {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("failed to insert notification: %w", err)
		}
	}
	return nil
}

func (r *NotificationRepo) List(ctx context.Context, tenantID, userID string, limit int) ([]notification.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, type, title, body, link, read_at, created_at
		FROM notifications
		WHERE tenant_id=$1 AND user_id=$2
		ORDER BY created_at DESC LIMIT $3`, tenantID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()
	var out []notification.Notification
	for rows.Next() {
		var n notification.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Link, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, tenantID, userID string) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE tenant_id=$1 AND user_id=$2 AND read_at IS NULL`, tenantID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread: %w", err)
	}
	return count, nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, tenantID, userID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at=now()
		WHERE tenant_id=$1 AND user_id=$2 AND id=$3 AND read_at IS NULL`, tenantID, userID, id)
	if err != nil {
		return fmt.Errorf("failed to mark read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("notification")
	}
	return nil
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at=now()
		WHERE tenant_id=$1 AND user_id=$2 AND read_at IS NULL`, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark all read: %w", err)
	}
	return nil
}

func (r *NotificationRepo) GetPrefs(ctx context.Context, tenantID, userID string) (*notification.Prefs, error) {
	var p notification.Prefs
	err := r.db.QueryRow(ctx, `
		SELECT email_low_stock, email_attendance, email_daily_summary
		FROM notification_prefs WHERE tenant_id=$1 AND user_id=$2`, tenantID, userID,
	).Scan(&p.EmailLowStock, &p.EmailAttendance, &p.EmailDailySummary)
	if errors.Is(err, pgx.ErrNoRows) {
		return &notification.Prefs{EmailLowStock: true, EmailAttendance: true, EmailDailySummary: true}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get prefs: %w", err)
	}
	return &p, nil
}

func (r *NotificationRepo) SavePrefs(ctx context.Context, tenantID, userID string, p *notification.Prefs) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO notification_prefs (tenant_id, user_id, email_low_stock, email_attendance, email_daily_summary)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, user_id) DO UPDATE SET
			email_low_stock=$3, email_attendance=$4, email_daily_summary=$5, updated_at=now()`,
		tenantID, userID, p.EmailLowStock, p.EmailAttendance, p.EmailDailySummary)
	if err != nil {
		return fmt.Errorf("failed to save prefs: %w", err)
	}
	return nil
}
