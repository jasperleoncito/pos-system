package service

import (
	"context"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/order"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/realtime"
)

// Kitchen display operations — extensions of OrderService.

// ListKitchenOrders returns the active kitchen queue.
func (s *OrderService) ListKitchenOrders(ctx context.Context, tenantID string) ([]order.Order, error) {
	return s.orders.ListKitchen(ctx, tenantID)
}

// SetKitchenStatus moves a ticket through pending→preparing→ready→completed.
func (s *OrderService) SetKitchenStatus(ctx context.Context, tenantID, userID, orderID, status string) (*order.Order, error) {
	if !order.ValidKitchenStatus(status) {
		return nil, apperror.Validation("invalid kitchen status")
	}
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status == order.StatusVoided || o.Status == order.StatusRefunded {
		return nil, apperror.Validation("this order is no longer active")
	}
	if o.KitchenStatus == status {
		return o, nil
	}

	if err := s.orders.UpdateKitchenStatus(ctx, tenantID, orderID, status); err != nil {
		return nil, err
	}
	if err := s.orders.AddStatusHistory(ctx, tenantID, orderID, "kitchen_status", o.KitchenStatus, status, userID); err != nil {
		return nil, apperror.Internal(err)
	}

	s.publishKitchen(ctx, tenantID, realtime.Event{
		Type: "kitchen_status", OrderID: orderID, OrderNumber: o.OrderNumber, Value: status,
	})
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "kitchen.status_changed",
		EntityType: "order", EntityID: orderID,
		Before: map[string]any{"kitchen_status": o.KitchenStatus},
		After:  map[string]any{"kitchen_status": status},
	})
	return s.orders.GetByID(ctx, tenantID, orderID)
}

// SetItemStatus marks a single line as pending/ready etc. on the board.
func (s *OrderService) SetItemStatus(ctx context.Context, tenantID, userID, orderID, itemID, status string) (*order.Order, error) {
	if !order.ValidKitchenStatus(status) {
		return nil, apperror.Validation("invalid item status")
	}
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if err := s.orders.UpdateItemStatus(ctx, tenantID, orderID, itemID, status); err != nil {
		return nil, err
	}
	s.publishKitchen(ctx, tenantID, realtime.Event{
		Type: "item_status", OrderID: orderID, OrderNumber: o.OrderNumber, Value: status,
	})
	return s.orders.GetByID(ctx, tenantID, orderID)
}

// SetPriority flags a rush order to the top of the board.
func (s *OrderService) SetPriority(ctx context.Context, tenantID, userID, orderID string, priority bool) (*order.Order, error) {
	o, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if err := s.orders.SetPriority(ctx, tenantID, orderID, priority); err != nil {
		return nil, err
	}
	s.publishKitchen(ctx, tenantID, realtime.Event{
		Type: "priority", OrderID: orderID, OrderNumber: o.OrderNumber,
		Value: map[bool]string{true: "on", false: "off"}[priority],
	})
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "kitchen.priority_changed",
		EntityType: "order", EntityID: orderID, After: map[string]any{"priority": priority},
	})
	return s.orders.GetByID(ctx, tenantID, orderID)
}

// fireToKitchen announces a new ticket. Called when an order is created
// unheld, resumed from hold, or settled directly from hold.
func (s *OrderService) fireToKitchen(ctx context.Context, tenantID string, o *order.Order) {
	s.publishKitchen(ctx, tenantID, realtime.Event{
		Type: "order_fired", OrderID: o.ID, OrderNumber: o.OrderNumber, Value: o.KitchenStatus,
	})
}

func (s *OrderService) publishKitchen(ctx context.Context, tenantID string, event realtime.Event) {
	if s.hub != nil {
		s.hub.Publish(ctx, tenantID, event)
	}
}
