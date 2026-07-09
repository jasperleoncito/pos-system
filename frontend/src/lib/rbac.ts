import type { Role } from "@/types/auth";

/**
 * Frontend mirror of the backend permission matrix — used only to shape
 * navigation and hide actions. The API is the real enforcement point.
 */
export type Permission =
  | "tenant_settings:read"
  | "tenant_settings:write"
  | "users:manage"
  | "billing:manage"
  | "catalog:read"
  | "catalog:write"
  | "orders:create"
  | "orders:read"
  | "orders:refund"
  | "orders:void"
  | "kitchen:read"
  | "inventory:read"
  | "inventory:write"
  | "employees:read"
  | "employees:write"
  | "attendance:clock"
  | "attendance:read"
  | "attendance:approve"
  | "customers:read"
  | "customers:write"
  | "reports:read"
  | "analytics:read"
  | "audit:read";

const ALL: Permission[] = [
  "tenant_settings:read", "tenant_settings:write", "users:manage", "billing:manage",
  "catalog:read", "catalog:write",
  "orders:create", "orders:read", "orders:refund", "orders:void",
  "kitchen:read",
  "inventory:read", "inventory:write",
  "employees:read", "employees:write",
  "attendance:clock", "attendance:read", "attendance:approve",
  "customers:read", "customers:write",
  "reports:read", "analytics:read", "audit:read",
];

const MATRIX: Record<Role, Permission[]> = {
  owner: ALL,
  manager: [
    "tenant_settings:read",
    "catalog:read", "catalog:write",
    "orders:create", "orders:read", "orders:refund", "orders:void",
    "kitchen:read",
    "inventory:read", "inventory:write",
    "employees:read", "employees:write",
    "attendance:clock", "attendance:read", "attendance:approve",
    "customers:read", "customers:write",
    "reports:read", "analytics:read",
  ],
  cashier: [
    "catalog:read",
    "orders:create", "orders:read",
    "customers:read", "customers:write",
    "attendance:clock",
  ],
  kitchen: ["kitchen:read", "attendance:clock"],
  employee: ["attendance:clock"],
};

export function can(role: Role | undefined, permission: Permission): boolean {
  if (!role) return false;
  return MATRIX[role]?.includes(permission) ?? false;
}
