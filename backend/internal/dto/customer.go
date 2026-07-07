package dto

type CustomerRequest struct {
	FullName string `json:"full_name" binding:"required,min=1,max=120"`
	Phone    string `json:"phone" binding:"max=40"`
	Email    string `json:"email" binding:"omitempty,email"`
	Birthday string `json:"birthday" binding:"omitempty,datetime=2006-01-02"`
	Notes    string `json:"notes" binding:"max=500"`
	IsActive *bool  `json:"is_active"`
}

type LoyaltySettingsRequest struct {
	IsEnabled        *bool   `json:"is_enabled"`
	EarnRate         int64   `json:"earn_rate" binding:"required,min=1"`
	RedeemValue      int64   `json:"redeem_value" binding:"required,min=1"`
	SilverThreshold  int64   `json:"silver_threshold" binding:"min=0"`
	GoldThreshold    int64   `json:"gold_threshold" binding:"min=0"`
	VIPThreshold     int64   `json:"vip_threshold" binding:"min=0"`
	SilverMultiplier float64 `json:"silver_multiplier" binding:"min=1,max=99"`
	GoldMultiplier   float64 `json:"gold_multiplier" binding:"min=1,max=99"`
	VIPMultiplier    float64 `json:"vip_multiplier" binding:"min=1,max=99"`
}
