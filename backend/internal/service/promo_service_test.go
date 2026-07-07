package service

import (
	"testing"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/promo"
)

func timePtr(t time.Time) *time.Time { return &t }

func TestCheckCouponUsable(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	base := promo.Coupon{
		Code: "SAVE10", DiscountType: promo.TypePercent, PercentValue: 10,
		IsActive: true,
	}

	tests := []struct {
		name     string
		mutate   func(c *promo.Coupon)
		subtotal int64
		wantErr  bool
	}{
		{"valid coupon", func(c *promo.Coupon) {}, 10000, false},
		{"inactive", func(c *promo.Coupon) { c.IsActive = false }, 10000, true},
		{"not started yet", func(c *promo.Coupon) { c.ValidFrom = timePtr(now.Add(time.Hour)) }, 10000, true},
		{"expired", func(c *promo.Coupon) { c.ValidTo = timePtr(now.Add(-time.Hour)) }, 10000, true},
		{"within window", func(c *promo.Coupon) {
			c.ValidFrom = timePtr(now.Add(-time.Hour))
			c.ValidTo = timePtr(now.Add(time.Hour))
		}, 10000, false},
		{"exhausted", func(c *promo.Coupon) { c.MaxUses = 5; c.UsesCount = 5 }, 10000, true},
		{"uses remaining", func(c *promo.Coupon) { c.MaxUses = 5; c.UsesCount = 4 }, 10000, false},
		{"unlimited uses", func(c *promo.Coupon) { c.MaxUses = 0; c.UsesCount = 999 }, 10000, false},
		{"below minimum", func(c *promo.Coupon) { c.MinOrderAmount = 20000 }, 10000, true},
		{"meets minimum", func(c *promo.Coupon) { c.MinOrderAmount = 10000 }, 10000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coupon := base
			tt.mutate(&coupon)
			err := CheckCouponUsable(&coupon, tt.subtotal, now)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCouponUsable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPromoApply(t *testing.T) {
	tests := []struct {
		name         string
		promoType    string
		percent      float64
		amount       int64
		subtotal     int64
		want         int64
	}{
		{"10 percent of php184", promo.TypePercent, 10, 0, 18400, 1840},
		{"percent rounds half up", promo.TypePercent, 12.5, 0, 999, 125}, // 124.875 → 125
		{"fixed php50", promo.TypeFixed, 0, 5000, 18400, 5000},
		{"fixed capped at subtotal", promo.TypeFixed, 0, 99999, 5000, 5000},
		{"100 percent", promo.TypePercent, 100, 0, 5000, 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := promo.Apply(tt.promoType, tt.percent, tt.amount, tt.subtotal); got != tt.want {
				t.Errorf("Apply() = %d, want %d", got, tt.want)
			}
		})
	}
}
