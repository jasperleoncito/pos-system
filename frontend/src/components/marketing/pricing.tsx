"use client";

import Link from "next/link";
import { Check, Loader2 } from "lucide-react";

import { usePlans } from "@/hooks/use-billing";
import { formatCentavos } from "@/lib/currency";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const INCLUDED = [
  "Unlimited businesses",
  "POS, kitchen display & receipts",
  "Inventory, recipes & purchase orders",
  "Employees, attendance & payroll data",
  "Customer loyalty & coupons",
  "Analytics & exportable reports",
];

export function Pricing() {
  const { data: plans, isLoading } = usePlans();

  const monthly = plans?.monthly_price ?? null;
  const yearly = plans?.yearly_price ?? null;
  const yearlySavesMonths =
    monthly && yearly && yearly < monthly * 12
      ? Math.round((monthly * 12 - yearly) / monthly)
      : 0;

  return (
    <section id="pricing" className="scroll-mt-20 border-y bg-muted/30">
      <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6 lg:py-24">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="font-display text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            One simple price. Every feature.
          </h2>
          <p className="mt-3 text-muted-foreground text-pretty">
            No per-terminal fees, no feature tiers — pay monthly or save with yearly.
          </p>
        </div>

        {isLoading ? (
          <div className="mt-12 flex justify-center">
            <Loader2 className="size-6 animate-spin text-muted-foreground" aria-hidden />
          </div>
        ) : (
          <div className="mx-auto mt-12 grid max-w-3xl gap-6 sm:grid-cols-2">
            <PlanCard
              name="Monthly"
              price={monthly}
              per="/month"
              cta="Start monthly"
            />
            <PlanCard
              name="Yearly"
              price={yearly}
              per="/year"
              cta="Start yearly"
              featured
              badge={yearlySavesMonths > 0 ? `${yearlySavesMonths} months free` : "Best value"}
            />
          </div>
        )}

        <div className="mx-auto mt-10 max-w-3xl">
          <ul className="grid gap-x-6 gap-y-2.5 sm:grid-cols-2">
            {INCLUDED.map((item) => (
              <li key={item} className="flex items-center gap-2 text-sm">
                <Check className="size-4 shrink-0 text-primary" aria-hidden />
                {item}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}

function PlanCard({
  name,
  price,
  per,
  cta,
  featured,
  badge,
}: {
  name: string;
  price: number | null;
  per: string;
  cta: string;
  featured?: boolean;
  badge?: string;
}) {
  return (
    <div
      className={cn(
        "relative flex flex-col rounded-2xl border bg-card p-6 shadow-sm",
        featured && "border-primary shadow-md ring-1 ring-primary",
      )}
    >
      {badge && (
        <span className="absolute -top-3 left-6 rounded-full bg-primary px-3 py-1 text-xs font-semibold text-primary-foreground">
          {badge}
        </span>
      )}
      <p className="text-sm font-medium text-muted-foreground">{name}</p>
      <p className="mt-2 flex items-baseline gap-1">
        <span className="font-display text-4xl font-semibold tracking-tight tabular-nums">
          {price !== null ? formatCentavos(price) : "—"}
        </span>
        <span className="text-sm text-muted-foreground">{per}</span>
      </p>
      <Button asChild size="lg" variant={featured ? "default" : "outline"} className="mt-6 w-full">
        <Link href="/register">{cta}</Link>
      </Button>
    </div>
  );
}
