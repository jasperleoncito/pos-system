"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useTheme } from "next-themes";
import { ChefHat, Moon, Sun } from "lucide-react";

import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const LINKS = [
  { href: "#features", label: "Features" },
  { href: "#pricing", label: "Pricing" },
  { href: "#faq", label: "FAQ" },
];

/** Resolve where an authenticated user's "Dashboard" button should go. */
function dashboardHref(auth: ReturnType<typeof useAuth>["auth"]): string | null {
  if (!auth) return null;
  if (auth.user.is_super_admin) return "/admin/tenants";
  const slug = auth.activeTenant?.tenant_slug;
  return slug ? `/${slug}/dashboard` : null;
}

export function LandingNav() {
  const { auth, isReady } = useAuth();
  const { resolvedTheme, setTheme } = useTheme();
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const dash = dashboardHref(auth);

  return (
    <header
      className={cn(
        "sticky top-0 z-50 border-b transition-colors",
        scrolled ? "border-border bg-background/80 backdrop-blur" : "border-transparent",
      )}
    >
      <nav className="mx-auto flex h-16 max-w-6xl items-center gap-4 px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-2.5">
          <span className="flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <ChefHat className="size-5" aria-hidden />
          </span>
          <span className="text-base font-semibold tracking-tight">POS System</span>
        </Link>

        <div className="hidden flex-1 items-center justify-center gap-1 md:flex">
          {LINKS.map((l) => (
            <a
              key={l.href}
              href={l.href}
              className="rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {l.label}
            </a>
          ))}
        </div>

        <div className="ml-auto flex items-center gap-2 md:ml-0">
          <Button
            variant="ghost"
            size="icon"
            aria-label="Toggle theme"
            onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
          >
            <Sun className="size-5 dark:hidden" aria-hidden />
            <Moon className="hidden size-5 dark:block" aria-hidden />
          </Button>

          {isReady && dash ? (
            <Button asChild>
              <Link href={dash}>Go to dashboard</Link>
            </Button>
          ) : (
            <>
              <Button asChild variant="ghost" className="hidden sm:inline-flex">
                <Link href="/login">Log in</Link>
              </Button>
              <Button asChild>
                <Link href="/register">Get started</Link>
              </Button>
            </>
          )}
        </div>
      </nav>
    </header>
  );
}
