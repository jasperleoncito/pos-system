// Package dto defines request/response payloads with validation tags.
package dto

type RegisterRequest struct {
	FullName     string `json:"full_name" binding:"required,min=2,max=120"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8,max=72"`
	BusinessName string `json:"business_name" binding:"required,min=2,max=120"`
	BusinessSlug string `json:"business_slug" binding:"required,min=2,max=60"`
	Plan         string `json:"plan" binding:"required,oneof=monthly yearly"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type SwitchTenantRequest struct {
	TenantID string `json:"tenant_id" binding:"required,uuid"`
}
