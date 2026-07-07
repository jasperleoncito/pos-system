// Package notification defines in-app notifications and email prefs.
package notification

import (
	"context"
	"time"
)

// Notification types.
const (
	TypeLowStock     = "low_stock"
	TypeAttendance   = "attendance"
	TypeDailySummary = "daily_summary"
	TypeSystem       = "system"
)

type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Link      string     `json:"link"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// Prefs are per-user email switches; in-app delivery is always on.
type Prefs struct {
	EmailLowStock     bool `json:"email_low_stock"`
	EmailAttendance   bool `json:"email_attendance"`
	EmailDailySummary bool `json:"email_daily_summary"`
}

type Repository interface {
	// CreateForUsers fans one notification out to many users.
	CreateForUsers(ctx context.Context, tenantID string, userIDs []string, n *Notification) error
	List(ctx context.Context, tenantID, userID string, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, tenantID, userID string) (int64, error)
	MarkRead(ctx context.Context, tenantID, userID, id string) error
	MarkAllRead(ctx context.Context, tenantID, userID string) error

	GetPrefs(ctx context.Context, tenantID, userID string) (*Prefs, error)
	SavePrefs(ctx context.Context, tenantID, userID string, p *Prefs) error
}
