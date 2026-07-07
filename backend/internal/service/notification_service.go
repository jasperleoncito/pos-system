package service

import (
	"context"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/notification"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/queue"
)

// Jobs enqueues background work; the API never blocks on delivery.
type Jobs interface {
	EnqueueLowStock(p queue.LowStockPayload) error
	EnqueueAttendanceAlert(p queue.AttendanceAlertPayload) error
}

// NotificationService serves the bell dropdown and preferences.
type NotificationService struct {
	repo notification.Repository
}

func NewNotificationService(repo notification.Repository) *NotificationService {
	return &NotificationService{repo: repo}
}

// Feed is the bell payload: recent items plus the unread count.
type Feed struct {
	Items  []notification.Notification `json:"items"`
	Unread int64                       `json:"unread"`
}

func (s *NotificationService) Feed(ctx context.Context, tenantID, userID string, limit int) (*Feed, error) {
	items, err := s.repo.List(ctx, tenantID, userID, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	unread, err := s.repo.UnreadCount(ctx, tenantID, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &Feed{Items: items, Unread: unread}, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, tenantID, userID, id string) error {
	return s.repo.MarkRead(ctx, tenantID, userID, id)
}

func (s *NotificationService) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	if err := s.repo.MarkAllRead(ctx, tenantID, userID); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (s *NotificationService) GetPrefs(ctx context.Context, tenantID, userID string) (*notification.Prefs, error) {
	return s.repo.GetPrefs(ctx, tenantID, userID)
}

func (s *NotificationService) SavePrefs(ctx context.Context, tenantID, userID string, p *notification.Prefs) (*notification.Prefs, error) {
	if err := s.repo.SavePrefs(ctx, tenantID, userID, p); err != nil {
		return nil, apperror.Internal(err)
	}
	return s.repo.GetPrefs(ctx, tenantID, userID)
}
