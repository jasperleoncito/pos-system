package rbac

import "testing"

func TestCan(t *testing.T) {
	tests := []struct {
		name string
		role Role
		perm Permission
		want bool
	}{
		{"owner can write settings", RoleOwner, PermTenantSettingsWrite, true},
		{"owner can read audit", RoleOwner, PermAuditRead, true},
		{"manager can approve attendance", RoleManager, PermAttendanceApprove, true},
		{"manager can void orders", RoleManager, PermOrdersVoid, true},
		{"manager cannot write settings", RoleManager, PermTenantSettingsWrite, false},
		{"manager cannot manage users", RoleManager, PermUsersManage, false},
		{"cashier can create orders", RoleCashier, PermOrdersCreate, true},
		{"cashier cannot void orders", RoleCashier, PermOrdersVoid, false},
		{"cashier cannot refund", RoleCashier, PermOrdersRefund, false},
		{"cashier cannot write settings", RoleCashier, PermTenantSettingsWrite, false},
		{"kitchen can write kitchen", RoleKitchen, PermKitchenWrite, true},
		{"kitchen cannot create orders", RoleKitchen, PermOrdersCreate, false},
		{"employee can clock", RoleEmployee, PermAttendanceClock, true},
		{"employee cannot read reports", RoleEmployee, PermReportsRead, false},
		{"unknown role has nothing", Role("ghost"), PermOrdersCreate, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Can(tt.role, tt.perm); got != tt.want {
				t.Errorf("Can(%s, %s) = %v, want %v", tt.role, tt.perm, got, tt.want)
			}
		})
	}
}

func TestValidRole(t *testing.T) {
	for _, r := range []string{"owner", "manager", "cashier", "kitchen", "employee"} {
		if !ValidRole(r) {
			t.Errorf("ValidRole(%q) = false, want true", r)
		}
	}
	for _, r := range []string{"super_admin", "admin", "", "OWNER"} {
		if ValidRole(r) {
			t.Errorf("ValidRole(%q) = true, want false", r)
		}
	}
}

func TestOwnerHasEveryPermission(t *testing.T) {
	for _, p := range allPermissions() {
		if !Can(RoleOwner, p) {
			t.Errorf("owner missing permission %s", p)
		}
	}
}
