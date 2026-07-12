package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/storage"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/imageproc"
)

// TenantService owns tenant settings, branding assets, and the
// super-admin tenant lifecycle.
type TenantService struct {
	tenants  tenant.Repository
	settings tenant.SettingsRepository
	store    storage.ObjectStorage
	auditor  *AuditService
	logger   *slog.Logger
}

func NewTenantService(
	tenants tenant.Repository,
	settings tenant.SettingsRepository,
	store storage.ObjectStorage,
	auditor *AuditService,
	logger *slog.Logger,
) *TenantService {
	return &TenantService{tenants: tenants, settings: settings, store: store, auditor: auditor, logger: logger}
}

// SettingsView augments stored settings with resolved public URLs.
type SettingsView struct {
	*tenant.Settings
	LogoURL      string            `json:"logo_url"`
	LogoThumbURL string            `json:"logo_thumb_url"`
	FaviconURLs  map[string]string `json:"favicon_urls"`
}

func (s *TenantService) view(settings *tenant.Settings) *SettingsView {
	faviconURLs := make(map[string]string, len(settings.FaviconKeys))
	for size, key := range settings.FaviconKeys {
		faviconURLs[size] = s.store.PublicURL(key)
	}
	return &SettingsView{
		Settings:     settings,
		LogoURL:      s.store.PublicURL(settings.LogoKey),
		LogoThumbURL: s.store.PublicURL(settings.LogoThumbKey),
		FaviconURLs:  faviconURLs,
	}
}

func (s *TenantService) GetSettings(ctx context.Context, tenantID string) (*SettingsView, error) {
	settings, err := s.settings.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return s.view(settings), nil
}

// UpdateSettingsInput carries the editable branding fields.
type UpdateSettingsInput struct {
	PrimaryColor   string
	SecondaryColor string
	AccentColor    string
	ReceiptHeader  string
	ReceiptFooter  string
	ContactNumber  string
	Facebook       string
	Website        string
	Address        string
	TaxLabel       string
	TaxID          string
}

func (s *TenantService) UpdateSettings(ctx context.Context, tenantID, userID string, in UpdateSettingsInput) (*SettingsView, error) {
	current, err := s.settings.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	updated := *current
	updated.PrimaryColor = in.PrimaryColor
	updated.SecondaryColor = in.SecondaryColor
	updated.AccentColor = in.AccentColor
	updated.ReceiptHeader = in.ReceiptHeader
	updated.ReceiptFooter = in.ReceiptFooter
	updated.ContactNumber = in.ContactNumber
	updated.Facebook = in.Facebook
	updated.Website = in.Website
	updated.Address = in.Address
	updated.TaxLabel = in.TaxLabel
	updated.TaxID = in.TaxID

	if err := s.settings.Update(ctx, &updated); err != nil {
		return nil, err
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "tenant.settings_updated",
		EntityType: "tenant_settings", EntityID: updated.ID,
		Before: map[string]any{"primary_color": current.PrimaryColor, "secondary_color": current.SecondaryColor, "accent_color": current.AccentColor},
		After:  map[string]any{"primary_color": updated.PrimaryColor, "secondary_color": updated.SecondaryColor, "accent_color": updated.AccentColor},
	})
	return s.view(&updated), nil
}

// UploadLogo optimizes and stores the tenant logo, thumbnail, and
// favicon set. Only optimized outputs are stored — never the original.
func (s *TenantService) UploadLogo(ctx context.Context, tenantID, userID string, data []byte) (*SettingsView, error) {
	settings, err := s.settings.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	result, err := imageproc.Optimize(data)
	if err != nil {
		return nil, err
	}
	img, err := imageproc.Decode(data)
	if err != nil {
		return nil, err
	}
	favicons, err := imageproc.Favicons(img)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "failed to generate favicons", err)
	}

	// Timestamped names bust browser caches on re-upload.
	version := time.Now().Unix()
	logoKey := storage.TenantKey(tenantID, storage.FolderLogos, fmt.Sprintf("logo-%d.webp", version))
	thumbKey := storage.TenantKey(tenantID, storage.FolderLogos, fmt.Sprintf("logo-%d-thumb.webp", version))

	if err := s.store.Put(ctx, logoKey, result.WebP, "image/webp"); err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.store.Put(ctx, thumbKey, result.ThumbWebP, "image/webp"); err != nil {
		return nil, apperror.Internal(err)
	}

	faviconKeys := make(map[string]string, len(favicons))
	for size, pngBytes := range favicons {
		key := storage.TenantKey(tenantID, storage.FolderLogos, fmt.Sprintf("favicon-%d-%dx%d.png", version, size, size))
		if err := s.store.Put(ctx, key, pngBytes, "image/png"); err != nil {
			return nil, apperror.Internal(err)
		}
		faviconKeys[strconv.Itoa(size)] = key
	}

	// Best-effort cleanup of the previous generation.
	s.deleteOldLogoObjects(settings)

	settings.LogoKey = logoKey
	settings.LogoThumbKey = thumbKey
	settings.FaviconKeys = faviconKeys
	if err := s.settings.Update(ctx, settings); err != nil {
		return nil, err
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "tenant.logo_uploaded",
		EntityType: "tenant_settings", EntityID: settings.ID,
		After: map[string]any{"logo_key": logoKey, "bytes": len(result.WebP)},
	})
	return s.view(settings), nil
}

func (s *TenantService) deleteOldLogoObjects(settings *tenant.Settings) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	keys := make([]string, 0, 2+len(settings.FaviconKeys))
	if settings.LogoKey != "" {
		keys = append(keys, settings.LogoKey)
	}
	if settings.LogoThumbKey != "" {
		keys = append(keys, settings.LogoThumbKey)
	}
	for _, key := range settings.FaviconKeys {
		keys = append(keys, key)
	}
	for _, key := range keys {
		if err := s.store.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to delete old logo object", "key", key, "error", err)
		}
	}
}

// ---- super-admin tenant lifecycle ----

func (s *TenantService) ListTenants(ctx context.Context, limit, offset int) ([]tenant.Tenant, int64, error) {
	return s.tenants.List(ctx, limit, offset)
}

func (s *TenantService) SetTenantStatus(ctx context.Context, actorID, tenantID, status string) (*tenant.Tenant, error) {
	if status != "active" && status != "suspended" {
		return nil, apperror.Validation("status must be active or suspended")
	}
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	before := t.Status
	t.Status = status
	if err := s.tenants.Update(ctx, t); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: actorID, Action: "tenant.status_changed",
		EntityType: "tenant", EntityID: tenantID,
		Before: map[string]any{"status": before}, After: map[string]any{"status": status},
	})
	return t, nil
}

// SetTenantPlan updates a tenant's subscription plan (platform admin).
func (s *TenantService) SetTenantPlan(ctx context.Context, actorID, tenantID, plan string) (*tenant.Tenant, error) {
	if plan != "free" && plan != "standard" && plan != "premium" {
		return nil, apperror.Validation("plan must be free, standard, or premium")
	}
	if err := s.tenants.SetPlan(ctx, tenantID, plan); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: actorID, Action: "tenant.plan_changed",
		EntityType: "tenant", EntityID: tenantID, After: map[string]any{"plan": plan},
	})
	return s.tenants.GetByID(ctx, tenantID)
}

// PlatformStats surfaces cross-tenant counters for the admin console.
func (s *TenantService) PlatformStats(ctx context.Context) (map[string]any, error) {
	stats, err := s.tenants.PlatformStats(ctx)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return stats, nil
}

// PlatformSales returns platform-wide sales analytics for the last N days
// (clamped to 1..365, default 30).
func (s *TenantService) PlatformSales(ctx context.Context, days int) (*tenant.PlatformSales, error) {
	if days < 1 || days > 365 {
		days = 30
	}
	sales, err := s.tenants.PlatformSales(ctx, days)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return sales, nil
}
