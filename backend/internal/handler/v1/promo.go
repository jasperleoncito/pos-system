package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/promo"
	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// PromoHandler exposes discount and coupon management.
type PromoHandler struct {
	promos *service.PromoService
}

func NewPromoHandler(promos *service.PromoService) *PromoHandler {
	return &PromoHandler{promos: promos}
}

// ---- discounts ----

// ListDiscounts godoc
//
//	@Summary	List discounts
//	@Tags		promos
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/discounts [get]
func (h *PromoHandler) ListDiscounts(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	discounts, err := h.promos.ListDiscounts(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", discounts)
}

func discountFromRequest(req dto.DiscountRequest) *promo.Discount {
	return &promo.Discount{
		Name: req.Name, Type: req.Type, PercentValue: req.PercentValue,
		AmountValue: req.AmountValue, RequiresApproval: req.RequiresApproval,
		IsActive: boolOrDefault(req.IsActive, true),
	}
}

// CreateDiscount godoc
//
//	@Summary	Create a discount
//	@Tags		promos
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.DiscountRequest	true	"Discount"
//	@Success	201		{object}	response.Envelope
//	@Router		/discounts [post]
func (h *PromoHandler) CreateDiscount(c *gin.Context) {
	var req dto.DiscountRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	discount := discountFromRequest(req)
	if err := h.promos.CreateDiscount(c.Request.Context(), tenantID, userID, discount); err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "discount created", discount)
}

// UpdateDiscount godoc
//
//	@Summary	Update a discount
//	@Tags		promos
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Discount ID"
//	@Param		payload	body		dto.DiscountRequest	true	"Discount"
//	@Success	200		{object}	response.Envelope
//	@Router		/discounts/{id} [put]
func (h *PromoHandler) UpdateDiscount(c *gin.Context) {
	var req dto.DiscountRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	discount := discountFromRequest(req)
	discount.ID = c.Param("id")
	if err := h.promos.UpdateDiscount(c.Request.Context(), tenantID, userID, discount); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "discount updated", discount)
}

// DeleteDiscount godoc
//
//	@Summary	Delete a discount
//	@Tags		promos
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Discount ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/discounts/{id} [delete]
func (h *PromoHandler) DeleteDiscount(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.promos.DeleteDiscount(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "discount deleted", nil)
}

// ---- coupons ----

// ListCoupons godoc
//
//	@Summary	List coupons
//	@Tags		promos
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/coupons [get]
func (h *PromoHandler) ListCoupons(c *gin.Context) {
	tenantID, _ := tenantUser(c)
	coupons, err := h.promos.ListCoupons(c.Request.Context(), tenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", coupons)
}

func couponFromRequest(req dto.CouponRequest) *promo.Coupon {
	return &promo.Coupon{
		Code: req.Code, DiscountType: req.DiscountType, PercentValue: req.PercentValue,
		AmountValue: req.AmountValue, MinOrderAmount: req.MinOrderAmount, MaxUses: req.MaxUses,
		ValidFrom: req.ValidFrom, ValidTo: req.ValidTo, IsActive: boolOrDefault(req.IsActive, true),
	}
}

// CreateCoupon godoc
//
//	@Summary	Create a coupon
//	@Tags		promos
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.CouponRequest	true	"Coupon"
//	@Success	201		{object}	response.Envelope
//	@Router		/coupons [post]
func (h *PromoHandler) CreateCoupon(c *gin.Context) {
	var req dto.CouponRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	coupon := couponFromRequest(req)
	if err := h.promos.CreateCoupon(c.Request.Context(), tenantID, userID, coupon); err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "coupon created", coupon)
}

// UpdateCoupon godoc
//
//	@Summary	Update a coupon
//	@Tags		promos
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string				true	"Coupon ID"
//	@Param		payload	body		dto.CouponRequest	true	"Coupon"
//	@Success	200		{object}	response.Envelope
//	@Router		/coupons/{id} [put]
func (h *PromoHandler) UpdateCoupon(c *gin.Context) {
	var req dto.CouponRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, userID := tenantUser(c)
	coupon := couponFromRequest(req)
	coupon.ID = c.Param("id")
	if err := h.promos.UpdateCoupon(c.Request.Context(), tenantID, userID, coupon); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "coupon updated", coupon)
}

// DeleteCoupon godoc
//
//	@Summary	Delete a coupon
//	@Tags		promos
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Coupon ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/coupons/{id} [delete]
func (h *PromoHandler) DeleteCoupon(c *gin.Context) {
	tenantID, userID := tenantUser(c)
	if err := h.promos.DeleteCoupon(c.Request.Context(), tenantID, userID, c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "coupon deleted", nil)
}

// ValidateCoupon godoc
//
//	@Summary	Validate a coupon code against a subtotal (no redemption)
//	@Tags		promos
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.ValidateCouponRequest	true	"Code and subtotal"
//	@Success	200		{object}	response.Envelope
//	@Failure	422		{object}	response.ErrorEnvelope
//	@Router		/coupons/validate [post]
func (h *PromoHandler) ValidateCoupon(c *gin.Context) {
	var req dto.ValidateCouponRequest
	if !bindJSON(c, &req) {
		return
	}
	tenantID, _ := tenantUser(c)
	coupon, discount, err := h.promos.ValidateCoupon(c.Request.Context(), tenantID, req.Code, req.Subtotal)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "coupon is valid", gin.H{"coupon": coupon, "discount": discount})
}
