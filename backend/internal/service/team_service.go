package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
)

// Invited members set their own password via an emailed link; the token
// reuses the password-reset purpose so the existing /auth/reset-password
// endpoint completes the flow.
const inviteSetPasswordTTL = 7 * 24 * time.Hour

// TeamService owns staff account management within a tenant and the
// super-admin "create business + owner" flow. Both share the same
// invite mechanics: create (or reuse) the user, then email a
// set-password link.
type TeamService struct {
	users       auth.UserRepository
	memberships tenant.MembershipRepository
	tenants     tenant.Repository
	settings    tenant.SettingsRepository
	otp         TokenStore
	mailer      EmailSender
	auditor     *AuditService
	logger      *slog.Logger
	appBaseURL  string
	appName     string
}

type TeamServiceDeps struct {
	Users       auth.UserRepository
	Memberships tenant.MembershipRepository
	Tenants     tenant.Repository
	Settings    tenant.SettingsRepository
	OTP         TokenStore
	Mailer      EmailSender
	Auditor     *AuditService
	Logger      *slog.Logger
	AppBaseURL  string
	AppName     string
}

func NewTeamService(d TeamServiceDeps) *TeamService {
	return &TeamService{
		users: d.Users, memberships: d.Memberships, tenants: d.Tenants, settings: d.Settings,
		otp: d.OTP, mailer: d.Mailer, auditor: d.Auditor, logger: d.Logger,
		appBaseURL: d.AppBaseURL, appName: d.AppName,
	}
}

// validTeamRole reports whether a role can be granted through team
// management. Ownership is never assignable — it belongs to the account
// that registered (or was assigned) the business.
func validTeamRole(role string) bool {
	switch rbac.Role(role) {
	case rbac.RoleManager, rbac.RoleCashier, rbac.RoleKitchen, rbac.RoleEmployee:
		return true
	}
	return false
}

func (s *TeamService) ListMembers(ctx context.Context, tenantID string) ([]tenant.Member, error) {
	return s.memberships.ListMembers(ctx, tenantID)
}

// InviteResult tells the caller whether a brand-new account was created
// (set-password email) or an existing account was attached.
type InviteResult struct {
	Member      *tenant.Member `json:"member"`
	UserCreated bool           `json:"user_created"`
}

// InviteMember creates a staff account under the tenant. New emails get
// an account with an unguessable password plus a set-password link;
// existing accounts are attached with the given role and notified.
func (s *TeamService) InviteMember(ctx context.Context, tenantID, actorID, fullName, email, role string) (*InviteResult, error) {
	if !validTeamRole(role) {
		return nil, apperror.Validation("role must be manager, cashier, kitchen, or employee")
	}

	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	user, created, err := s.ensureUser(ctx, fullName, email)
	if err != nil {
		return nil, err
	}

	membership := &tenant.Membership{TenantID: tenantID, UserID: user.ID, Role: role}
	if err := s.memberships.Create(ctx, membership); err != nil {
		return nil, err
	}

	if created {
		s.sendSetPasswordEmail(ctx, user, t.Name, role)
	} else {
		s.sendAddedEmail(user, t.Name, role)
	}

	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: actorID, Action: "team.member_invited",
		EntityType: "user", EntityID: user.ID,
		After: map[string]any{"email": user.Email, "role": role, "user_created": created},
	})

	return &InviteResult{
		Member: &tenant.Member{
			Membership: *membership,
			FullName:   user.FullName,
			Email:      user.Email,
			UserStatus: user.Status,
			JoinedAt:   time.Now(),
		},
		UserCreated: created,
	}, nil
}

// UpdateMemberRole changes a member's role. The business owner's role
// and the caller's own role are immutable here.
func (s *TeamService) UpdateMemberRole(ctx context.Context, tenantID, actorID, userID, role string) error {
	if !validTeamRole(role) {
		return apperror.Validation("role must be manager, cashier, kitchen, or employee")
	}
	if userID == actorID {
		return apperror.Validation("you cannot change your own role")
	}
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if t.OwnerUserID == userID {
		return apperror.Validation("the business owner's role cannot be changed")
	}
	membership, err := s.memberships.Get(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if err := s.memberships.UpdateRole(ctx, tenantID, userID, role); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: actorID, Action: "team.role_changed",
		EntityType: "user", EntityID: userID,
		Before: map[string]any{"role": membership.Role}, After: map[string]any{"role": role},
	})
	return nil
}

// RemoveMember detaches a member from the tenant. The owner cannot be
// removed and callers cannot remove themselves. The user account itself
// is untouched — it may belong to other tenants.
func (s *TeamService) RemoveMember(ctx context.Context, tenantID, actorID, userID string) error {
	if userID == actorID {
		return apperror.Validation("you cannot remove yourself from the team")
	}
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if t.OwnerUserID == userID {
		return apperror.Validation("the business owner cannot be removed")
	}
	membership, err := s.memberships.Get(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	if err := s.memberships.Delete(ctx, tenantID, userID); err != nil {
		return err
	}
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: actorID, Action: "team.member_removed",
		EntityType: "user", EntityID: userID,
		Before: map[string]any{"role": membership.Role},
	})
	return nil
}

// ResendInvite emails a fresh set-password link to a member — for staff
// who lost the original invite.
func (s *TeamService) ResendInvite(ctx context.Context, tenantID, actorID, userID string) error {
	membership, err := s.memberships.Get(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	s.sendSetPasswordEmail(ctx, user, t.Name, membership.Role)
	s.auditor.Record(audit.Log{
		TenantID: tenantID, UserID: actorID, Action: "team.invite_resent",
		EntityType: "user", EntityID: userID,
	})
	return nil
}

// AdminCreateBusiness provisions a tenant with default settings and an
// owner account from the super-admin console. A brand-new owner gets a
// set-password invite; an existing account is attached as owner.
func (s *TeamService) AdminCreateBusiness(ctx context.Context, actorID, businessName, businessSlug, ownerName, ownerEmail string) (*tenant.Tenant, error) {
	user, created, err := s.ensureUser(ctx, ownerName, ownerEmail)
	if err != nil {
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
	if t.Slug == "" {
		return nil, apperror.Validation("business slug must contain letters or numbers")
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

	if created {
		s.sendSetPasswordEmail(ctx, user, t.Name, string(rbac.RoleOwner))
	} else {
		s.sendAddedEmail(user, t.Name, string(rbac.RoleOwner))
	}

	s.auditor.Record(audit.Log{
		TenantID: t.ID, UserID: actorID, Action: "admin.tenant_created",
		EntityType: "tenant", EntityID: t.ID,
		After: map[string]any{"name": t.Name, "slug": t.Slug, "owner_email": user.Email, "owner_created": created},
	})
	return t, nil
}

// ensureUser finds an account by email or creates one with an
// unguessable placeholder password (replaced via the set-password link).
func (s *TeamService) ensureUser(ctx context.Context, fullName, email string) (*auth.User, bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if existing, err := s.users.GetByEmail(ctx, email); err == nil {
		return existing, false, nil
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, false, apperror.Internal(err)
	}
	hash, err := password.Hash(hex.EncodeToString(randomBytes))
	if err != nil {
		return nil, false, apperror.Internal(err)
	}

	user := &auth.User{
		Email:        email,
		PasswordHash: hash,
		FullName:     strings.TrimSpace(fullName),
		Status:       "active",
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, false, err
	}
	return user, true, nil
}

func (s *TeamService) sendSetPasswordEmail(ctx context.Context, user *auth.User, tenantName, role string) {
	inviteToken, err := s.otp.Issue(ctx, purposePasswordReset, user.ID, inviteSetPasswordTTL)
	if err != nil {
		s.logger.Error("failed to issue invite token", "error", err)
		return
	}
	link := fmt.Sprintf("%s/reset-password?token=%s&welcome=1", s.appBaseURL, inviteToken)
	body := mailer.Render(mailer.Email{
		AppName:    s.appName,
		Title:      fmt.Sprintf("You've been invited to %s", tenantName),
		Intro:      fmt.Sprintf("Hi %s, an account was created for you at %s on %s with the %s role. Set your password to start — the link is valid for 7 days.", user.FullName, tenantName, s.appName, role),
		ButtonText: "Set your password",
		ButtonURL:  link,
		FooterNote: "If you weren't expecting this invitation, you can safely ignore this email.",
	})
	if err := s.mailer.Send(user.Email, fmt.Sprintf("You've been invited to %s", tenantName), body); err != nil {
		s.logger.Error("failed to send invite email", "error", err)
	}
}

func (s *TeamService) sendAddedEmail(user *auth.User, tenantName, role string) {
	body := mailer.Render(mailer.Email{
		AppName:    s.appName,
		Title:      fmt.Sprintf("You've been added to %s", tenantName),
		Intro:      fmt.Sprintf("Hi %s, your existing %s account now has access to %s with the %s role. Sign in and switch businesses to get started.", user.FullName, s.appName, tenantName, role),
		ButtonText: "Sign in",
		ButtonURL:  fmt.Sprintf("%s/login", s.appBaseURL),
		FooterNote: "If you weren't expecting this, contact the business owner.",
	})
	if err := s.mailer.Send(user.Email, fmt.Sprintf("You've been added to %s", tenantName), body); err != nil {
		s.logger.Error("failed to send member-added email", "error", err)
	}
}
