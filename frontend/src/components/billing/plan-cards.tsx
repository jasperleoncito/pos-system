"use client";

import { Check } from "lucide-react";

import { usePlans } from "@/hooks/use-billing";
import { formatCentavos } from "@/lib/currency";
import type { BillingPlan } from "@/types/billing";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface PlanCardsProps {
  value: BillingPlan;
  onChange: (plan: BillingPlan) => void;
  disabled?: boolean;
}

/** Selectable monthly/yearly plan cards with live platform prices. */
export function PlanCards({ value, onChange, disabled }: PlanCardsProps) {
  const { data: plans, isLoading } = usePlans();

  if (isLoading || !plans) {
    return (
      <div className="grid gap-3 sm:grid-cols-2">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  const options: { plan: BillingPlan; label: string; price: number; per: string; hint?: string }[] = [
    { plan: "monthly", label: "Monthly", price: plans.monthly_price, per: "/month" },
    {
      plan: "yearly",
      label: "Yearly",
      price: plans.yearly_price,
      per: "/year",
      hint:
        plans.yearly_price < plans.monthly_price * 12
          ? `Save ${formatCentavos(plans.monthly_price * 12 - plans.yearly_price)} vs monthly`
          : undefined,
    },
  ];

  return (
    <div className="grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label="Subscription plan">
      {options.map((o) => {
        const selected = value === o.plan;
        return (
          <button
            key={o.plan}
            type="button"
            role="radio"
            aria-checked={selected}
            disabled={disabled}
            onClick={() => onChange(o.plan)}
            className={cn(
              "relative min-h-[6rem] rounded-lg border p-4 text-left transition-colors",
              selected ? "border-primary bg-primary/5 ring-1 ring-primary" : "hover:bg-accent/5",
              disabled && "opacity-60",
            )}
          >
            {selected && (
              <span className="absolute right-3 top-3 flex size-5 items-center justify-center rounded-full bg-primary text-primary-foreground">
                <Check className="size-3.5" aria-hidden />
              </span>
            )}
            <p className="text-sm font-medium">{o.label}</p>
            <p className="text-xl font-bold tracking-tight">
              {formatCentavos(o.price)}
              <span className="text-sm font-normal text-muted-foreground">{o.per}</span>
            </p>
            {o.hint && <p className="mt-1 text-xs text-emerald-600">{o.hint}</p>}
          </button>
        );
      })}
    </div>
  );
}
