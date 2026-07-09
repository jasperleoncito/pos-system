"use client";

import { useEffect, useMemo, type CSSProperties, type ReactNode } from "react";
import { useParams, useRouter } from "next/navigation";

import { useAuth } from "@/hooks/use-auth";
import { useTenantSettings } from "@/hooks/use-tenant";
import { tenantThemeVars } from "@/lib/theme";
import { AppSidebar } from "@/components/layout/app-sidebar";
import { Topbar } from "@/components/layout/topbar";
import { SubscriptionGate } from "@/components/billing/subscription-gate";
import { Skeleton } from "@/components/ui/skeleton";

/**
 * Client-side guard + app shell for tenant-scoped pages. Redirects to
 * /login when unauthenticated, and re-homes the URL when it doesn't
 * match the active tenant slug.
 */
export default function TenantLayout({ children }: { children: ReactNode }) {
  const router = useRouter();
  const params = useParams<{ tenant: string }>();
  const { auth, isReady } = useAuth();
  const { data: settings } = useTenantSettings();

  const activeSlug = auth?.activeTenant?.tenant_slug;

  // Tenant branding recolors the whole shell via CSS variables.
  const themeStyle = useMemo<CSSProperties | undefined>(() => {
    if (!settings) return undefined;
    return tenantThemeVars({
      primary: settings.primary_color,
      secondary: settings.secondary_color,
      accent: settings.accent_color,
    }) as CSSProperties;
  }, [settings]);

  // Mirror the brand variables onto <html> so Radix portals (dialogs,
  // dropdowns render on document.body, outside this layout) stay branded.
  useEffect(() => {
    if (!themeStyle) return;
    const root = document.documentElement;
    const vars = themeStyle as Record<string, string>;
    for (const [name, value] of Object.entries(vars)) {
      root.style.setProperty(name, value);
    }
    return () => {
      for (const name of Object.keys(vars)) {
        root.style.removeProperty(name);
      }
    };
  }, [themeStyle]);

  useEffect(() => {
    if (!isReady) return;
    if (!auth) {
      router.replace("/login");
      return;
    }
    if (activeSlug && params.tenant !== activeSlug) {
      router.replace(`/${activeSlug}/dashboard`);
    }
  }, [isReady, auth, activeSlug, params.tenant, router]);

  if (!isReady || !auth || (activeSlug && params.tenant !== activeSlug)) {
    return (
      <div className="flex min-h-dvh">
        <div className="hidden w-64 border-r p-4 lg:block">
          <Skeleton className="mb-4 h-10 w-full" />
          <div className="space-y-2">
            {Array.from({ length: 8 }, (_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        </div>
        <div className="flex-1 p-6">
          <Skeleton className="mb-6 h-14 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-dvh" style={themeStyle}>
      <aside className="sticky top-0 hidden h-dvh w-64 shrink-0 border-r bg-sidebar lg:block">
        <AppSidebar auth={auth} tenantSlug={params.tenant} />
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar auth={auth} tenantSlug={params.tenant} />
        <main className="flex-1 p-4 sm:p-6">
          <SubscriptionGate>{children}</SubscriptionGate>
        </main>
      </div>
    </div>
  );
}
