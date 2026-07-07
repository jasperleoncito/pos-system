package service

import (
	"context"
	"strings"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/promo"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// PromoService owns discounts and coupons.
type PromoService struct {
	discounts promo.DiscountRepository
	coupons   promo.CouponRepository
	auditor   *AuditService
}

func NewPromoService(discounts promo.DiscountRepository, coupons promo.CouponRepository, auditor *AuditService) *PromoService {
	return &PromoService{discounts: discounts, coupons: coupons, auditor: auditor}
}

// ---- discounts ----

func (s *PromoService) CreateDiscount(ctx context.Context, tenantID, userID string, d *promo.Discount) error {
	if err := validatePromoValues(d.Type, d.PercentValue, d.AmountValue); err != nil {
		return err
	}
	if err := s.discounts.Create(ctx, tenantID, d); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "promo.discount_created",
		EntityType: "discount", EntityID: d.ID, After: map[string]any{"name": d.Name},
	})
	return nil
}

func (s *PromoService) ListDiscounts(ctx context.Context, tenantID string) ([]promo.Discount, error) {
	return s.discounts.List(ctx, tenantID)
}

func (s *PromoService) UpdateDiscount(ctx context.Context, tenantID, userID string, d *promo.Discount) error {
	if err := validatePromoValues(d.Type, d.PercentValue, d.AmountValue); err != nil {
		return err
	}
	if err := s.discounts.Update(ctx, tenantID, d); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "promo.discount_updated",
		EntityType: "discount", EntityID: d.ID, After: map[string]any{"name": d.Name},
	})
	return nil
}

func (s *PromoService) DeleteDiscount(ctx context.Context, tenantID, userID, id string) error {
	if err := s.discounts.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "promo.discount_deleted",
		EntityType: "discount", EntityID: id,
	})
	return nil
}

// ---- coupons ----

func (s *PromoService) CreateCoupon(ctx context.Context, tenantID, userID string, c *promo.Coupon) error {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	if err := validatePromoValues(c.DiscountType, c.PercentValue, c.AmountValue); err != nil {
		return err
	}
	if err := s.coupons.Create(ctx, tenantID, c); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "promo.coupon_created",
		EntityType: "coupon", EntityID: c.ID, After: map[string]any{"code": c.Code},
	})
	return nil
}

func (s *PromoService) ListCoupons(ctx context.Context, tenantID string) ([]promo.Coupon, error) {
	return s.coupons.List(ctx, tenantID)
}

func (s *PromoService) UpdateCoupon(ctx context.Context, tenantID, userID string, c *promo.Coupon) error {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	if err := validatePromoValues(c.DiscountType, c.PercentValue, c.AmountValue); err != nil {
		return err
	}
	if err := s.coupons.Update(ctx, tenantID, c); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "promo.coupon_updated",
		EntityType: "coupon", EntityID: c.ID, After: map[string]any{"code": c.Code},
	})
	return nil
}

func (s *PromoService) DeleteCoupon(ctx context.Context, tenantID, userID, id string) error {
	if err := s.coupons.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "promo.coupon_deleted",
		EntityType: "coupon", EntityID: id,
	})
	return nil
}

// ValidateCoupon checks a code against a subtotal without redeeming it.
func (s *PromoService) ValidateCoupon(ctx context.Context, tenantID, code string, subtotal int64) (*promo.Coupon, int64, error) {
	coupon, err := s.coupons.GetByCode(ctx, tenantID, code)
	if err != nil {
		return nil, 0, apperror.Validation("coupon code not found")
	}
	if err := CheckCouponUsable(coupon, subtotal, time.Now()); err != nil {
		return nil, 0, err
	}
	discount := promo.Apply(coupon.DiscountType, coupon.PercentValue, coupon.AmountValue, subtotal)
	return coupon, discount, nil
}

// CheckCouponUsable enforces activity, validity window, min order, and
// remaining uses.
func CheckCouponUsable(c *promo.Coupon, subtotal int64, now time.Time) error {
	if !c.IsActive {
		return apperror.Validation("this coupon is no longer active")
	}
	if c.ValidFrom != nil && now.Before(*c.ValidFrom) {
		return apperror.Validation("this coupon is not valid yet")
	}
	if c.ValidTo != nil && now.After(*c.ValidTo) {
		return apperror.Validation("this coupon has expired")
	}
	if c.MaxUses > 0 && c.UsesCount >= c.MaxUses {
		return apperror.Validation("this coupon has reached its usage limit")
	}
	if subtotal < c.MinOrderAmount {
		return apperror.Validation("order does not meet the coupon minimum")
	}
	return nil
}

func validatePromoValues(promoType string, percentValue float64, amountValue int64) error {
	if !promo.ValidType(promoType) {
		return apperror.Validation("type must be percent or fixed")
	}
	if promoType == promo.TypePercent && (percentValue <= 0 || percentValue > 100) {
		return apperror.Validation("percent must be between 0 and 100")
	}
	if promoType == promo.TypeFixed && amountValue <= 0 {
		return apperror.Validation("fixed amount must be positive")
	}
	return nil
}
