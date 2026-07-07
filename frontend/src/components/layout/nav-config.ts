import {
  BadgePercent,
  BarChart3,
  CalendarClock,
  ChefHat,
  LayoutDashboard,
  Package,
  Settings,
  ShoppingCart,
  UserRound,
  UsersRound,
  UtensilsCrossed,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

import type { Permission } from "@/lib/rbac";

export interface NavItem {
  label: string;
  segment: string; // path under /[tenant]/
  icon: LucideIcon;
  permission: Permission;
}

export const NAV_ITEMS: NavItem[] = [
  { label: "Dashboard", segment: "dashboard", icon: LayoutDashboard, permission: "analytics:read" },
  { label: "POS", segment: "pos", icon: ShoppingCart, permission: "orders:create" },
  { label: "Kitchen", segment: "kitchen", icon: ChefHat, permission: "kitchen:read" },
  { label: "Orders", segment: "orders", icon: UtensilsCrossed, permission: "orders:read" },
  { label: "Menu", segment: "menu", icon: UtensilsCrossed, permission: "catalog:read" },
  { label: "Promos", segment: "promos", icon: BadgePercent, permission: "catalog:write" },
  { label: "Inventory", segment: "inventory", icon: Package, permission: "inventory:read" },
  { label: "Customers", segment: "customers", icon: UserRound, permission: "customers:read" },
  { label: "Employees", segment: "employees", icon: UsersRound, permission: "employees:read" },
  { label: "Attendance", segment: "attendance", icon: CalendarClock, permission: "attendance:clock" },
  { label: "Reports", segment: "reports", icon: BarChart3, permission: "reports:read" },
  { label: "Settings", segment: "settings", icon: Settings, permission: "tenant_settings:read" },
];
