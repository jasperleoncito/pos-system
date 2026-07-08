package dto

type UpdateTenantSettingsRequest struct {
	PrimaryColor   string `json:"primary_color" binding:"required,hexcolor"`
	SecondaryColor string `json:"secondary_color" binding:"required,hexcolor"`
	AccentColor    string `json:"accent_color" binding:"required,hexcolor"`
	ReceiptHeader  string `json:"receipt_header" binding:"max=500"`
	ReceiptFooter  string `json:"receipt_footer" binding:"max=500"`
	ContactNumber  string `json:"contact_number" binding:"max=40"`
	Facebook       string `json:"facebook" binding:"max=200"`
	Website        string `json:"website" binding:"max=200"`
	Address        string `json:"address" binding:"max=400"`
	TaxLabel       string `json:"tax_label" binding:"max=60"`
	TaxID          string `json:"tax_id" binding:"max=60"`
}

type SetTenantStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active suspended"`
}

type SetTenantPlanRequest struct {
	Plan string `json:"plan" binding:"required,oneof=free standard premium"`
}
