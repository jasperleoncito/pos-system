// Package rbac defines the static role → permission matrix. Roles are
// fixed by the product spec; permissions are strings checked per route.
package rbac

import "slices"

// Role is a tenant-scoped user role.
type Role string

const (
	RoleOwner    Role = "owner"
	RoleManager  Role = "manager"
	RoleCashier  Role = "cashier"
	RoleKitchen  Role = "kitchen"
	RoleEmployee Role = "employee"
)

// Permission names follow "resource:action".
type Permission string

const (
	PermTenantSettingsRead  Permission = "tenant_settings:read"
	PermTenantSettingsWrite Permission = "tenant_settings:write"
	PermUsersManage         Permission = "users:manage"
	PermSessionsManage      Permission = "sessions:manage"
	PermBillingManage       Permission = "billing:manage"

	PermCatalogRead  Permission = "catalog:read"
	PermCatalogWrite Permission = "catalog:write"

	PermOrdersCreate Permission = "orders:create"
	PermOrdersRead   Permission = "orders:read"
	PermOrdersRefund Permission = "orders:refund"
	PermOrdersVoid   Permission = "orders:void"

	PermKitchenRead  Permission = "kitchen:read"
	PermKitchenWrite Permission = "kitchen:write"

	PermInventoryRead  Permission = "inventory:read"
	PermInventoryWrite Permission = "inventory:write"

	PermEmployeesRead  Permission = "employees:read"
	PermEmployeesWrite Permission = "employees:write"

	PermAttendanceClock   Permission = "attendance:clock"
	PermAttendanceRead    Permission = "attendance:read"
	PermAttendanceApprove Permission = "attendance:approve"

	PermCustomersRead  Permission = "customers:read"
	PermCustomersWrite Permission = "customers:write"

	PermReportsRead   Permission = "reports:read"
	PermAnalyticsRead Permission = "analytics:read"
	PermAuditRead     Permission = "audit:read"
)

// matrix maps each role to its allowed permissions. Owner gets everything.
var matrix = map[Role][]Permission{
	RoleOwner: allPermissions(),
	RoleManager: {
		PermTenantSettingsRead,
		PermCatalogRead, PermCatalogWrite,
		PermOrdersCreate, PermOrdersRead, PermOrdersRefund, PermOrdersVoid,
		PermKitchenRead, PermKitchenWrite,
		PermInventoryRead, PermInventoryWrite,
		PermEmployeesRead, PermEmployeesWrite,
		PermAttendanceClock, PermAttendanceRead, PermAttendanceApprove,
		PermCustomersRead, PermCustomersWrite,
		PermReportsRead, PermAnalyticsRead,
	},
	RoleCashier: {
		PermCatalogRead,
		PermOrdersCreate, PermOrdersRead,
		PermCustomersRead, PermCustomersWrite,
		PermAttendanceClock,
	},
	RoleKitchen: {
		PermKitchenRead, PermKitchenWrite,
		PermAttendanceClock,
	},
	RoleEmployee: {
		PermAttendanceClock,
	},
}

func allPermissions() []Permission {
	return []Permission{
		PermTenantSettingsRead, PermTenantSettingsWrite,
		PermUsersManage, PermSessionsManage, PermBillingManage,
		PermCatalogRead, PermCatalogWrite,
		PermOrdersCreate, PermOrdersRead, PermOrdersRefund, PermOrdersVoid,
		PermKitchenRead, PermKitchenWrite,
		PermInventoryRead, PermInventoryWrite,
		PermEmployeesRead, PermEmployeesWrite,
		PermAttendanceClock, PermAttendanceRead, PermAttendanceApprove,
		PermCustomersRead, PermCustomersWrite,
		PermReportsRead, PermAnalyticsRead, PermAuditRead,
	}
}

// ValidRole reports whether the string is a known tenant role.
func ValidRole(r string) bool {
	_, ok := matrix[Role(r)]
	return ok
}

// Can reports whether the role grants the permission.
func Can(role Role, perm Permission) bool {
	return slices.Contains(matrix[role], perm)
}
