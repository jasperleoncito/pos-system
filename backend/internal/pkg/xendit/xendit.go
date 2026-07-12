// Package xendit is a minimal client for the Xendit Invoice API — the
// hosted checkout page used for subscription payments. Amounts are in
// whole PHP (the rest of the codebase uses centavos; callers convert).
package xendit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.xendit.co"

type Client struct {
	secretKey string
	baseURL   string
	http      *http.Client
}

func New(secretKey string) *Client {
	return &Client{
		secretKey: secretKey,
		baseURL:   defaultBaseURL,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

// Configured reports whether a secret key is present — dev environments
// may run without one and must fail checkout cleanly.
func (c *Client) Configured() bool { return c.secretKey != "" }

// CreateInvoiceRequest maps to POST /v2/invoices.
type CreateInvoiceRequest struct {
	ExternalID         string  `json:"external_id"`
	Amount             float64 `json:"amount"` // whole PHP
	PayerEmail         string  `json:"payer_email,omitempty"`
	Description        string  `json:"description"`
	Currency           string  `json:"currency"`
	SuccessRedirectURL string  `json:"success_redirect_url,omitempty"`
	FailureRedirectURL string  `json:"failure_redirect_url,omitempty"`
	InvoiceDuration    int     `json:"invoice_duration,omitempty"` // seconds
}

// Invoice is the subset of the invoice response we persist / reconcile.
type Invoice struct {
	ID             string  `json:"id"`
	InvoiceURL     string  `json:"invoice_url"`
	Status         string  `json:"status"` // PENDING | PAID | SETTLED | EXPIRED
	ExpiryDate     string  `json:"expiry_date"`
	ExternalID     string  `json:"external_id"`
	PaidAmount     float64 `json:"paid_amount"`
	PaidAt         string  `json:"paid_at"`
	PaymentChannel string  `json:"payment_channel"`
}

// IsPaid reports whether the invoice status means money arrived.
func (i Invoice) IsPaid() bool { return i.Status == "PAID" || i.Status == "SETTLED" }

// CreateInvoice creates a hosted invoice and returns its checkout URL.
func (c *Client) CreateInvoice(ctx context.Context, in CreateInvoiceRequest) (*Invoice, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("xendit is not configured (XENDIT_SECRET_KEY is empty)")
	}
	if in.Currency == "" {
		in.Currency = "PHP"
	}

	body, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("failed to encode invoice request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v2/invoices", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build invoice request: %w", err)
	}
	req.SetBasicAuth(c.secretKey, "")
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xendit request failed: %w", err)
	}
	defer res.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read xendit response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("xendit returned %d: %s", res.StatusCode, string(payload))
	}

	var invoice Invoice
	if err := json.Unmarshal(payload, &invoice); err != nil {
		return nil, fmt.Errorf("failed to decode xendit invoice: %w", err)
	}
	return &invoice, nil
}

// GetInvoice fetches an invoice by ID. Used to reconcile payment status
// directly with Xendit (the return page polls this) so confirmation never
// depends on the webhook being delivered.
func (c *Client) GetInvoice(ctx context.Context, id string) (*Invoice, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("xendit is not configured (XENDIT_SECRET_KEY is empty)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v2/invoices/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build get-invoice request: %w", err)
	}
	req.SetBasicAuth(c.secretKey, "")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xendit request failed: %w", err)
	}
	defer res.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read xendit response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("xendit returned %d: %s", res.StatusCode, string(payload))
	}

	var invoice Invoice
	if err := json.Unmarshal(payload, &invoice); err != nil {
		return nil, fmt.Errorf("failed to decode xendit invoice: %w", err)
	}
	return &invoice, nil
}

// InvoiceCallback is the webhook body Xendit POSTs on invoice events.
// Only the fields we consume are declared.
type InvoiceCallback struct {
	ID             string  `json:"id"`
	ExternalID     string  `json:"external_id"`
	Status         string  `json:"status"` // PAID | SETTLED | EXPIRED
	PaidAmount     float64 `json:"paid_amount"`
	PaidAt         string  `json:"paid_at"`
	PaymentMethod  string  `json:"payment_method"`
	PaymentChannel string  `json:"payment_channel"`
	PayerEmail     string  `json:"payer_email"`
	Currency       string  `json:"currency"`
}

// IsPaid reports whether the callback status means money arrived —
// Xendit may send PAID, SETTLED, or both.
func (c InvoiceCallback) IsPaid() bool {
	return c.Status == "PAID" || c.Status == "SETTLED"
}
