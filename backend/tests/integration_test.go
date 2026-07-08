//go:build integration

// Package tests runs black-box integration tests against a running
// stack (docker compose up). Run with:
//
//	go test -tags integration ./tests/ -v
//
// Requires the seeded demo tenant (docker compose exec backend go run ./cmd/seed).
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

var baseURL = func() string {
	if v := os.Getenv("POS_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:7642/api/v1"
}()

type envelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func call(t *testing.T, method, path, token string, body any) (int, envelope) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	var env envelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatalf("%s %s: decode: %v", method, path, err)
	}
	return res.StatusCode, env
}

func login(t *testing.T, email string) (token string, refresh string) {
	t.Helper()
	status, env := call(t, http.MethodPost, "/auth/login", "", map[string]string{
		"email": email, "password": "password123",
	})
	if status != http.StatusOK {
		t.Fatalf("login %s: status %d (%s)", email, status, env.Message)
	}
	var data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("login decode: %v", err)
	}
	return data.AccessToken, data.RefreshToken
}

// TestAuthRefreshRotation verifies refresh tokens rotate and that a
// replayed (already-used) refresh token is rejected.
func TestAuthRefreshRotation(t *testing.T) {
	_, refresh := login(t, "cashier@teresas.ph")

	status, env := call(t, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": refresh})
	if status != http.StatusOK {
		t.Fatalf("first refresh failed: %d (%s)", status, env.Message)
	}
	// Replaying the consumed token must fail.
	status, _ = call(t, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": refresh})
	if status == http.StatusOK {
		t.Fatal("replayed refresh token was accepted — rotation is broken")
	}
}

// TestRBACDenies verifies role gates on sensitive routes.
func TestRBACDenies(t *testing.T) {
	kitchenToken, _ := login(t, "kitchen@teresas.ph")
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/customers"},
		{http.MethodGet, "/analytics/overview"},
		{http.MethodGet, "/audit-logs"},
		{http.MethodGet, "/employees"},
		{http.MethodPost, "/products"},
	}
	for _, tc := range cases {
		status, _ := call(t, tc.method, tc.path, kitchenToken, map[string]string{})
		if status != http.StatusForbidden {
			t.Errorf("%s %s as kitchen: want 403, got %d", tc.method, tc.path, status)
		}
	}
}

// TestTenantIsolation registers a fresh tenant, creates a customer in
// it, and proves the demo tenant cannot read it by ID.
func TestTenantIsolation(t *testing.T) {
	suffix := time.Now().UnixNano() % 1_000_000_000
	email := fmt.Sprintf("iso%d@test.ph", suffix)
	slug := fmt.Sprintf("iso-cafe-%d", suffix)

	status, env := call(t, http.MethodPost, "/auth/register", "", map[string]string{
		"full_name": "Iso Owner", "email": email, "password": "password123",
		"business_name": "Iso Cafe", "business_slug": slug,
	})
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("register: %d (%s)", status, env.Message)
	}
	var reg struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(env.Data, &reg); err != nil || reg.AccessToken == "" {
		t.Fatalf("register token decode failed: %v", err)
	}

	status, env = call(t, http.MethodPost, "/customers", reg.AccessToken, map[string]any{
		"full_name": "Secret Customer", "phone": fmt.Sprintf("099%d", suffix),
		"email": "", "notes": "", "is_active": true,
	})
	if status != http.StatusCreated {
		t.Fatalf("create customer in new tenant: %d (%s)", status, env.Message)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &created); err != nil {
		t.Fatalf("customer decode: %v", err)
	}

	teresaToken, _ := login(t, "manager@teresas.ph")
	status, _ = call(t, http.MethodGet, "/customers/"+created.ID, teresaToken, nil)
	if status != http.StatusNotFound {
		t.Fatalf("cross-tenant customer read: want 404, got %d", status)
	}
}

// TestOrderFlowAndInventoryDeduction runs a real sale end to end:
// price server-side, pay cash, verify recipe deduction and settle
// idempotency (a second payment attempt must fail).
func TestOrderFlowAndInventoryDeduction(t *testing.T) {
	manager, _ := login(t, "manager@teresas.ph")

	// Locate Katsudon (recipe: Rice 0.2 per unit) and current Rice stock.
	status, env := call(t, http.MethodGet, "/products?limit=200&search=Katsudon", manager, nil)
	if status != http.StatusOK {
		t.Fatalf("products: %d", status)
	}
	var products []struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		ModifierGroups []struct {
			IsRequired bool `json:"is_required"`
			Modifiers  []struct {
				ID string `json:"id"`
			} `json:"modifiers"`
		} `json:"modifier_groups"`
	}
	if err := json.Unmarshal(env.Data, &products); err != nil || len(products) == 0 {
		t.Fatalf("katsudon not found: %v", err)
	}
	product := products[0]
	// Satisfy required modifier groups with their first option.
	var modifierIDs []string
	for _, g := range product.ModifierGroups {
		if g.IsRequired && len(g.Modifiers) > 0 {
			modifierIDs = append(modifierIDs, g.Modifiers[0].ID)
		}
	}
	if modifierIDs == nil {
		modifierIDs = []string{}
	}

	riceBefore := itemStock(t, manager, "Rice")

	// Drawer for cash (idempotent: may already be open).
	call(t, http.MethodPost, "/cash-drawer/open", manager, map[string]any{"opening_float": 100000})

	status, env = call(t, http.MethodPost, "/orders", manager, map[string]any{
		"order_type": "takeout",
		"items": []map[string]any{{
			"product_id": product.ID, "qty": 1, "modifier_ids": modifierIDs,
		}},
	})
	if status != http.StatusCreated {
		t.Fatalf("create order: %d (%s)", status, env.Message)
	}
	var order struct {
		ID    string `json:"id"`
		Total int64  `json:"total"`
	}
	if err := json.Unmarshal(env.Data, &order); err != nil {
		t.Fatalf("order decode: %v", err)
	}
	if order.Total <= 0 {
		t.Fatalf("server-side pricing produced total %d", order.Total)
	}

	status, env = call(t, http.MethodPost, "/orders/"+order.ID+"/payments", manager, map[string]any{
		"payments": []map[string]any{{"method": "cash", "amount": order.Total}},
	})
	if status != http.StatusOK {
		t.Fatalf("pay: %d (%s)", status, env.Message)
	}
	var paid struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(env.Data, &paid)
	if paid.Status != "completed" {
		t.Fatalf("order status after pay: %s", paid.Status)
	}

	// Recipe deduction: Rice drops by exactly 0.2.
	riceAfter := itemStock(t, manager, "Rice")
	if diff := riceBefore - riceAfter; diff < 0.199 || diff > 0.201 {
		t.Fatalf("rice deduction: want 0.2, got %.3f (before %.3f after %.3f)", diff, riceBefore, riceAfter)
	}

	// Idempotency: settling again must be rejected and must not deduct.
	status, _ = call(t, http.MethodPost, "/orders/"+order.ID+"/payments", manager, map[string]any{
		"payments": []map[string]any{{"method": "cash", "amount": order.Total}},
	})
	if status == http.StatusOK {
		t.Fatal("second settle of a completed order was accepted")
	}
	if again := itemStock(t, manager, "Rice"); again != riceAfter {
		t.Fatalf("stock moved on rejected settle: %.3f → %.3f", riceAfter, again)
	}
}

func itemStock(t *testing.T, token, name string) float64 {
	t.Helper()
	status, env := call(t, http.MethodGet, "/inventory/items?search="+name, token, nil)
	if status != http.StatusOK {
		t.Fatalf("inventory items: %d", status)
	}
	var items []struct {
		Name         string  `json:"name"`
		CurrentStock float64 `json:"current_stock"`
	}
	if err := json.Unmarshal(env.Data, &items); err != nil {
		t.Fatalf("items decode: %v", err)
	}
	for _, item := range items {
		if item.Name == name {
			return item.CurrentStock
		}
	}
	t.Fatalf("item %s not found", name)
	return 0
}

// TestSecurityHeaders verifies the hardening headers are on responses.
func TestSecurityHeaders(t *testing.T) {
	res, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	defer res.Body.Close()
	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	} {
		if got := res.Header.Get(header); got != want {
			t.Errorf("header %s: want %q, got %q", header, want, got)
		}
	}
}
