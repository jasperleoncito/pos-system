"use client";

import { motion } from "motion/react";
import {
  BarChart3,
  Boxes,
  ChefHat,
  CreditCard,
  Gift,
  Store,
  Users,
  Zap,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

interface Feature {
  icon: LucideIcon;
  title: string;
  body: string;
}

const FEATURES: Feature[] = [
  {
    icon: Zap,
    title: "Touch-first POS",
    body: "Tap-to-cart ordering with variants, modifiers, split bills, held orders, and cash-drawer sessions.",
  },
  {
    icon: CreditCard,
    title: "Payments & receipts",
    body: "Cash, GCash, Maya, card, and split payments with 80mm thermal receipts, refunds, and voids.",
  },
  {
    icon: ChefHat,
    title: "Kitchen Display",
    body: "Orders stream to the kitchen in real time — mark tickets done, skip the paper.",
  },
  {
    icon: Boxes,
    title: "Inventory & recipes",
    body: "Stock depletes per sale via recipes. Purchase orders, suppliers, and low-stock alerts.",
  },
  {
    icon: BarChart3,
    title: "Sales analytics",
    body: "Net sales, profit, COGS, AOV, hourly charts, and top products over any date range.",
  },
  {
    icon: Users,
    title: "Employees & attendance",
    body: "Clock in/out with late, break, and overtime tracking, schedules, salary, and roles.",
  },
  {
    icon: Gift,
    title: "Customer loyalty",
    body: "Reward points, redemption at checkout, coupons, discounts, and purchase history.",
  },
  {
    icon: Store,
    title: "Multi-business",
    body: "Run unlimited businesses from one account — each isolated, branded, and role-controlled.",
  },
];

export function Features() {
  return (
    <section id="features" className="mx-auto max-w-6xl scroll-mt-20 px-4 py-16 sm:px-6 lg:py-24">
      <div className="mx-auto max-w-2xl text-center">
        <h2 className="font-display text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
          Everything your restaurant needs, in one place
        </h2>
        <p className="mt-3 text-muted-foreground text-pretty">
          From the first tap to the end-of-day report — no stitched-together apps.
        </p>
      </div>

      <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {FEATURES.map((f, i) => (
          <motion.div
            key={f.title}
            initial={{ opacity: 0, y: 12 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, margin: "-60px" }}
            transition={{ duration: 0.35, ease: "easeOut", delay: (i % 4) * 0.05 }}
            className="group rounded-xl border bg-card p-5 shadow-sm transition-shadow hover:shadow-md"
          >
            <span className="flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
              <f.icon className="size-5" aria-hidden />
            </span>
            <h3 className="mt-4 font-semibold">{f.title}</h3>
            <p className="mt-1.5 text-sm text-muted-foreground">{f.body}</p>
          </motion.div>
        ))}
      </div>
    </section>
  );
}
