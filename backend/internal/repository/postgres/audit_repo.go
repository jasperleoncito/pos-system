package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
)

type AuditRepo struct {
	db *pgxpool.Pool
}

func NewAuditRepo(db *pgxpool.Pool) *AuditRepo { return &AuditRepo{db: db} }

func (r *AuditRepo) Insert(ctx context.Context, l *audit.Log) error {
	var tenantID, userID any
	if l.TenantID != "" {
		tenantID = l.TenantID
	}
	if l.UserID != "" {
		userID = l.UserID
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO audit_logs (tenant_id, user_id, action, entity_type, entity_id, before, after, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		tenantID, userID, l.Action, l.EntityType, l.EntityID, l.Before, l.After, l.IP, l.UserAgent)
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}
	return nil
}

func (r *AuditRepo) List(ctx context.Context, tenantID string, limit, offset int) ([]audit.Log, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE tenant_id = $1`, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT a.id, coalesce(a.tenant_id::text, ''), coalesce(a.user_id::text, ''),
		       coalesce(u.full_name, ''), a.action, a.entity_type, a.entity_id,
		       a.before, a.after, a.ip, a.user_agent, a.created_at
		FROM audit_logs a LEFT JOIN users u ON u.id = a.user_id
		WHERE a.tenant_id = $1
		ORDER BY a.created_at DESC LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []audit.Log
	for rows.Next() {
		var l audit.Log
		if err := rows.Scan(&l.ID, &l.TenantID, &l.UserID, &l.UserName, &l.Action, &l.EntityType, &l.EntityID,
			&l.Before, &l.After, &l.IP, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}
