"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { cn } from "@/lib/utils";
import { can } from "@/lib/rbac";
import type { AuthState } from "@/lib/auth-store";
import { NAV_ITEMS } from "@/components/layout/nav-config";
import { TenantSwitcher } from "@/components/layout/tenant-switcher";

interface AppSidebarProps {
  auth: AuthState;
  tenantSlug: string;
  onNavigate?: () => void;
}

/** Sidebar nav list — used in the desktop rail and the mobile sheet. */
export function AppSidebar({ auth, tenantSlug, onNavigate }: AppSidebarProps) {
  const pathname = usePathname();
  const role = auth.activeTenant?.role;

  const items = NAV_ITEMS.filter((item) => can(role, item.permission));

  return (
    <div className="flex h-full flex-col gap-2">
      <div className="px-2 pt-2">
        <TenantSwitcher memberships={auth.memberships} activeTenant={auth.activeTenant} />
      </div>

      <nav aria-label="Main navigation" className="flex-1 space-y-1 px-2 py-2">
        {items.map((item) => {
          const href = `/${tenantSlug}/${item.segment}`;
          const isActive = pathname.startsWith(href);
          const Icon = item.icon;
          return (
            <Link
              key={item.segment}
              href={href}
              onClick={onNavigate}
              className={cn(
                "flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm font-medium transition-colors",
                isActive
                  ? "bg-primary text-primary-foreground shadow-sm"
                  : "text-sidebar-foreground hover:bg-sidebar-accent",
              )}
            >
              <Icon className="size-4 shrink-0" aria-hidden />
              {item.label}
            </Link>
          );
        })}
      </nav>

      <p className="px-4 pb-4 text-xs text-muted-foreground">
        Signed in as <span className="font-medium capitalize">{role}</span>
      </p>
    </div>
  );
}
