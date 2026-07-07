package service

import (
	"context"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/order"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/promo"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// applyPromo applies an order-level discount and/or coupon to a freshly
// created order, redeeming the coupon atomically.
func (s *OrderService) applyPromo(ctx context.Context, tenantID string, o *order.Order, discountID, couponCode string) error {
	var totalDiscount int64
	var discountRef, couponRef *string

	if discountID != "" {
		d, err := s.discounts.GetByID(ctx, tenantID, discountID)
		if err != nil {
			return apperror.Validation("discount not found")
		}
		if !d.IsActive {
			return apperror.Validation("this discount is no longer active")
		}
		totalDiscount += promo.Apply(d.Type, d.PercentValue, d.AmountValue, o.Subtotal)
		discountRef = &d.ID
	}

	if couponCode != "" {
		c, err := s.coupons.GetByCode(ctx, tenantID, couponCode)
		if err != nil {
			return apperror.Validation("coupon code not found")
		}
		if err := CheckCouponUsable(c, o.Subtotal, time.Now()); err != nil {
			return err
		}
		redeemed, err := s.coupons.Redeem(ctx, tenantID, c.ID, o.ID)
		if err != nil {
			return apperror.Internal(err)
		}
		if !redeemed {
			return apperror.Validation("this coupon has reached its usage limit")
		}
		remaining := o.Subtotal - totalDiscount
		totalDiscount += promo.Apply(c.DiscountType, c.PercentValue, c.AmountValue, remaining)
		couponRef = &c.ID
	}

	o.DiscountTotal = totalDiscount
	o.Total = o.Subtotal - totalDiscount
	o.DiscountID = discountRef
	o.CouponID = couponRef
	return s.orders.UpdatePromo(ctx, tenantID, o.ID, discountRef, couponRef, o.DiscountTotal, o.Total)
}

// ---- split bills ----

// CreateSplits divides an unpaid order into per-person amounts.
func (s *OrderService) CreateSplits(ctx context.Context, tenantID, userID, orderID string, amounts []int64) (*order.Order, error) {
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status != order.StatusOpen && o.Status != order.StatusHeld {
		return nil, apperror.Validation("only unpaid orders can be split")
	}
	if len(o.Splits) > 0 {
		return nil, apperror.Conflict("this order already has splits")
	}
	if len(amounts) < 2 {
		return nil, apperror.Validation("a split needs at least 2 parts")
	}
	var sum int64
	for _, a := range amounts {
		if a <= 0 {
			return nil, apperror.Validation("split amounts must be positive")
		}
		sum += a
	}
	if sum != o.Total {
		return nil, apperror.Validation("split amounts must add up to the order total")
	}

	if _, err := s.orders.CreateSplits(ctx, tenantID, orderID, amounts); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "order.split",
		EntityType: "order", EntityID: orderID, After: map[string]any{"parts": len(amounts)},
	})
	return s.orders.GetByID(ctx, tenantID, orderID)
}

// PaySplit settles one split. The order completes when every split is
// paid. Overpayment on a split becomes cash change.
func (s *OrderService) PaySplit(ctx context.Context, tenantID, userID, orderID, splitID string, payments []PaymentInput) (*order.Order, error) {
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	split, err := s.orders.GetSplit(ctx, tenantID, splitID)
	if err != nil {
		return nil, err
	}
	if split.OrderID != orderID {
		return nil, apperror.Validation("split does not belong to this order")
	}
	if split.Status != "pending" {
		return nil, apperror.Validation("this split is already paid")
	}

	cashPaid, nonCashPaid, err := sumPayments(payments)
	if err != nil {
		return nil, err
	}
	if nonCashPaid > split.Amount {
		return nil, apperror.Validation("non-cash payments exceed this split's amount")
	}
	tendered := cashPaid + nonCashPaid
	if tendered < split.Amount {
		return nil, apperror.Validation("payments do not cover this split")
	}
	change := tendered - split.Amount

	var drawerSession *order.DrawerSession
	if cashPaid > 0 {
		drawerSession, err = s.drawer.Current(ctx, tenantID)
		if err != nil {
			return nil, apperror.Validation("open the cash drawer before accepting cash")
		}
	}

	if _, err := s.redeemPointsPayments(ctx, tenantID, userID, o, payments); err != nil {
		return nil, err
	}

	for _, p := range payments {
		payment := &order.Payment{
			OrderID: orderID, SplitID: &split.ID, Method: p.Method, Amount: p.Amount,
			ReferenceNo: p.ReferenceNo, ReceivedBy: userID,
		}
		if err := s.orders.AddPayment(ctx, tenantID, payment); err != nil {
			return nil, apperror.Internal(err)
		}
	}
	if err := s.recordCashSale(ctx, tenantID, userID, orderID, drawerSession, cashPaid, change); err != nil {
		return nil, err
	}
	if err := s.orders.MarkSplitPaid(ctx, tenantID, splitID); err != nil {
		return nil, err
	}

	// Complete the order once every split is settled.
	splits, err := s.orders.ListSplits(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	allPaid := true
	for _, sp := range splits {
		if sp.Status != "paid" {
			allPaid = false
			break
		}
	}
	if allPaid {
		now := time.Now()
		if err := s.orders.UpdateStatus(ctx, tenantID, orderID, order.StatusCompleted, &now); err != nil {
			return nil, apperror.Internal(err)
		}
		if err := s.orders.AddStatusHistory(ctx, tenantID, orderID, "status", o.Status, order.StatusCompleted, userID); err != nil {
			return nil, apperror.Internal(err)
		}
		s.auditor.Record(audit.Log{
			TenantID: tenantID, UserID: userID, Action: "order.completed",
			EntityType: "order", EntityID: orderID, After: map[string]any{"via": "split_bill"},
		})
		s.deductInventory(ctx, tenantID, userID, o)
		s.awardLoyalty(ctx, tenantID, userID, orderID)
		s.bustSalesCache(ctx, tenantID)
	}
	return s.orders.GetByID(ctx, tenantID, orderID)
}

// ---- refunds ----

type RefundItemInput struct {
	OrderItemID string
	Qty         int
}

// Refund issues a full or partial refund. With items, the amount is the
// sum of unit prices × qty; with a custom amount, that amount is used;
// with neither, the remaining refundable balance is refunded.
func (s *OrderService) Refund(ctx context.Context, tenantID, userID, orderID, reason string, items []RefundItemInput, customAmount int64) (*order.Order, error) {
	if reason == "" {
		return nil, apperror.Validation("a refund reason is required")
	}
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status != order.StatusCompleted && o.Status != order.StatusPartiallyRefunded {
		return nil, apperror.Validation("only completed orders can be refunded")
	}

	refunded, err := s.orders.RefundedTotal(ctx, tenantID, orderID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	refundable := o.Total - refunded
	if refundable <= 0 {
		return nil, apperror.Validation("this order is fully refunded")
	}

	var amount int64
	var refundItems []order.RefundItem
	switch {
	case len(items) > 0:
		itemByID := map[string]order.Item{}
		for _, it := range o.Items {
			itemByID[it.ID] = it
		}
		for _, in := range items {
			it, ok := itemByID[in.OrderItemID]
			if !ok {
				return nil, apperror.Validation("refund item not found on this order")
			}
			if in.Qty <= 0 || in.Qty > it.Qty {
				return nil, apperror.Validation("invalid refund quantity for " + it.Name)
			}
			lineAmount := it.UnitPrice * int64(in.Qty)
			amount += lineAmount
			refundItems = append(refundItems, order.RefundItem{
				OrderItemID: in.OrderItemID, Qty: in.Qty, Amount: lineAmount,
			})
		}
	case customAmount > 0:
		amount = customAmount
	default:
		amount = refundable
	}
	if amount > refundable {
		return nil, apperror.Validation("refund exceeds the remaining refundable amount")
	}

	number, err := s.orders.NextRefundNumber(ctx, tenantID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	refund := &order.Refund{
		OrderID: orderID, RefundNumber: number, Reason: reason,
		Amount: amount, RefundedBy: userID, Items: refundItems,
	}
	if err := s.orders.CreateRefund(ctx, tenantID, refund); err != nil {
		return nil, apperror.Internal(err)
	}

	// Cash leaves the drawer when one is open.
	if session, err := s.drawer.Current(ctx, tenantID); err == nil {
		if err := s.drawer.AddMovement(ctx, tenantID, &order.CashMovement{
			SessionID: session.ID, Type: "refund", Amount: -amount,
			OrderID: &orderID, Reason: reason, CreatedBy: userID,
		}); err != nil {
			return nil, apperror.Internal(err)
		}
	}

	newStatus := order.StatusPartiallyRefunded
	if refunded+amount >= o.Total {
		newStatus = order.StatusRefunded
	}
	if err := s.orders.UpdateStatus(ctx, tenantID, orderID, newStatus, nil); err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.orders.AddStatusHistory(ctx, tenantID, orderID, "status", o.Status, newStatus, userID); err != nil {
		return nil, apperror.Internal(err)
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "order.refunded",
		EntityType: "order", EntityID: orderID,
		After: map[string]any{"amount": amount, "reason": reason, "status": newStatus},
	})
	s.bustSalesCache(ctx, tenantID)
	return s.orders.GetByID(ctx, tenantID, orderID)
}

// ---- voids ----

// Void cancels an order. Cash already taken goes back out of the
// drawer; a redeemed coupon is released.
func (s *OrderService) Void(ctx context.Context, tenantID, userID, orderID, reason string) (*order.Order, error) {
	if reason == "" {
		return nil, apperror.Validation("a void reason is required")
	}
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	switch o.Status {
	case order.StatusVoided:
		return nil, apperror.Validation("this order is already voided")
	case order.StatusRefunded, order.StatusPartiallyRefunded:
		return nil, apperror.Validation("refunded orders cannot be voided")
	}

	var cashPaid int64
	for _, p := range o.Payments {
		if p.Method == order.MethodCash && p.Status == "paid" {
			cashPaid += p.Amount
		}
	}
	cashOut := cashPaid - o.Change // net cash the drawer actually kept
	if cashOut > 0 {
		if session, err := s.drawer.Current(ctx, tenantID); err == nil {
			if err := s.drawer.AddMovement(ctx, tenantID, &order.CashMovement{
				SessionID: session.ID, Type: "refund", Amount: -cashOut,
				OrderID: &orderID, Reason: "void: " + reason, CreatedBy: userID,
			}); err != nil {
				return nil, apperror.Internal(err)
			}
		}
	}

	if o.CouponID != nil {
		if err := s.coupons.Release(ctx, tenantID, *o.CouponID, orderID); err != nil {
			s.logger.Warn("failed to release coupon on void", "order_id", orderID, "error", err)
		}
	}

	// Undo loyalty activity: earned points come back out, redeemed go back.
	if s.loyalty != nil && o.CustomerID != nil {
		s.loyalty.ReverseForOrder(ctx, tenantID, userID, orderID)
	}

	if err := s.orders.SetVoided(ctx, tenantID, orderID, userID, reason); err != nil {
		return nil, err
	}
	if err := s.orders.AddStatusHistory(ctx, tenantID, orderID, "status", o.Status, order.StatusVoided, userID); err != nil {
		return nil, apperror.Internal(err)
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "order.voided",
		EntityType: "order", EntityID: orderID,
		Before: map[string]any{"status": o.Status},
		After:  map[string]any{"reason": reason, "cash_returned": cashOut},
	})
	s.bustSalesCache(ctx, tenantID)
	return s.orders.GetByID(ctx, tenantID, orderID)
}

// ---- shared helpers ----

func sumPayments(payments []PaymentInput) (cash, nonCash int64, err error) {
	if len(payments) == 0 {
		return 0, 0, apperror.Validation("at least one payment is required")
	}
	for _, p := range payments {
		if !order.ValidMethod(p.Method) {
			return 0, 0, apperror.Validation("unsupported payment method: " + p.Method)
		}
		if p.Amount <= 0 {
			return 0, 0, apperror.Validation("payment amounts must be positive")
		}
		if p.Method == order.MethodCash {
			cash += p.Amount
		} else {
			nonCash += p.Amount
		}
	}
	return cash, nonCash, nil
}

// recordCashSale writes the drawer movements for a cash-bearing payment.
func (s *OrderService) recordCashSale(ctx context.Context, tenantID, userID, orderID string, session *order.DrawerSession, cashPaid, change int64) error {
	if session == nil || cashPaid == 0 {
		return nil
	}
	if err := s.drawer.AddMovement(ctx, tenantID, &order.CashMovement{
		SessionID: session.ID, Type: "sale", Amount: cashPaid,
		OrderID: &orderID, CreatedBy: userID,
	}); err != nil {
		return apperror.Internal(err)
	}
	if change > 0 {
		if err := s.drawer.AddMovement(ctx, tenantID, &order.CashMovement{
			SessionID: session.ID, Type: "change", Amount: -change,
			OrderID: &orderID, CreatedBy: userID,
		}); err != nil {
			return apperror.Internal(err)
		}
	}
	return nil
}
