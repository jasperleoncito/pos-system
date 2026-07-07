package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
)

const auditInsertTimeout = 5 * time.Second

// AuditService records actions to the append-only audit trail. Failures
// are logged, never propagated — auditing must not break business flows.
type AuditService struct {
	repo   audit.Repository
	logger *slog.Logger
}

func NewAuditService(repo audit.Repository, logger *slog.Logger) *AuditService {
	return &AuditService{repo: repo, logger: logger}
}

// Record inserts an audit entry detached from the request context so the
// write survives request cancellation.
func (s *AuditService) Record(entry audit.Log) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), auditInsertTimeout)
		defer cancel()
		if err := s.repo.Insert(ctx, &entry); err != nil {
			s.logger.Error("failed to record audit log", "action", entry.Action, "error", err)
		}
	}()
}

// List returns audit entries for a tenant.
func (s *AuditService) List(ctx context.Context, tenantID string, limit, offset int) ([]audit.Log, int64, error) {
	return s.repo.List(ctx, tenantID, limit, offset)
}
