"use client";

import { useEffect, type ReactNode } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import { LogOut, Moon, ShieldCheck, Sun } from "lucide-react";

import { useAuth, useLogout } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

const ADMIN_NAV = [
  { href: "/admin/tenants", label: "Businesses" },
  { href: "/admin/sales", label: "Sales" },
  { href: "/admin/vouchers", label: "Vouchers" },
];

/** Guarded shell for platform super-admin pages. */
export default function AdminLayout({ children }: { children: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { auth, isReady } = useAuth();
  const { resolvedTheme, setTheme } = useTheme();
  const logout = useLogout();

  useEffect(() => {
    if (!isReady) return;
    if (!auth) {
      router.replace("/login");
    } else if (!auth.user.is_super_admin) {
      const slug = auth.activeTenant?.tenant_slug;
      router.replace(slug ? `/${slug}/dashboard` : "/login");
    }
  }, [isReady, auth, router]);

  if (!isReady || !auth?.user.is_super_admin) {
    return (
      <div className="p-6">
        <Skeleton className="mb-6 h-14 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="flex min-h-dvh flex-col">
      <header className="sticky top-0 z-30 flex h-14 items-center gap-3 border-b bg-background/80 px-4 backdrop-blur">
        <Link href="/admin/tenants" className="flex items-center gap-2">
          <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <ShieldCheck className="size-4" aria-hidden />
          </span>
          <span className="text-sm font-semibold">Platform Admin</span>
        </Link>
        <nav aria-label="Admin sections" className="ml-4 flex items-center gap-1">
          {ADMIN_NAV.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "rounded-md px-3 py-1.5 text-sm transition-colors",
                pathname.startsWith(item.href)
                  ? "bg-accent font-medium text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
              )}
            >
              {item.label}
            </Link>
          ))}
        </nav>
        <div className="flex-1" />
        <Button
          variant="ghost"
          size="icon"
          aria-label="Toggle theme"
          onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
        >
          <Sun className="size-5 dark:hidden" aria-hidden />
          <Moon className="hidden size-5 dark:block" aria-hidden />
        </Button>
        <Button variant="ghost" size="sm" onClick={() => logout()}>
          <LogOut className="size-4" aria-hidden />
          Sign out
        </Button>
      </header>
      <main className="flex-1 p-4 sm:p-6">{children}</main>
    </div>
  );
}
