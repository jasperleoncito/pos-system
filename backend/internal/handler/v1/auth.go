package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jasperleoncito/pos-system/backend/internal/dto"
	"github.com/jasperleoncito/pos-system/backend/internal/middleware"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/response"
	"github.com/jasperleoncito/pos-system/backend/internal/service"
)

// AuthHandler exposes authentication endpoints. Handlers only validate
// input and delegate to the service layer.
type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func requestMeta(c *gin.Context) service.RequestMeta {
	return service.RequestMeta{
		IP:         c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		DeviceName: c.GetHeader("X-Device-Name"),
	}
}

// respondError maps a service error onto the error envelope.
func respondError(c *gin.Context, err error) {
	appErr := apperror.From(err)
	response.Error(c, appErr.HTTPStatus(), appErr.Message, appErr.Errors...)
}

// bindJSON binds and reports validation problems in the envelope shape.
func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		var details []string
		var vErr interface{ Error() string }
		if errors.As(err, &vErr) {
			details = append(details, vErr.Error())
		}
		response.Error(c, http.StatusUnprocessableEntity, "validation failed", details...)
		return false
	}
	return true
}

// Register godoc
//
//	@Summary	Register an owner account with their first business
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.RegisterRequest	true	"Registration payload"
//	@Success	201		{object}	response.Envelope
//	@Failure	409		{object}	response.ErrorEnvelope
//	@Router		/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.auth.Register(c.Request.Context(),
		req.FullName, req.Email, req.Password, req.BusinessName, req.BusinessSlug, requestMeta(c))
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, "account created — check your inbox to verify your email", result)
}

// Login godoc
//
//	@Summary	Login with email and password
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.LoginRequest	true	"Credentials"
//	@Success	200		{object}	response.Envelope
//	@Failure	401		{object}	response.ErrorEnvelope
//	@Router		/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password, requestMeta(c))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "welcome back", result)
}

// Refresh godoc
//
//	@Summary	Rotate the refresh token and get a new access token
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.RefreshRequest	true	"Refresh token"
//	@Success	200		{object}	response.Envelope
//	@Failure	401		{object}	response.ErrorEnvelope
//	@Router		/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken, requestMeta(c))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "token refreshed", result)
}

// Logout godoc
//
//	@Summary	Logout the current device session
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.LogoutRequest	true	"Refresh token"
//	@Success	200		{object}	response.Envelope
//	@Router		/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken, requestMeta(c)); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "logged out", nil)
}

// LogoutAll godoc
//
//	@Summary	Logout from every device
//	@Tags		auth
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := c.GetString(middleware.CtxUserID)
	if err := h.auth.LogoutAll(c.Request.Context(), userID, requestMeta(c)); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "logged out from all devices", nil)
}

// Sessions godoc
//
//	@Summary	List active device sessions
//	@Tags		auth
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/auth/sessions [get]
func (h *AuthHandler) Sessions(c *gin.Context) {
	sessions, err := h.auth.Sessions(c.Request.Context(), c.GetString(middleware.CtxUserID))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "", gin.H{"sessions": sessions, "current_session_id": c.GetString(middleware.CtxSessionID)})
}

// RevokeSession godoc
//
//	@Summary	Revoke one of your device sessions
//	@Tags		auth
//	@Security	BearerAuth
//	@Produce	json
//	@Param		id	path		string	true	"Session ID"
//	@Success	200	{object}	response.Envelope
//	@Router		/auth/sessions/{id} [delete]
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	err := h.auth.RevokeSession(c.Request.Context(), c.GetString(middleware.CtxUserID), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "session revoked", nil)
}

// ForgotPassword godoc
//
//	@Summary	Request a password reset email
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.ForgotPasswordRequest	true	"Email"
//	@Success	200		{object}	response.Envelope
//	@Router		/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	h.auth.ForgotPassword(c.Request.Context(), req.Email)
	response.OK(c, "if that email is registered, a reset link is on its way", nil)
}

// ResetPassword godoc
//
//	@Summary	Reset password using an emailed token
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.ResetPasswordRequest	true	"Token and new password"
//	@Success	200		{object}	response.Envelope
//	@Failure	422		{object}	response.ErrorEnvelope
//	@Router		/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.auth.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "password updated — please sign in again", nil)
}

// VerifyEmail godoc
//
//	@Summary	Verify email using an emailed token
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.VerifyEmailRequest	true	"Verification token"
//	@Success	200		{object}	response.Envelope
//	@Failure	422		{object}	response.ErrorEnvelope
//	@Router		/auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := h.auth.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "email verified", nil)
}

// ResendVerification godoc
//
//	@Summary	Resend the verification email
//	@Tags		auth
//	@Security	BearerAuth
//	@Produce	json
//	@Success	200	{object}	response.Envelope
//	@Router		/auth/resend-verification [post]
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	if err := h.auth.ResendVerification(c.Request.Context(), c.GetString(middleware.CtxUserID)); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "verification email sent", nil)
}

// SwitchTenant godoc
//
//	@Summary	Switch the active business
//	@Tags		auth
//	@Security	BearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		payload	body		dto.SwitchTenantRequest	true	"Target tenant"
//	@Success	200		{object}	response.Envelope
//	@Failure	403		{object}	response.ErrorEnvelope
//	@Router		/auth/switch-tenant [post]
func (h *AuthHandler) SwitchTenant(c *gin.Context) {
	var req dto.SwitchTenantRequest
	if !bindJSON(c, &req) {
		return
	}
	access, membership, err := h.auth.SwitchTenant(c.Request.Context(),
		c.GetString(middleware.CtxUserID), c.GetString(middleware.CtxSessionID), req.TenantID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, "switched business", gin.H{"access_token": access, "active_tenant": membership})
}
