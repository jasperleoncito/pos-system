package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

type MembershipRepo struct {
	db *pgxpool.Pool
}

func NewMembershipRepo(db *pgxpool.Pool) *MembershipRepo { return &MembershipRepo{db: db} }

func (r *MembershipRepo) Create(ctx context.Context, m *tenant.Membership) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO tenant_users (tenant_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING id`,
		m.TenantID, m.UserID, m.Role,
	).Scan(&m.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.Conflict("user already belongs to this business")
		}
		return fmt.Errorf("failed to create membership: %w", err)
	}
	return nil
}

func (r *MembershipRepo) Get(ctx context.Context, tenantID, userID string) (*tenant.Membership, error) {
	var m tenant.Membership
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, role FROM tenant_users
		WHERE tenant_id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		tenantID, userID,
	).Scan(&m.ID, &m.TenantID, &m.UserID, &m.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("membership")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get membership: %w", err)
	}
	return &m, nil
}

func (r *MembershipRepo) ListByUser(ctx context.Context, userID string) ([]tenant.Membership, error) {
	rows, err := r.db.Query(ctx, `
		SELECT tu.id, tu.tenant_id, tu.user_id, tu.role, t.name, t.slug
		FROM tenant_users tu
		JOIN tenants t ON t.id = tu.tenant_id AND t.deleted_at IS NULL
		WHERE tu.user_id = $1 AND tu.deleted_at IS NULL
		ORDER BY t.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list memberships: %w", err)
	}
	defer rows.Close()

	var memberships []tenant.Membership
	for rows.Next() {
		var m tenant.Membership
		if err := rows.Scan(&m.ID, &m.TenantID, &m.UserID, &m.Role, &m.TenantName, &m.TenantSlug); err != nil {
			return nil, fmt.Errorf("failed to scan membership: %w", err)
		}
		memberships = append(memberships, m)
	}
	return memberships, rows.Err()
}

func (r *MembershipRepo) ListByTenant(ctx context.Context, tenantID string) ([]tenant.Membership, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, user_id, role FROM tenant_users
		WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant members: %w", err)
	}
	defer rows.Close()

	var memberships []tenant.Membership
	for rows.Next() {
		var m tenant.Membership
		if err := rows.Scan(&m.ID, &m.TenantID, &m.UserID, &m.Role); err != nil {
			return nil, fmt.Errorf("failed to scan membership: %w", err)
		}
		memberships = append(memberships, m)
	}
	return memberships, rows.Err()
}

func (r *MembershipRepo) UpdateRole(ctx context.Context, tenantID, userID, role string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tenant_users SET role = $3, updated_at = now()
		WHERE tenant_id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		tenantID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to update membership role: %w", err)
	}
	return nil
}

func (r *MembershipRepo) Delete(ctx context.Context, tenantID, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tenant_users SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete membership: %w", err)
	}
	return nil
}
