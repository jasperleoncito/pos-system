package seed

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/catalog"
	"github.com/jasperleoncito/pos-system/backend/internal/repository/postgres"
)

// pesos converts whole pesos to centavos.
func pesos(p int64) int64 { return p * 100 }

type menuProduct struct {
	name        string
	description string
	price       int64    // centavos
	variants    []catalog.Variant
	modGroups   []string // modifier group names to attach
}

type menuCategory struct {
	name        string
	description string
	products    []menuProduct
}

// teresasMenu is the full menu transcribed from the reference photos.
var teresasMenu = []menuCategory{
	{
		name:        "Big Plate Meal",
		description: "Meals are served with Rice, Iced Tea or Lemonade and choice of (1) side dish — Mac & Cheese, Buttered Corn, French Fries",
		products: []menuProduct{
			{name: "BBQ Pork Ribs", price: pesos(160), modGroups: []string{"Choice of Side Dish", "Choice of Drink"}},
			{name: "Pot Roast Pork Belly w/ Orange Sauce", price: pesos(140), modGroups: []string{"Choice of Side Dish", "Choice of Drink"}},
			{name: "Herbed Grilled Double Porkchop", price: pesos(140), modGroups: []string{"Choice of Side Dish", "Choice of Drink"}},
			{name: "Grilled Boneless Teriyaki Chicken", price: pesos(140), modGroups: []string{"Choice of Side Dish", "Choice of Drink"}},
			{name: "Grilled Boneless BBQ Chicken", price: pesos(140), modGroups: []string{"Choice of Side Dish", "Choice of Drink"}},
			{name: "Chicken Tenders", price: pesos(140), modGroups: []string{"Choice of Side Dish", "Choice of Drink"}},
			{name: "Pot Roast Beef w/ Mushroom Gravy Sauce", price: pesos(150), modGroups: []string{"Choice of Side Dish", "Choice of Drink"}},
		},
	},
	{
		name:        "Silog",
		description: "Served with Garlic Rice, Egg & Atchara",
		products: []menuProduct{
			{name: "Pork Tapa", price: pesos(75)},
			{name: "Beef Bulgogi", price: pesos(95)},
			{name: "Boneless Chicken Tocino", price: pesos(85)},
			{name: "Crispy Pork Belly", price: pesos(115)},
		},
	},
	{
		name:        "Favorites",
		description: "Served with Rice",
		products: []menuProduct{
			{name: "Dinakdakan", price: pesos(90)},
			{name: "Chicken Inasal", price: pesos(130)},
			{name: "Tuna Fillet with Soy Sauce & Onions", price: pesos(75)},
			{name: "Braised Beef", price: pesos(90)},
			{name: "Garlic-Mushroom Beef", price: pesos(90)},
			{name: "Boneless Fried Chicken", price: pesos(80)},
			{name: "Chinese Lemon Chicken", price: pesos(75)},
			{name: "Katsu Curry", price: pesos(90)},
			{name: "Katsudon", price: pesos(95)},
			{name: "Beef Pinapaitan", price: pesos(100)},
		},
	},
	{
		name:        "Pasta",
		description: "Served with Garlic-Butter Toasted Bread",
		products: []menuProduct{
			{name: "Baked Spaghetti", price: pesos(100)},
			{name: "Creamy Cajun & Chicken-Penne", price: pesos(120)},
		},
	},
	{
		name: "Side Order",
		products: []menuProduct{
			{name: "Tortang Talong at Giniling", price: pesos(75)},
			{name: "Laing with Hipon", price: pesos(60)},
			{name: "Crinkle Cut French Fries & Dip", price: pesos(75), modGroups: []string{"Choice of Dip"}},
		},
	},
	{
		name: "Extra Order",
		products: []menuProduct{
			{name: "Plain Rice", price: pesos(15)},
			{name: "Garlic Rice", price: pesos(18)},
			{name: "Egg", price: pesos(15)},
			{name: "Dips", price: pesos(15), modGroups: []string{"Choice of Dip"}},
			{name: "Mushroom Gravy", price: pesos(15)},
			{name: "Side Dish", price: pesos(45)},
		},
	},
	{
		name: "Softdrinks",
		products: []menuProduct{
			{name: "C2 Solo", price: pesos(24)},
			{name: "Minute Maid", price: pesos(24)},
			{name: "Mismo", price: pesos(26), variants: []catalog.Variant{
				{Name: "Coke"}, {Name: "Sprite"}, {Name: "Royal"},
			}},
			{name: "Pepsi", price: pesos(27)},
			{name: "Mountain Dew", price: pesos(27)},
			{name: "Coke Zero Swakto", price: pesos(18)},
			{name: "Lipton Iced Tea", price: pesos(38)},
			{name: "Nestea Iced Tea", price: pesos(38)},
			{name: "Real Leaf Iced Tea", price: pesos(38)},
			{name: "Smart-C", price: pesos(38)},
			{name: "Four Seasons in Can", price: pesos(38)},
			{name: "Pineapple Juice in Can", price: pesos(38)},
			{name: "Bottled Water", price: pesos(12)},
			{name: "RC Products 1L", price: pesos(48), variants: []catalog.Variant{
				{Name: "RC Cola"}, {Name: "Fruit Soda"}, {Name: "Root Beer"}, {Name: "Soda Lemon"},
			}},
			{name: "Coca-Cola Products 1.5L", price: pesos(85)},
		},
	},
}

var teresasModifierGroups = []struct {
	group   catalog.ModifierGroup
	options []string
}{
	{
		group:   catalog.ModifierGroup{Name: "Choice of Side Dish", MinSelect: 1, MaxSelect: 1, IsRequired: true, SortOrder: 0},
		options: []string{"Mac & Cheese", "Buttered Corn", "French Fries"},
	},
	{
		group:   catalog.ModifierGroup{Name: "Choice of Drink", MinSelect: 1, MaxSelect: 1, IsRequired: true, SortOrder: 1},
		options: []string{"Iced Tea", "Lemonade"},
	},
	{
		group:   catalog.ModifierGroup{Name: "Choice of Dip", MinSelect: 1, MaxSelect: 1, IsRequired: true, SortOrder: 2},
		options: []string{"Sriracha Mayo", "Tartar", "Honey Mustard", "BBQ Mayo"},
	},
}

// seedMenu populates Teresa's Eatery catalog. Skips when the tenant
// already has categories (idempotent).
func seedMenu(ctx context.Context, db *pgxpool.Pool, tenantID string, logger *slog.Logger) error {
	categories := postgres.NewCategoryRepo(db)
	products := postgres.NewProductRepo(db)
	modifiers := postgres.NewModifierRepo(db)
	taxes := postgres.NewTaxRepo(db)

	existing, err := categories.List(ctx, tenantID, false)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		logger.Info("menu already seeded — skipping", "categories", len(existing))
		return nil
	}

	// Default PH VAT, price-inclusive.
	vat := &catalog.Tax{Name: "VAT 12% (inclusive)", RatePercent: 12, IsInclusive: true, IsDefault: true, IsActive: true}
	if err := taxes.Create(ctx, tenantID, vat); err != nil {
		return err
	}

	// Modifier groups first so products can link to them by name.
	groupIDs := map[string]string{}
	for _, mg := range teresasModifierGroups {
		group := mg.group
		if err := modifiers.CreateGroup(ctx, tenantID, &group); err != nil {
			return err
		}
		options := make([]catalog.Modifier, len(mg.options))
		for i, name := range mg.options {
			options[i] = catalog.Modifier{Name: name, IsActive: true}
		}
		if err := modifiers.ReplaceModifiers(ctx, tenantID, group.ID, options); err != nil {
			return err
		}
		groupIDs[group.Name] = group.ID
	}

	productCount := 0
	for sortOrder, mc := range teresasMenu {
		category := &catalog.Category{
			Name: mc.name, Description: mc.description, SortOrder: sortOrder, IsActive: true,
		}
		if err := categories.Create(ctx, tenantID, category); err != nil {
			return err
		}

		for i, mp := range mc.products {
			product := &catalog.Product{
				CategoryID: category.ID, TaxID: &vat.ID, Name: mp.name,
				Description: mp.description, BasePrice: mp.price,
				IsActive: true, SortOrder: i,
			}
			if err := products.Create(ctx, tenantID, product); err != nil {
				return fmt.Errorf("failed to seed product %q: %w", mp.name, err)
			}
			if len(mp.variants) > 0 {
				if err := products.ReplaceVariants(ctx, tenantID, product.ID, mp.variants); err != nil {
					return err
				}
			}
			if len(mp.modGroups) > 0 {
				ids := make([]string, len(mp.modGroups))
				for j, name := range mp.modGroups {
					ids[j] = groupIDs[name]
				}
				if err := products.ReplaceModifierGroups(ctx, tenantID, product.ID, ids); err != nil {
					return err
				}
			}
			productCount++
		}
	}

	logger.Info("menu seeded",
		"categories", len(teresasMenu), "products", productCount, "modifier_groups", len(teresasModifierGroups))
	return nil
}
