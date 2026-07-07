package main

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/inventory"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
)

// seedInventory adds sample units, ingredients, and recipes so recipe
// deduction is demonstrable out of the box. Idempotent.
func seedInventory(ctx context.Context, db *pgxpool.Pool, tenantID string, logger *slog.Logger) error {
	repo := postgres.NewInventoryRepo(db)

	existing, err := repo.ListItems(ctx, tenantID, "")
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		logger.Info("inventory already seeded — skipping", "items", len(existing))
		return nil
	}

	units := map[string]*inventory.Unit{
		"kg":  {Name: "Kilogram", Abbreviation: "kg"},
		"pcs": {Name: "Piece", Abbreviation: "pcs"},
		"L":   {Name: "Liter", Abbreviation: "L"},
	}
	for _, u := range units {
		if err := repo.CreateUnit(ctx, tenantID, u); err != nil {
			return err
		}
	}

	items := []struct {
		name    string
		unit    string
		stock   float64
		reorder float64
		cost    int64 // centavos per unit
	}{
		{"Rice", "kg", 50, 10, 6000},
		{"Pork Belly", "kg", 20, 5, 32000},
		{"Chicken", "kg", 25, 5, 21000},
		{"Beef", "kg", 15, 4, 42000},
		{"Egg", "pcs", 120, 30, 900},
		{"Cooking Oil", "L", 18, 5, 9500},
		{"C2 Solo Bottle", "pcs", 48, 12, 1600},
	}
	itemIDs := map[string]string{}
	for _, def := range items {
		item := &inventory.Item{
			Name: def.name, Type: inventory.TypeIngredient, UnitID: units[def.unit].ID,
			CurrentStock: def.stock, ReorderLevel: def.reorder, CostPerUnit: def.cost, IsActive: true,
		}
		if def.name == "C2 Solo Bottle" {
			item.Type = inventory.TypeFinishedGood
		}
		if err := repo.CreateItem(ctx, tenantID, item); err != nil {
			return err
		}
		itemIDs[def.name] = item.ID
	}

	// Recipes keyed by product name → ingredient consumption per sale.
	recipes := map[string][]inventory.RecipeItem{
		"Katsudon": {
			{InventoryItemID: itemIDs["Rice"], Qty: 0.2},
			{InventoryItemID: itemIDs["Pork Belly"], Qty: 0.15},
			{InventoryItemID: itemIDs["Egg"], Qty: 1},
			{InventoryItemID: itemIDs["Cooking Oil"], Qty: 0.05},
		},
		"Pork Tapa": {
			{InventoryItemID: itemIDs["Rice"], Qty: 0.15},
			{InventoryItemID: itemIDs["Pork Belly"], Qty: 0.12},
			{InventoryItemID: itemIDs["Egg"], Qty: 1},
		},
		"C2 Solo": {
			{InventoryItemID: itemIDs["C2 Solo Bottle"], Qty: 1},
		},
	}
	for productName, recipeItems := range recipes {
		var productID string
		err := db.QueryRow(ctx, `
			SELECT id FROM products WHERE tenant_id = $1 AND name = $2 AND deleted_at IS NULL`,
			tenantID, productName).Scan(&productID)
		if err != nil {
			logger.Warn("recipe product not found — skipping", "product", productName)
			continue
		}
		if err := repo.ReplaceRecipe(ctx, tenantID, productID, recipeItems); err != nil {
			return err
		}
	}

	logger.Info("inventory seeded", "units", len(units), "items", len(items), "recipes", len(recipes))
	return nil
}
