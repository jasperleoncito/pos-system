package service

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jasperleoncito/pos-system/backend/internal/domain/audit"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/auth"
	"github.com/jasperleoncito/pos-system/backend/internal/domain/tenant"
	"github.com/jasperleoncito/pos-system/backend/internal/pkg/apperror"
)

// ---- in-memory fakes ----

type fakeUserRepo struct {
	users  map[string]*auth.User // by id
	nextID int
}

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{users: map[string]*auth.User{}} }

func (r *fakeUserRepo) Create(_ context.Context, u *auth.User) error {
	for _, existing := range r.users {
		if strings.EqualFold(existing.Email, u.Email) {
			return apperror.Conflict("email already registered")
		}
	}
	r.nextID++
	u.ID = "user-" + strconv.Itoa(r.nextID)
	r.users[u.ID] = u
	return nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id string) (*auth.User, error) {
	if u, ok := r.users[id]; ok {
		return u, nil
	}
	return nil, apperror.NotFound("user")
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*auth.User, error) {
	for _, u := range r.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, apperror.NotFound("user")
}

func (r *fakeUserRepo) Update(_ context.Context, _ *auth.User) error { return nil }
func (r *fakeUserRepo) MarkEmailVerified(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (r *fakeUserRepo) UpdatePassword(_ context.Context, _, _ string) error { return nil }

type fakeMembershipRepo struct {
	memberships []*tenant.Membership
	nextID      int
}

func (r *fakeMembershipRepo) Create(_ context.Context, m *tenant.Membership) error {
	for _, existing := range r.memberships {
		if existing.TenantID == m.TenantID && existing.UserID == m.UserID {
			return apperror.Conflict("user already belongs to this business")
		}
	}
	r.nextID++
	m.ID = "membership-" + strconv.Itoa(r.nextID)
	r.memberships = append(r.memberships, m)
	return nil
}

func (r *fakeMembershipRepo) Get(_ context.Context, tenantID, userID string) (*tenant.Membership, error) {
	for _, m := range r.memberships {
		if m.TenantID == tenantID && m.UserID == userID {
			return m, nil
		}
	}
	return nil, apperror.NotFound("membership")
}

func (r *fakeMembershipRepo) ListByUser(_ context.Context, _ string) ([]tenant.Membership, error) {
	return nil, nil
}
func (r *fakeMembershipRepo) ListByTenant(_ context.Context, _ string) ([]tenant.Membership, error) {
	return nil, nil
}
func (r *fakeMembershipRepo) ListMembers(_ context.Context, _ string) ([]tenant.Member, error) {
	return nil, nil
}

func (r *fakeMembershipRepo) UpdateRole(_ context.Context, tenantID, userID, role string) error {
	for _, m := range r.memberships {
		if m.TenantID == tenantID && m.UserID == userID {
			m.Role = role
			return nil
		}
	}
	return apperror.NotFound("membership")
}

func (r *fakeMembershipRepo) Delete(_ context.Context, tenantID, userID string) error {
	for i, m := range r.memberships {
		if m.TenantID == tenantID && m.UserID == userID {
			r.memberships = append(r.memberships[:i], r.memberships[i+1:]...)
			return nil
		}
	}
	return apperror.NotFound("membership")
}

type fakeTenantRepo struct {
	tenants map[string]*tenant.Tenant
	nextID  int
}

func newFakeTenantRepo() *fakeTenantRepo { return &fakeTenantRepo{tenants: map[string]*tenant.Tenant{}} }

func (r *fakeTenantRepo) Create(_ context.Context, t *tenant.Tenant) error {
	for _, existing := range r.tenants {
		if existing.Slug == t.Slug {
			return apperror.Conflict("slug already taken")
		}
	}
	r.nextID++
	t.ID = "tenant-" + strconv.Itoa(r.nextID)
	r.tenants[t.ID] = t
	return nil
}

func (r *fakeTenantRepo) GetByID(_ context.Context, id string) (*tenant.Tenant, error) {
	if t, ok := r.tenants[id]; ok {
		return t, nil
	}
	return nil, apperror.NotFound("tenant")
}

func (r *fakeTenantRepo) GetBySlug(_ context.Context, _ string) (*tenant.Tenant, error) {
	return nil, apperror.NotFound("tenant")
}
func (r *fakeTenantRepo) Update(_ context.Context, _ *tenant.Tenant) error { return nil }
func (r *fakeTenantRepo) List(_ context.Context, _, _ int) ([]tenant.Tenant, int64, error) {
	return nil, 0, nil
}
func (r *fakeTenantRepo) SetPlan(_ context.Context, _, _ string) error { return nil }
func (r *fakeTenantRepo) PlatformStats(_ context.Context) (map[string]any, error) {
	return nil, nil
}
func (r *fakeTenantRepo) PlatformSales(_ context.Context, _ int) (*tenant.PlatformSales, error) {
	return &tenant.PlatformSales{}, nil
}

type fakeSettingsRepo struct{ created int }

func (r *fakeSettingsRepo) Create(_ context.Context, _ *tenant.Settings) error {
	r.created++
	return nil
}
func (r *fakeSettingsRepo) GetByTenant(_ context.Context, _ string) (*tenant.Settings, error) {
	return nil, apperror.NotFound("settings")
}
func (r *fakeSettingsRepo) Update(_ context.Context, _ *tenant.Settings) error { return nil }

type fakeTokenStore struct{ issued []string }

func (s *fakeTokenStore) Issue(_ context.Context, purpose, userID string, _ time.Duration) (string, error) {
	s.issued = append(s.issued, purpose+":"+userID)
	return "token-" + strconv.Itoa(len(s.issued)), nil
}
func (s *fakeTokenStore) Consume(_ context.Context, _, _ string) (string, error) { return "", nil }

type sentMail struct{ to, subject string }

type fakeMailer struct{ sent []sentMail }

func (m *fakeMailer) Send(to, subject, _ string) error {
	m.sent = append(m.sent, sentMail{to: to, subject: subject})
	return nil
}

type noopAuditRepo struct{}

func (noopAuditRepo) Insert(_ context.Context, _ *audit.Log) error { return nil }
func (noopAuditRepo) List(_ context.Context, _ string, _, _ int) ([]audit.Log, int64, error) {
	return nil, 0, nil
}

// ---- harness ----

type teamFixture struct {
	svc         *TeamService
	users       *fakeUserRepo
	memberships *fakeMembershipRepo
	tenants     *fakeTenantRepo
	settings    *fakeSettingsRepo
	otp         *fakeTokenStore
	mail        *fakeMailer
}

func newTeamFixture(t *testing.T) *teamFixture {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	f := &teamFixture{
		users:       newFakeUserRepo(),
		memberships: &fakeMembershipRepo{},
		tenants:     newFakeTenantRepo(),
		settings:    &fakeSettingsRepo{},
		otp:         &fakeTokenStore{},
		mail:        &fakeMailer{},
	}
	f.svc = NewTeamService(TeamServiceDeps{
		Users: f.users, Memberships: f.memberships, Tenants: f.tenants, Settings: f.settings,
		OTP: f.otp, Mailer: f.mail, Auditor: NewAuditService(noopAuditRepo{}, logger),
		Logger: logger, AppBaseURL: "http://localhost:7642", AppName: "POS System",
	})
	return f
}

// seedTenant creates a tenant with an owner user + membership.
func (f *teamFixture) seedTenant(t *testing.T) (*tenant.Tenant, *auth.User) {
	t.Helper()
	owner := &auth.User{Email: "owner@test.ph", FullName: "Owner", Status: "active"}
	if err := f.users.Create(context.Background(), owner); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	biz := &tenant.Tenant{Name: "Test Eatery", Slug: "test-eatery", OwnerUserID: owner.ID, Status: "active"}
	if err := f.tenants.Create(context.Background(), biz); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	if err := f.memberships.Create(context.Background(), &tenant.Membership{
		TenantID: biz.ID, UserID: owner.ID, Role: "owner",
	}); err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	return biz, owner
}

// ---- tests ----

func TestInviteMemberCreatesUserAndSendsSetPasswordEmail(t *testing.T) {
	f := newTeamFixture(t)
	biz, owner := f.seedTenant(t)

	result, err := f.svc.InviteMember(context.Background(), biz.ID, owner.ID,
		"New Cashier", "Cashier@Test.PH", "cashier")
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}
	if !result.UserCreated {
		t.Error("expected a new user to be created")
	}
	if result.Member.Email != "cashier@test.ph" {
		t.Errorf("email = %q, want normalized lowercase", result.Member.Email)
	}
	if len(f.otp.issued) != 1 || !strings.HasPrefix(f.otp.issued[0], "password_reset:") {
		t.Errorf("expected one password_reset token, got %v", f.otp.issued)
	}
	if len(f.mail.sent) != 1 || !strings.Contains(f.mail.sent[0].subject, "invited") {
		t.Errorf("expected one invite email, got %+v", f.mail.sent)
	}
	if _, err := f.memberships.Get(context.Background(), biz.ID, result.Member.UserID); err != nil {
		t.Errorf("membership not created: %v", err)
	}
}

func TestInviteMemberAttachesExistingUserWithoutToken(t *testing.T) {
	f := newTeamFixture(t)
	biz, owner := f.seedTenant(t)
	existing := &auth.User{Email: "kitchen@test.ph", FullName: "Existing Kitchen", Status: "active"}
	if err := f.users.Create(context.Background(), existing); err != nil {
		t.Fatalf("seed existing user: %v", err)
	}

	result, err := f.svc.InviteMember(context.Background(), biz.ID, owner.ID,
		"Ignored Name", "kitchen@test.ph", "kitchen")
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}
	if result.UserCreated {
		t.Error("expected the existing user to be reused")
	}
	if result.Member.UserID != existing.ID {
		t.Errorf("user id = %s, want %s", result.Member.UserID, existing.ID)
	}
	if len(f.otp.issued) != 0 {
		t.Errorf("no set-password token should be issued for existing users, got %v", f.otp.issued)
	}
	if len(f.mail.sent) != 1 || !strings.Contains(f.mail.sent[0].subject, "added") {
		t.Errorf("expected one member-added email, got %+v", f.mail.sent)
	}
}

func TestInviteMemberRejectsInvalidRolesAndDuplicates(t *testing.T) {
	f := newTeamFixture(t)
	biz, owner := f.seedTenant(t)

	for _, role := range []string{"owner", "boss", ""} {
		if _, err := f.svc.InviteMember(context.Background(), biz.ID, owner.ID, "X Y", "x@test.ph", role); err == nil {
			t.Errorf("role %q should be rejected", role)
		}
	}

	if _, err := f.svc.InviteMember(context.Background(), biz.ID, owner.ID, "A B", "a@test.ph", "cashier"); err != nil {
		t.Fatalf("first invite: %v", err)
	}
	if _, err := f.svc.InviteMember(context.Background(), biz.ID, owner.ID, "A B", "a@test.ph", "kitchen"); err == nil {
		t.Error("second invite of the same email should conflict on membership")
	}
}

func TestUpdateMemberRoleProtections(t *testing.T) {
	f := newTeamFixture(t)
	biz, owner := f.seedTenant(t)
	result, err := f.svc.InviteMember(context.Background(), biz.ID, owner.ID, "Staff", "staff@test.ph", "employee")
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	staffID := result.Member.UserID

	if err := f.svc.UpdateMemberRole(context.Background(), biz.ID, owner.ID, owner.ID, "manager"); err == nil {
		t.Error("changing your own role should be rejected")
	}
	if err := f.svc.UpdateMemberRole(context.Background(), biz.ID, staffID, owner.ID, "manager"); err == nil {
		t.Error("changing the owner's role should be rejected")
	}
	if err := f.svc.UpdateMemberRole(context.Background(), biz.ID, owner.ID, staffID, "owner"); err == nil {
		t.Error("granting the owner role should be rejected")
	}

	if err := f.svc.UpdateMemberRole(context.Background(), biz.ID, owner.ID, staffID, "manager"); err != nil {
		t.Fatalf("valid role change: %v", err)
	}
	m, _ := f.memberships.Get(context.Background(), biz.ID, staffID)
	if m.Role != "manager" {
		t.Errorf("role = %s, want manager", m.Role)
	}
}

func TestRemoveMemberProtections(t *testing.T) {
	f := newTeamFixture(t)
	biz, owner := f.seedTenant(t)
	result, err := f.svc.InviteMember(context.Background(), biz.ID, owner.ID, "Staff", "staff@test.ph", "employee")
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	staffID := result.Member.UserID

	if err := f.svc.RemoveMember(context.Background(), biz.ID, owner.ID, owner.ID); err == nil {
		t.Error("removing yourself should be rejected")
	}
	if err := f.svc.RemoveMember(context.Background(), biz.ID, staffID, owner.ID); err == nil {
		t.Error("removing the owner should be rejected")
	}
	if err := f.svc.RemoveMember(context.Background(), biz.ID, owner.ID, staffID); err != nil {
		t.Fatalf("valid removal: %v", err)
	}
	if _, err := f.memberships.Get(context.Background(), biz.ID, staffID); err == nil {
		t.Error("membership should be gone after removal")
	}
	if _, err := f.users.GetByID(context.Background(), staffID); err != nil {
		t.Error("the user account itself must survive removal")
	}
}

func TestResendInviteIssuesFreshToken(t *testing.T) {
	f := newTeamFixture(t)
	biz, owner := f.seedTenant(t)
	result, err := f.svc.InviteMember(context.Background(), biz.ID, owner.ID, "Staff", "staff@test.ph", "employee")
	if err != nil {
		t.Fatalf("invite: %v", err)
	}

	if err := f.svc.ResendInvite(context.Background(), biz.ID, owner.ID, result.Member.UserID); err != nil {
		t.Fatalf("ResendInvite: %v", err)
	}
	if len(f.otp.issued) != 2 {
		t.Errorf("expected 2 issued tokens (invite + resend), got %d", len(f.otp.issued))
	}
	if len(f.mail.sent) != 2 {
		t.Errorf("expected 2 emails, got %d", len(f.mail.sent))
	}
}

func TestAdminCreateBusinessWithNewOwner(t *testing.T) {
	f := newTeamFixture(t)

	biz, err := f.svc.AdminCreateBusiness(context.Background(), "super-admin",
		"Bella's Bistro", "Bella's Bistro!!", "Bella Cruz", "bella@test.ph")
	if err != nil {
		t.Fatalf("AdminCreateBusiness: %v", err)
	}
	if biz.Slug != "bella-s-bistro" {
		t.Errorf("slug = %q, want slugified %q", biz.Slug, "bella-s-bistro")
	}
	owner, err := f.users.GetByEmail(context.Background(), "bella@test.ph")
	if err != nil {
		t.Fatalf("owner user not created: %v", err)
	}
	if biz.OwnerUserID != owner.ID {
		t.Errorf("owner_user_id = %s, want %s", biz.OwnerUserID, owner.ID)
	}
	m, err := f.memberships.Get(context.Background(), biz.ID, owner.ID)
	if err != nil {
		t.Fatalf("owner membership missing: %v", err)
	}
	if m.Role != "owner" {
		t.Errorf("membership role = %s, want owner", m.Role)
	}
	if f.settings.created != 1 {
		t.Errorf("settings created = %d, want 1", f.settings.created)
	}
	if len(f.otp.issued) != 1 {
		t.Errorf("expected a set-password token for the new owner, got %v", f.otp.issued)
	}
	if len(f.mail.sent) != 1 || !strings.Contains(f.mail.sent[0].subject, "invited") {
		t.Errorf("expected an invite email, got %+v", f.mail.sent)
	}
}

func TestAdminCreateBusinessReusesExistingOwnerAndRejectsDuplicateSlug(t *testing.T) {
	f := newTeamFixture(t)
	existing := &auth.User{Email: "veteran@test.ph", FullName: "Veteran Owner", Status: "active"}
	if err := f.users.Create(context.Background(), existing); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	biz, err := f.svc.AdminCreateBusiness(context.Background(), "super-admin",
		"Second Branch", "second-branch", "Ignored", "veteran@test.ph")
	if err != nil {
		t.Fatalf("AdminCreateBusiness: %v", err)
	}
	if biz.OwnerUserID != existing.ID {
		t.Errorf("owner should be the existing account")
	}
	if len(f.otp.issued) != 0 {
		t.Errorf("no set-password token for existing owners, got %v", f.otp.issued)
	}
	if len(f.mail.sent) != 1 || !strings.Contains(f.mail.sent[0].subject, "added") {
		t.Errorf("expected a member-added email, got %+v", f.mail.sent)
	}

	if _, err := f.svc.AdminCreateBusiness(context.Background(), "super-admin",
		"Clone", "second-branch", "Someone", "someone@test.ph"); err == nil {
		t.Error("duplicate slug should be rejected")
	}
}

func TestValidTeamRole(t *testing.T) {
	for _, valid := range []string{"manager", "cashier", "kitchen", "employee"} {
		if !validTeamRole(valid) {
			t.Errorf("expected %q to be valid", valid)
		}
	}
	for _, invalid := range []string{"owner", "", "MANAGER", "admin"} {
		if validTeamRole(invalid) {
			t.Errorf("expected %q to be invalid", invalid)
		}
	}
}
