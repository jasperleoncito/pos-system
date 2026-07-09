"use client";

import { useEffect, type ReactNode } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { motion } from "motion/react";
import { BarChart3, ChefHat, Flame, Zap } from "lucide-react";

import { useAuth } from "@/hooks/use-auth";

const VALUE_PROPS = [
  { icon: Zap, title: "Sell in seconds", body: "Tap-to-cart ordering on any phone, tablet, or PC." },
  { icon: Flame, title: "Live kitchen tickets", body: "Orders stream straight to the kitchen — no paper." },
  { icon: BarChart3, title: "Know your numbers", body: "Sales, profit, and stock, updated as you sell." },
];

/**
 * Shared shell for login / register / forgot / reset. Two-column on
 * lg+: a warm branded panel on the left, the form on the right. Already
 * authenticated visitors are bounced to their dashboard.
 */
export default function AuthLayout({ children }: { children: ReactNode }) {
  const router = useRouter();
  const { auth, isReady } = useAuth();

  useEffect(() => {
    if (!isReady || !auth) return;
    if (auth.user.is_super_admin) {
      router.replace("/admin/tenants");
      return;
    }
    const slug = auth.activeTenant?.tenant_slug;
    if (slug) router.replace(`/${slug}/dashboard`);
  }, [isReady, auth, router]);

  return (
    <div className="grid min-h-dvh lg:grid-cols-[1.05fr_1fr]">
      {/* Brand panel — hidden on small screens */}
      <aside className="bg-warm-hero relative hidden overflow-hidden lg:flex lg:flex-col lg:justify-between lg:p-12">
        <div className="warm-grain pointer-events-none absolute inset-0 opacity-60" aria-hidden />
        <div
          className="pointer-events-none absolute -right-28 -top-28 size-96 rounded-full bg-white/15 blur-3xl"
          aria-hidden
        />

        <Link href="/" className="relative flex items-center gap-3">
          <span className="flex size-11 items-center justify-center rounded-2xl bg-white/15 shadow-sm backdrop-blur">
            <ChefHat className="size-6" aria-hidden />
          </span>
          <span className="text-lg font-semibold tracking-tight">POS System</span>
        </Link>

        <motion.div
          className="relative space-y-8"
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4, ease: "easeOut" }}
        >
          <h1 className="max-w-md font-display text-4xl font-semibold leading-[1.1] tracking-tight text-balance xl:text-5xl">
            Run your eatery, the modern way.
          </h1>
          <p className="max-w-md text-primary-foreground/80">
            Point of sale, kitchen display, inventory, attendance, and analytics —
            one account for every business you own, branded your way.
          </p>

          <ul className="space-y-4">
            {VALUE_PROPS.map((p) => (
              <li key={p.title} className="flex items-start gap-3">
                <span className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-xl bg-white/15 backdrop-blur">
                  <p.icon className="size-4" aria-hidden />
                </span>
                <div>
                  <p className="font-medium">{p.title}</p>
                  <p className="text-sm text-primary-foreground/70">{p.body}</p>
                </div>
              </li>
            ))}
          </ul>
        </motion.div>

        <p className="relative text-sm text-primary-foreground/60">
          Built for Filipino restaurants — ₱, GCash &amp; Maya, VAT-ready receipts.
        </p>
      </aside>

      <main className="flex items-center justify-center p-6 sm:p-10">
        <div className="w-full max-w-md">{children}</div>
      </main>
    </div>
  );
}
