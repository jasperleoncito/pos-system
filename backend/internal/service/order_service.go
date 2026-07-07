package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/catalog"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/order"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/promo"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/realtime"
)

// OrderService owns the sale lifecycle: create/hold/resume, payments,
// completion, receipts, and the cash drawer.
type OrderService struct {
	orders    order.Repository
	drawer    order.DrawerRepository
	products  catalog.ProductRepository
	taxes     catalog.TaxRepository
	settings  tenant.SettingsRepository
	tenants   tenant.Repository
	discounts promo.DiscountRepository
	coupons   promo.CouponRepository
	hub       *realtime.Hub
	inventory *InventoryService
	loyalty   *LoyaltyService
	auditor   *AuditService
	logger    *slog.Logger
}

// SetInventory attaches the inventory service after construction
// (both live in this package; avoids a constructor cycle).
func (s *OrderService) SetInventory(inv *InventoryService) { s.inventory = inv }

// SetLoyalty attaches the loyalty service after construction.
func (s *OrderService) SetLoyalty(l *LoyaltyService) { s.loyalty = l }

// awardLoyalty credits earned points once an order completes. The earn
// base excludes value paid with points themselves.
func (s *OrderService) awardLoyalty(ctx context.Context, tenantID, userID, orderID string) {
	if s.loyalty == nil {
		return
	}
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil || o.CustomerID == nil {
		return
	}
	var pointsValue int64
	for _, p := range o.Payments {
		if p.Method == order.MethodPoints {
			pointsValue += p.Amount
		}
	}
	s.loyalty.AwardForOrder(ctx, tenantID, userID, *o.CustomerID, orderID, o.Total-pointsValue)
}

// redeemPointsPayments deducts loyalty points for any "points" tender
// lines before payments are recorded. Returns the total points value.
func (s *OrderService) redeemPointsPayments(ctx context.Context, tenantID, userID string, o *order.Order, payments []PaymentInput) (int64, error) {
	var pointsValue int64
	for _, p := range payments {
		if p.Method == order.MethodPoints {
			pointsValue += p.Amount
		}
	}
	if pointsValue == 0 {
		return 0, nil
	}
	if s.loyalty == nil {
		return 0, apperror.Validation("loyalty payments are not available")
	}
	if o.CustomerID == nil {
		return 0, apperror.Validation("attach a customer before redeeming points")
	}
	if _, err := s.loyalty.RedeemForPayment(ctx, tenantID, userID, *o.CustomerID, o.ID, pointsValue); err != nil {
		return 0, err
	}
	return pointsValue, nil
}

// deductInventory runs recipe deduction after an order completes.
func (s *OrderService) deductInventory(ctx context.Context, tenantID, userID string, o *order.Order) {
	if s.inventory != nil {
		s.inventory.DeductForOrder(ctx, tenantID, userID, o)
	}
}

type OrderServiceDeps struct {
	Orders    order.Repository
	Drawer    order.DrawerRepository
	Products  catalog.ProductRepository
	Taxes     catalog.TaxRepository
	Settings  tenant.SettingsRepository
	Tenants   tenant.Repository
	Discounts promo.DiscountRepository
	Coupons   promo.CouponRepository
	Hub       *realtime.Hub
	Auditor   *AuditService
	Logger    *slog.Logger
}

func NewOrderService(d OrderServiceDeps) *OrderService {
	return &OrderService{
		orders: d.Orders, drawer: d.Drawer, products: d.Products, taxes: d.Taxes,
		settings: d.Settings, tenants: d.Tenants, discounts: d.Discounts, coupons: d.Coupons,
		hub: d.Hub, auditor: d.Auditor, logger: d.Logger,
	}
}

// CreateOrderItemInput is one cart line from the client. Prices are
// looked up server-side; the client never supplies amounts.
type CreateOrderItemInput struct {
	ProductID   string
	VariantID   string
	Qty         int
	ModifierIDs []string
	Notes       string
}

type CreateOrderInput struct {
	OrderType   string
	TableNumber string
	Notes       string
	Hold        bool // park the order instead of opening it for payment
	DiscountID  string
	CouponCode  string
	CustomerID  string // optional loyalty customer
	Items       []CreateOrderItemInput
}

// CreateOrder prices the cart from the catalog and persists the order.
func (s *OrderService) CreateOrder(ctx context.Context, tenantID, userID string, in CreateOrderInput) (*order.Order, error) {
	if !order.ValidOrderType(in.OrderType) {
		return nil, apperror.Validation("order type must be dine_in, takeout, or delivery")
	}
	if len(in.Items) == 0 {
		return nil, apperror.Validation("order must contain at least one item")
	}
	if in.OrderType == order.TypeDineIn && in.TableNumber == "" {
		return nil, apperror.Validation("table number is required for dine-in orders")
	}

	var (
		items    []order.Item
		subtotal int64
		taxTotal int64
	)
	for _, line := range in.Items {
		if line.Qty <= 0 {
			return nil, apperror.Validation("item quantity must be at least 1")
		}
		product, err := s.products.GetByID(ctx, tenantID, line.ProductID)
		if err != nil {
			return nil, apperror.Validation("a product in the cart no longer exists")
		}
		if !product.IsActive {
			return nil, apperror.Validation(fmt.Sprintf("%s is not available", product.Name))
		}

		item, err := s.priceLine(ctx, tenantID, product, line)
		if err != nil {
			return nil, err
		}
		subtotal += item.LineTotal
		taxTotal += item.TaxAmount
		items = append(items, *item)
	}

	number, err := s.orders.NextOrderNumber(ctx, tenantID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	var customerID *string
	if in.CustomerID != "" {
		if s.loyalty == nil {
			return nil, apperror.Validation("customers are not available")
		}
		c, err := s.loyalty.GetCustomer(ctx, tenantID, in.CustomerID)
		if err != nil {
			return nil, apperror.Validation("customer not found")
		}
		customerID = &c.ID
	}

	status := order.StatusOpen
	if in.Hold {
		status = order.StatusHeld
	}
	o := &order.Order{
		OrderNumber:   number,
		OrderType:     in.OrderType,
		TableNumber:   in.TableNumber,
		CustomerID:    customerID,
		CashierUserID: userID,
		Status:        status,
		Subtotal:      subtotal,
		TaxTotal:      taxTotal,
		Total:         subtotal,
		Notes:         in.Notes,
		Items:         items,
	}
	if err := s.orders.Create(ctx, tenantID, o); err != nil {
		return nil, apperror.Internal(err)
	}

	if in.DiscountID != "" || in.CouponCode != "" {
		if err := s.applyPromo(ctx, tenantID, o, in.DiscountID, in.CouponCode); err != nil {
			return nil, err
		}
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "order.created",
		EntityType: "order", EntityID: o.ID,
		After: map[string]any{"order_number": o.OrderNumber, "total": o.Total, "status": o.Status},
	})
	if !in.Hold {
		s.fireToKitchen(ctx, tenantID, o)
	}
	return s.orders.GetByID(ctx, tenantID, o.ID)
}

// priceLine snapshots names and computes the line total and tax.
func (s *OrderService) priceLine(ctx context.Context, tenantID string, product *catalog.Product, line CreateOrderItemInput) (*order.Item, error) {
	unitPrice := product.BasePrice

	var variantID *string
	variantName := ""
	if line.VariantID != "" {
		idx := slices.IndexFunc(product.Variants, func(v catalog.Variant) bool { return v.ID == line.VariantID })
		if idx < 0 {
			return nil, apperror.Validation(fmt.Sprintf("variant not found for %s", product.Name))
		}
		v := product.Variants[idx]
		unitPrice += v.PriceDelta
		variantID = &v.ID
		variantName = v.Name
	}

	// Validate modifiers against the product's assigned groups and
	// enforce required groups.
	modifierByID := map[string]struct {
		mod   catalog.Modifier
		group catalog.ModifierGroup
	}{}
	for _, g := range product.ModifierGroups {
		for _, m := range g.Modifiers {
			modifierByID[m.ID] = struct {
				mod   catalog.Modifier
				group catalog.ModifierGroup
			}{m, g}
		}
	}

	var mods []order.ItemModifier
	selectedPerGroup := map[string]int{}
	for _, modID := range line.ModifierIDs {
		entry, ok := modifierByID[modID]
		if !ok {
			return nil, apperror.Validation(fmt.Sprintf("invalid modifier for %s", product.Name))
		}
		selectedPerGroup[entry.group.ID]++
		if selectedPerGroup[entry.group.ID] > entry.group.MaxSelect {
			return nil, apperror.Validation(fmt.Sprintf("too many %s selections for %s", entry.group.Name, product.Name))
		}
		unitPrice += entry.mod.PriceDelta
		mods = append(mods, order.ItemModifier{
			ModifierID: entry.mod.ID,
			GroupName:  entry.group.Name,
			Name:       entry.mod.Name,
			PriceDelta: entry.mod.PriceDelta,
		})
	}
	for _, g := range product.ModifierGroups {
		if g.IsRequired && selectedPerGroup[g.ID] < g.MinSelect {
			return nil, apperror.Validation(fmt.Sprintf("%s requires a %s selection", product.Name, g.Name))
		}
	}

	lineTotal := unitPrice * int64(line.Qty)

	var taxAmount int64
	if product.TaxID != nil {
		tax, err := s.taxes.GetByID(ctx, tenantID, *product.TaxID)
		if err == nil && tax.IsActive {
			if tax.IsInclusive {
				taxAmount = InclusiveTaxPortion(lineTotal, tax.RatePercent)
			} else {
				taxAmount = ExclusiveTaxAmount(lineTotal, tax.RatePercent)
				lineTotal += taxAmount
			}
		}
	}

	return &order.Item{
		ProductID:   product.ID,
		VariantID:   variantID,
		Name:        product.Name,
		VariantName: variantName,
		UnitPrice:   unitPrice,
		Qty:         line.Qty,
		TaxAmount:   taxAmount,
		LineTotal:   lineTotal,
		Notes:       line.Notes,
		Status:      "pending",
		Modifiers:   mods,
	}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, tenantID, id string) (*order.Order, error) {
	return s.orders.GetByID(ctx, tenantID, id)
}

func (s *OrderService) ListOrders(ctx context.Context, tenantID string, f order.Filter) ([]order.Order, int64, error) {
	return s.orders.List(ctx, tenantID, f)
}

// SetHold parks or resumes an unpaid order.
func (s *OrderService) SetHold(ctx context.Context, tenantID, userID, orderID string, hold bool) (*order.Order, error) {
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	target := order.StatusOpen
	if hold {
		target = order.StatusHeld
	}
	if o.Status != order.StatusOpen && o.Status != order.StatusHeld {
		return nil, apperror.Validation("only unpaid orders can be held or resumed")
	}
	if o.Status == target {
		return o, nil
	}
	if err := s.orders.UpdateStatus(ctx, tenantID, orderID, target, nil); err != nil {
		return nil, err
	}
	if err := s.orders.AddStatusHistory(ctx, tenantID, orderID, "status", o.Status, target, userID); err != nil {
		return nil, err
	}
	if target == order.StatusOpen {
		s.fireToKitchen(ctx, tenantID, o) // resumed orders hit the kitchen now
	}
	return s.orders.GetByID(ctx, tenantID, orderID)
}

// PaymentInput is one tender line in a (possibly mixed) payment.
type PaymentInput struct {
	Method      string
	Amount      int64
	ReferenceNo string
}

// Pay settles an order with one or more payments. Cash overpayment
// becomes change; non-cash methods must not exceed what is due.
func (s *OrderService) Pay(ctx context.Context, tenantID, userID, orderID string, payments []PaymentInput) (*order.Order, error) {
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status != order.StatusOpen && o.Status != order.StatusHeld {
		return nil, apperror.Validation("this order has already been settled")
	}
	if len(o.Splits) > 0 {
		return nil, apperror.Validation("this order is split — settle each split instead")
	}

	cashPaid, nonCashPaid, err := sumPayments(payments)
	if err != nil {
		return nil, err
	}

	total := o.Total
	if nonCashPaid > total {
		return nil, apperror.Validation("non-cash payments exceed the amount due")
	}
	tendered := cashPaid + nonCashPaid
	if tendered < total {
		return nil, apperror.Validation("payments do not cover the total")
	}
	change := tendered - total // only possible from cash by the checks above

	// Cash requires an open drawer to keep expected cash honest.
	var drawerSession *order.DrawerSession
	if cashPaid > 0 {
		drawerSession, err = s.drawer.Current(ctx, tenantID)
		if err != nil {
			return nil, apperror.Validation("open the cash drawer before accepting cash")
		}
	}

	// Points redemption deducts the balance before payments are booked.
	if _, err := s.redeemPointsPayments(ctx, tenantID, userID, o, payments); err != nil {
		return nil, err
	}

	for _, p := range payments {
		payment := &order.Payment{
			OrderID: orderID, Method: p.Method, Amount: p.Amount,
			ReferenceNo: p.ReferenceNo, ReceivedBy: userID,
		}
		if err := s.orders.AddPayment(ctx, tenantID, payment); err != nil {
			return nil, apperror.Internal(err)
		}
	}

	if err := s.recordCashSale(ctx, tenantID, userID, orderID, drawerSession, cashPaid, change); err != nil {
		return nil, err
	}

	if err := s.orders.UpdatePaymentTotals(ctx, tenantID, orderID, tendered, change); err != nil {
		return nil, apperror.Internal(err)
	}
	now := time.Now()
	if err := s.orders.UpdateStatus(ctx, tenantID, orderID, order.StatusCompleted, &now); err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.orders.AddStatusHistory(ctx, tenantID, orderID, "status", o.Status, order.StatusCompleted, userID); err != nil {
		return nil, apperror.Internal(err)
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "order.completed",
		EntityType: "order", EntityID: orderID,
		After: map[string]any{"total": total, "tendered": tendered, "change": change},
	})
	if o.Status == order.StatusHeld {
		s.fireToKitchen(ctx, tenantID, o) // settled straight from hold
	}
	s.deductInventory(ctx, tenantID, userID, o)
	s.awardLoyalty(ctx, tenantID, userID, orderID)
	return s.orders.GetByID(ctx, tenantID, orderID)
}

// Receipt bundles everything a printable receipt needs.
type Receipt struct {
	Order    *order.Order    `json:"order"`
	Business ReceiptBusiness `json:"business"`
}

type ReceiptBusiness struct {
	Name          string `json:"name"`
	LogoURL       string `json:"logo_url"`
	ReceiptHeader string `json:"receipt_header"`
	ReceiptFooter string `json:"receipt_footer"`
	Address       string `json:"address"`
	ContactNumber string `json:"contact_number"`
	TaxLabel      string `json:"tax_label"`
	TaxID         string `json:"tax_id"`
}

func (s *OrderService) GetReceipt(ctx context.Context, tenantID, orderID string, logoURL func(string) string) (*Receipt, error) {
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	settings, err := s.settings.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &Receipt{
		Order: o,
		Business: ReceiptBusiness{
			Name:          t.Name,
			LogoURL:       logoURL(settings.LogoThumbKey),
			ReceiptHeader: settings.ReceiptHeader,
			ReceiptFooter: settings.ReceiptFooter,
			Address:       settings.Address,
			ContactNumber: settings.ContactNumber,
			TaxLabel:      settings.TaxLabel,
			TaxID:         settings.TaxID,
		},
	}, nil
}

// ---- cash drawer ----

func (s *OrderService) OpenDrawer(ctx context.Context, tenantID, userID string, openingFloat int64) (*order.DrawerSession, error) {
	if openingFloat < 0 {
		return nil, apperror.Validation("opening float cannot be negative")
	}
	session := &order.DrawerSession{OpenedBy: userID, OpeningFloat: openingFloat}
	if err := s.drawer.Open(ctx, tenantID, session); err != nil {
		return nil, err
	}
	if err := s.drawer.AddMovement(ctx, tenantID, &order.CashMovement{
		SessionID: session.ID, Type: "open_float", Amount: 0, CreatedBy: userID,
		Reason: "drawer opened",
	}); err != nil {
		return nil, apperror.Internal(err)
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "drawer.opened",
		EntityType: "cash_drawer", EntityID: session.ID,
		After: map[string]any{"opening_float": openingFloat},
	})
	return s.drawer.Current(ctx, tenantID)
}

func (s *OrderService) CurrentDrawer(ctx context.Context, tenantID string) (*order.DrawerSession, []order.CashMovement, error) {
	session, err := s.drawer.Current(ctx, tenantID)
	if err != nil {
		return nil, nil, err
	}
	movements, err := s.drawer.ListMovements(ctx, tenantID, session.ID)
	if err != nil {
		return nil, nil, err
	}
	return session, movements, nil
}

func (s *OrderService) CloseDrawer(ctx context.Context, tenantID, userID string, countedCash int64) (*order.DrawerSession, error) {
	if countedCash < 0 {
		return nil, apperror.Validation("counted cash cannot be negative")
	}
	session, err := s.drawer.Current(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	closed, err := s.drawer.Close(ctx, tenantID, session.ID, userID, countedCash)
	if err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "drawer.closed",
		EntityType: "cash_drawer", EntityID: closed.ID,
		After: map[string]any{
			"expected_cash": closed.ExpectedCash,
			"counted_cash":  closed.CountedCash,
			"variance":      closed.Variance,
		},
	})
	return closed, nil
}
