package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/auth"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/rbac"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/mailer"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/password"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/token"
)

const (
	purposePasswordReset = "password_reset"
	purposeVerifyEmail   = "verify_email"

	passwordResetTTL = 30 * time.Minute
	verifyEmailTTL   = 48 * time.Hour
)

// TokenStore is the ephemeral one-time-token contract (Redis-backed).
type TokenStore interface {
	Issue(ctx context.Context, purpose, userID string, ttl time.Duration) (string, error)
	Consume(ctx context.Context, purpose, token string) (string, error)
}

// EmailSender delivers transactional mail.
type EmailSender interface {
	Send(to, subject, htmlBody string) error
}

// AuthService implements registration, login, token lifecycle, and
// account recovery flows.
type AuthService struct {
	users       auth.UserRepository
	sessions    auth.SessionRepository
	tenants     tenant.Repository
	settings    tenant.SettingsRepository
	memberships tenant.MembershipRepository
	tokens      *token.Manager
	otp         TokenStore
	mailer      EmailSender
	auditor     *AuditService
	logger      *slog.Logger
	appBaseURL  string
	appName     string
}

type AuthServiceDeps struct {
	Users       auth.UserRepository
	Sessions    auth.SessionRepository
	Tenants     tenant.Repository
	Settings    tenant.SettingsRepository
	Memberships tenant.MembershipRepository
	Tokens      *token.Manager
	OTP         TokenStore
	Mailer      EmailSender
	Auditor     *AuditService
	Logger      *slog.Logger
	AppBaseURL  string
	AppName     string
}

func NewAuthService(d AuthServiceDeps) *AuthService {
	return &AuthService{
		users: d.Users, sessions: d.Sessions, tenants: d.Tenants, settings: d.Settings,
		memberships: d.Memberships, tokens: d.Tokens, otp: d.OTP, mailer: d.Mailer,
		auditor: d.Auditor, logger: d.Logger, appBaseURL: d.AppBaseURL, appName: d.AppName,
	}
}

// RequestMeta carries client information for sessions and audit rows.
type RequestMeta struct {
	IP         string
	UserAgent  string
	DeviceName string
}

// AuthResult is returned by flows that establish a session.
type AuthResult struct {
	User         *auth.User          `json:"user"`
	Memberships  []tenant.Membership `json:"memberships"`
	ActiveTenant *tenant.Membership  `json:"active_tenant,omitempty"`
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
}

// Register creates an owner account together with their first business.
func (s *AuthService) Register(ctx context.Context, fullName, email, plainPassword, businessName, businessSlug string, meta RequestMeta) (*AuthResult, error) {
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	user := &auth.User{
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: hash,
		FullName:     strings.TrimSpace(fullName),
		Status:       "active",
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	t := &tenant.Tenant{
		Name:        strings.TrimSpace(businessName),
		Slug:        slugify(businessSlug),
		OwnerUserID: user.ID,
		Status:      "active",
		Currency:    "PHP",
		Timezone:    "Asia/Manila",
	}
	if err := s.tenants.Create(ctx, t); err != nil {
		return nil, err
	}
	if err := s.settings.Create(ctx, &tenant.Settings{
		TenantID:       t.ID,
		PrimaryColor:   "#DC2626",
		SecondaryColor: "#F87171",
		AccentColor:    "#CA8A04",
	}); err != nil {
		return nil, err
	}
	membership := &tenant.Membership{TenantID: t.ID, UserID: user.ID, Role: string(rbac.RoleOwner)}
	if err := s.memberships.Create(ctx, membership); err != nil {
		return nil, err
	}

	s.sendVerificationEmail(ctx, user)

	s.auditor.Record(audit.Log{
		TenantID: t.ID, UserID: user.ID, Action: "auth.register",
		EntityType: "user", EntityID: user.ID, IP: meta.IP, UserAgent: meta.UserAgent,
	})

	return s.establishSession(ctx, user, meta)
}

// Login authenticates by email/password and opens a device session.
func (s *AuthService) Login(ctx context.Context, email, plainPassword string, meta RequestMeta) (*AuthResult, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Uniform error hides whether the account exists.
		return nil, apperror.Unauthorized("invalid email or password")
	}
	if !password.Verify(user.PasswordHash, plainPassword) {
		return nil, apperror.Unauthorized("invalid email or password")
	}
	if user.Status != "active" {
		return nil, apperror.Forbidden("account is disabled")
	}

	result, err := s.establishSession(ctx, user, meta)
	if err != nil {
		return nil, err
	}

	tenantID := ""
	if result.ActiveTenant != nil {
		tenantID = result.ActiveTenant.TenantID
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: user.ID, Action: "auth.login",
		EntityType: "user", EntityID: user.ID, IP: meta.IP, UserAgent: meta.UserAgent,
	})
	return result, nil
}

// Refresh rotates the refresh token and issues a new access token.
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string, meta RequestMeta) (*AuthResult, error) {
	sess, err := s.sessions.GetByTokenHash(ctx, token.HashToken(rawRefreshToken))
	if err != nil {
		return nil, apperror.Unauthorized("invalid refresh token")
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, apperror.Unauthorized("refresh token expired")
	}

	user, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, apperror.Unauthorized("account not found")
	}
	if user.Status != "active" {
		return nil, apperror.Forbidden("account is disabled")
	}

	raw, hash, err := token.NewRefreshToken()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.sessions.Rotate(ctx, sess.ID, hash, time.Now().Add(s.tokens.RefreshTTL())); err != nil {
		return nil, err
	}

	return s.buildAuthResult(ctx, user, sess.ID, raw)
}

// Logout revokes the session belonging to the given refresh token.
func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string, meta RequestMeta) error {
	sess, err := s.sessions.GetByTokenHash(ctx, token.HashToken(rawRefreshToken))
	if err != nil {
		return nil // already gone â€” logout is idempotent
	}
	if err := s.sessions.Revoke(ctx, sess.ID); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		UserID: sess.UserID, Action: "auth.logout",
		EntityType: "session", EntityID: sess.ID, IP: meta.IP, UserAgent: meta.UserAgent,
	})
	return nil
}

// LogoutAll revokes every active session for the user.
func (s *AuthService) LogoutAll(ctx context.Context, userID string, meta RequestMeta) error {
	if err := s.sessions.RevokeAllForUser(ctx, userID); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		UserID: userID, Action: "auth.logout_all",
		EntityType: "user", EntityID: userID, IP: meta.IP, UserAgent: meta.UserAgent,
	})
	return nil
}

// Sessions lists the user's active device sessions.
func (s *AuthService) Sessions(ctx context.Context, userID string) ([]auth.DeviceSession, error) {
	return s.sessions.ListActiveByUser(ctx, userID)
}

// RevokeSession revokes one of the user's own sessions.
func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	sessions, err := s.sessions.ListActiveByUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		if sess.ID == sessionID {
			return s.sessions.Revoke(ctx, sessionID)
		}
	}
	return apperror.NotFound("session")
}

// ForgotPassword issues a reset token and emails it. Always succeeds to
// avoid leaking which emails are registered.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return
	}
	resetToken, err := s.otp.Issue(ctx, purposePasswordReset, user.ID, passwordResetTTL)
	if err != nil {
		s.logger.Error("failed to issue reset token", "error", err)
		return
	}
	link := fmt.Sprintf("%s/reset-password?token=%s", s.appBaseURL, resetToken)
	body := mailer.Render(mailer.Email{
		AppName:    s.appName,
		Title:      "Reset your password",
		Intro:      fmt.Sprintf("Hi %s, we received a request to reset your password. The link below expires in 30 minutes.", user.FullName),
		ButtonText: "Reset password",
		ButtonURL:  link,
		FooterNote: "If you didn't request this, you can safely ignore this email â€” your password will not change.",
	})
	if err := s.mailer.Send(user.Email, "Reset your password", body); err != nil {
		s.logger.Error("failed to send reset email", "error", err)
	}
}

// ResetPassword consumes a reset token and sets the new password.
func (s *AuthService) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	userID, err := s.otp.Consume(ctx, purposePasswordReset, resetToken)
	if err != nil {
		return apperror.Internal(err)
	}
	if userID == "" {
		return apperror.Validation("reset link is invalid or has expired")
	}
	hash, err := password.Hash(newPassword)
	if err != nil {
		return apperror.Internal(err)
	}
	if err := s.users.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	// Force re-login everywhere after a password change.
	if err := s.sessions.RevokeAllForUser(ctx, userID); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		UserID: userID, Action: "auth.password_reset", EntityType: "user", EntityID: userID,
	})
	return nil
}

// VerifyEmail consumes a verification token.
func (s *AuthService) VerifyEmail(ctx context.Context, verifyToken string) error {
	userID, err := s.otp.Consume(ctx, purposeVerifyEmail, verifyToken)
	if err != nil {
		return apperror.Internal(err)
	}
	if userID == "" {
		return apperror.Validation("verification link is invalid or has expired")
	}
	if err := s.users.MarkEmailVerified(ctx, userID, time.Now()); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		UserID: userID, Action: "auth.email_verified", EntityType: "user", EntityID: userID,
	})
	return nil
}

// ResendVerification issues a fresh verification email.
func (s *AuthService) ResendVerification(ctx context.Context, userID string) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.EmailVerifiedAt != nil {
		return apperror.Validation("email is already verified")
	}
	s.sendVerificationEmail(ctx, user)
	return nil
}

// SwitchTenant issues a new access token scoped to another tenant the
// user belongs to. The refresh session is unchanged.
func (s *AuthService) SwitchTenant(ctx context.Context, userID, sessionID, tenantID string) (string, *tenant.Membership, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "", nil, err
	}
	membership, err := s.memberships.Get(ctx, tenantID, userID)
	if err != nil {
		return "", nil, apperror.Forbidden("you do not belong to this business")
	}
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return "", nil, err
	}
	if t.Status != "active" {
		return "", nil, apperror.Forbidden("this business is suspended")
	}
	membership.TenantName = t.Name
	membership.TenantSlug = t.Slug

	access, err := s.tokens.NewAccessToken(user.ID, membership.TenantID, membership.Role, sessionID, user.IsSuperAdmin)
	if err != nil {
		return "", nil, apperror.Internal(err)
	}
	return access, membership, nil
}

// establishSession creates a device session and returns tokens.
func (s *AuthService) establishSession(ctx context.Context, user *auth.User, meta RequestMeta) (*AuthResult, error) {
	raw, hash, err := token.NewRefreshToken()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	sess := &auth.DeviceSession{
		UserID:           user.ID,
		RefreshTokenHash: hash,
		DeviceName:       meta.DeviceName,
		UserAgent:        meta.UserAgent,
		IP:               meta.IP,
		ExpiresAt:        time.Now().Add(s.tokens.RefreshTTL()),
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return nil, err
	}
	return s.buildAuthResult(ctx, user, sess.ID, raw)
}

// buildAuthResult picks the default tenant and mints the access token.
func (s *AuthService) buildAuthResult(ctx context.Context, user *auth.User, sessionID, rawRefresh string) (*AuthResult, error) {
	memberships, err := s.memberships.ListByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	var active *tenant.Membership
	if len(memberships) > 0 {
		active = &memberships[0]
	}

	tenantID, role := "", ""
	if active != nil {
		tenantID, role = active.TenantID, active.Role
	}
	access, err := s.tokens.NewAccessToken(user.ID, tenantID, role, sessionID, user.IsSuperAdmin)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &AuthResult{
		User:         user,
		Memberships:  memberships,
		ActiveTenant: active,
		AccessToken:  access,
		RefreshToken: rawRefresh,
	}, nil
}

func (s *AuthService) sendVerificationEmail(ctx context.Context, user *auth.User) {
	verifyToken, err := s.otp.Issue(ctx, purposeVerifyEmail, user.ID, verifyEmailTTL)
	if err != nil {
		s.logger.Error("failed to issue verification token", "error", err)
		return
	}
	link := fmt.Sprintf("%s/verify-email?token=%s", s.appBaseURL, verifyToken)
	body := mailer.Render(mailer.Email{
		AppName:    s.appName,
		Title:      "Verify your email address",
		Intro:      fmt.Sprintf("Hi %s, welcome! Confirm this email address to finish setting up your account.", user.FullName),
		ButtonText: "Verify email",
		ButtonURL:  link,
		FooterNote: "If you didn't create this account, you can safely ignore this email.",
	})
	if err := s.mailer.Send(user.Email, "Verify your email", body); err != nil {
		s.logger.Error("failed to send verification email", "error", err)
	}
}

// slugify normalizes a business slug: lowercase, alphanumerics and dashes.
func slugify(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	var b strings.Builder
	lastDash := false
	for _, r := range in {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case !lastDash && b.Len() > 0:
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
