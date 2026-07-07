package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/inventory"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/order"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// InventoryService owns stock items, recipes, and the movement ledger.
type InventoryService struct {
	repo    inventory.Repository
	alerts  AlertSink
	auditor *AuditService
	logger  *slog.Logger
}

func NewInventoryService(repo inventory.Repository, auditor *AuditService, logger *slog.Logger) *InventoryService {
	return &InventoryService{repo: repo, auditor: auditor, logger: logger}
}

func (s *InventoryService) ListUnits(ctx context.Context, tenantID string) ([]inventory.Unit, error) {
	return s.repo.ListUnits(ctx, tenantID)
}

func (s *InventoryService) CreateUnit(ctx context.Context, tenantID string, u *inventory.Unit) error {
	return s.repo.CreateUnit(ctx, tenantID, u)
}

func (s *InventoryService) ListItems(ctx context.Context, tenantID, search string) ([]inventory.Item, error) {
	return s.repo.ListItems(ctx, tenantID, search)
}

func (s *InventoryService) CreateItem(ctx context.Context, tenantID, userID string, i *inventory.Item) error {
	if i.Type != inventory.TypeIngredient && i.Type != inventory.TypeFinishedGood {
		return apperror.Validation("type must be ingredient or finished_good")
	}
	if err := s.repo.CreateItem(ctx, tenantID, i); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "inventory.item_created",
		EntityType: "inventory_item", EntityID: i.ID, After: map[string]any{"name": i.Name},
	})
	return nil
}

func (s *InventoryService) UpdateItem(ctx context.Context, tenantID, userID string, i *inventory.Item) error {
	if err := s.repo.UpdateItem(ctx, tenantID, i); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "inventory.item_updated",
		EntityType: "inventory_item", EntityID: i.ID, After: map[string]any{"name": i.Name},
	})
	return nil
}

func (s *InventoryService) DeleteItem(ctx context.Context, tenantID, userID, id string) error {
	if err := s.repo.SoftDeleteItem(ctx, tenantID, id); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "inventory.item_deleted",
		EntityType: "inventory_item", EntityID: id,
	})
	return nil
}

// Move applies a manual stock movement (stock in/out, adjustment, waste).
func (s *InventoryService) Move(ctx context.Context, tenantID, userID string, in inventory.ApplyInput) (*inventory.Movement, error) {
	switch in.MovementType {
	case inventory.MoveStockIn:
		if in.QtyDelta <= 0 {
			return nil, apperror.Validation("stock in quantity must be positive")
		}
	case inventory.MoveStockOut, inventory.MoveWaste:
		if in.QtyDelta >= 0 {
			in.QtyDelta = -in.QtyDelta
		}
	case inventory.MoveAdjustment:
		if in.QtyDelta == 0 {
			return nil, apperror.Validation("adjustment cannot be zero")
		}
		if in.Notes == "" {
			return nil, apperror.Validation("adjustments require a reason in notes")
		}
	default:
		return nil, apperror.Validation("invalid movement type")
	}
	in.ReferenceType = "manual"
	in.PerformedBy = userID

	movement, err := s.repo.Apply(ctx, tenantID, in)
	if err != nil {
		return nil, err
	}
	s.checkAlert(ctx, tenantID, in.ItemID)
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "inventory." + in.MovementType,
		EntityType: "inventory_item", EntityID: in.ItemID,
		After: map[string]any{"delta": in.QtyDelta, "after": movement.QtyAfter, "notes": in.Notes},
	})
	return movement, nil
}

func (s *InventoryService) ListMovements(ctx context.Context, tenantID, itemID string, limit, offset int) ([]inventory.Movement, int64, error) {
	return s.repo.ListMovements(ctx, tenantID, itemID, limit, offset)
}

func (s *InventoryService) GetRecipe(ctx context.Context, tenantID, productID string) ([]inventory.RecipeItem, error) {
	return s.repo.GetRecipe(ctx, tenantID, productID)
}

func (s *InventoryService) SaveRecipe(ctx context.Context, tenantID, userID, productID string, items []inventory.RecipeItem) ([]inventory.RecipeItem, error) {
	for _, ri := range items {
		if ri.Qty <= 0 {
			return nil, apperror.Validation("recipe quantities must be positive")
		}
		if _, err := s.repo.GetItem(ctx, tenantID, ri.InventoryItemID); err != nil {
			return nil, apperror.Validation("recipe references an unknown inventory item")
		}
	}
	if err := s.repo.ReplaceRecipe(ctx, tenantID, productID, items); err != nil {
		return nil, err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: userID, Action: "inventory.recipe_updated",
		EntityType: "product", EntityID: productID, After: map[string]any{"ingredients": len(items)},
	})
	return s.repo.GetRecipe(ctx, tenantID, productID)
}

// DeductForOrder consumes recipe ingredients for every line of a
// completed order. Idempotent: an order that already produced sale
// movements is skipped, so double-settlement never double-deducts.
func (s *InventoryService) DeductForOrder(ctx context.Context, tenantID, userID string, o *order.Order) {
	done, err := s.repo.HasMovements(ctx, tenantID, "order", o.ID, inventory.MoveSale)
	if err != nil {
		s.logger.Error("inventory deduction check failed", "order_id", o.ID, "error", err)
		return
	}
	if done {
		return
	}

	for _, item := range o.Items {
		recipe, err := s.repo.GetRecipe(ctx, tenantID, item.ProductID)
		if err != nil {
			s.logger.Error("failed to load recipe", "product_id", item.ProductID, "error", err)
			continue
		}
		for _, ri := range recipe {
			if _, err := s.repo.Apply(ctx, tenantID, inventory.ApplyInput{
				ItemID:        ri.InventoryItemID,
				MovementType:  inventory.MoveSale,
				QtyDelta:      -ri.Qty * float64(item.Qty),
				ReferenceType: "order",
				ReferenceID:   o.ID,
				Notes:         fmt.Sprintf("order #%d — %dx %s", o.OrderNumber, item.Qty, item.Name),
				PerformedBy:   userID,
			}); err != nil {
				s.logger.Error("inventory deduction failed",
					"order_id", o.ID, "item_id", ri.InventoryItemID, "error", err)
				continue
			}
			s.checkAlert(ctx, tenantID, ri.InventoryItemID)
		}
	}
}
