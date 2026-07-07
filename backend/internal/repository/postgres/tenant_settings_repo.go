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

type TenantSettingsRepo struct {
	db *pgxpool.Pool
}

func NewTenantSettingsRepo(db *pgxpool.Pool) *TenantSettingsRepo {
	return &TenantSettingsRepo{db: db}
}

func (r *TenantSettingsRepo) Create(ctx context.Context, s *tenant.Settings) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO tenant_settings (tenant_id, primary_color, secondary_color, accent_color,
			receipt_header, receipt_footer, contact_number, facebook, website, address, tax_label, tax_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, updated_at`,
		s.TenantID, s.PrimaryColor, s.SecondaryColor, s.AccentColor,
		s.ReceiptHeader, s.ReceiptFooter, s.ContactNumber, s.Facebook, s.Website, s.Address, s.TaxLabel, s.TaxID,
	).Scan(&s.ID, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create tenant settings: %w", err)
	}
	return nil
}

func (r *TenantSettingsRepo) GetByTenant(ctx context.Context, tenantID string) (*tenant.Settings, error) {
	var s tenant.Settings
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, logo_key, logo_thumb_key, favicon_keys, primary_color, secondary_color,
		       accent_color, receipt_header, receipt_footer, contact_number, facebook, website, address,
		       tax_label, tax_id, updated_at
		FROM tenant_settings WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID,
	).Scan(&s.ID, &s.TenantID, &s.LogoKey, &s.LogoThumbKey, &s.FaviconKeys, &s.PrimaryColor, &s.SecondaryColor,
		&s.AccentColor, &s.ReceiptHeader, &s.ReceiptFooter, &s.ContactNumber, &s.Facebook, &s.Website, &s.Address,
		&s.TaxLabel, &s.TaxID, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NotFound("tenant settings")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant settings: %w", err)
	}
	return &s, nil
}

func (r *TenantSettingsRepo) Update(ctx context.Context, s *tenant.Settings) error {
	_, err := r.db.Exec(ctx, `
		UPDATE tenant_settings SET
			logo_key = $2, logo_thumb_key = $3, favicon_keys = $4,
			primary_color = $5, secondary_color = $6, accent_color = $7,
			receipt_header = $8, receipt_footer = $9, contact_number = $10,
			facebook = $11, website = $12, address = $13, tax_label = $14, tax_id = $15,
			updated_at = now()
		WHERE tenant_id = $1 AND deleted_at IS NULL`,
		s.TenantID, s.LogoKey, s.LogoThumbKey, s.FaviconKeys,
		s.PrimaryColor, s.SecondaryColor, s.AccentColor,
		s.ReceiptHeader, s.ReceiptFooter, s.ContactNumber,
		s.Facebook, s.Website, s.Address, s.TaxLabel, s.TaxID)
	if err != nil {
		return fmt.Errorf("failed to update tenant settings: %w", err)
	}
	return nil
}
