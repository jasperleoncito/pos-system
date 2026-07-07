"use client";

import { useState } from "react";
import { motion } from "motion/react";

import { useAuth } from "@/hooks/use-auth";
import { useDashboard, useOverview } from "@/hooks/use-analytics";
import { formatCentavos } from "@/lib/currency";
import { cn } from "@/lib/utils";
import { StatCards } from "@/components/dashboard/stat-cards";
import { HourlyChart, PaymentDonut } from "@/components/dashboard/charts";
import { SalesHeatmap } from "@/components/dashboard/heatmap";
import { TopList } from "@/components/dashboard/top-lists";
import { ExpensesCard } from "@/components/dashboard/expenses-card";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

/** Local YYYY-MM-DD. */
function isoDate(d: Date): string {
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${month}-${day}`;
}

function daysAgo(n: number): string {
  return isoDate(new Date(Date.now() - n * 86_400_000));
}

const PRESETS = [
  { label: "Today", days: 0 },
  { label: "7 days", days: 6 },
  { label: "30 days", days: 29 },
  { label: "90 days", days: 89 },
];

export default function DashboardPage() {
  const { auth } = useAuth();
  const [from, setFrom] = useState(() => daysAgo(6));
  const [to, setTo] = useState(() => daysAgo(0));

  const { data: overview } = useOverview();
  const { data: dashboard, isLoading } = useDashboard(from, to);
  const summary = dashboard?.summary;

  if (!auth) return null;
  const firstName = auth.user.full_name.split(" ")[0];

  const applyPreset = (days: number) => {
    setFrom(daysAgo(days));
    setTo(daysAgo(0));
  };
  const activePreset = PRESETS.find((p) => from === daysAgo(p.days) && to === daysAgo(0))?.label;

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
      className="space-y-5"
    >
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Welcome back, {firstName}</h1>
        <p className="text-muted-foreground">Here’s how {auth.activeTenant?.tenant_name} is doing.</p>
      </header>

      <StatCards stats={overview} />

      {/* Range filter row */}
      <div className="flex flex-wrap items-center gap-2">
        {PRESETS.map((preset) => (
          <button
            key={preset.label}
            type="button"
            onClick={() => applyPreset(preset.days)}
            className={cn(
              "min-h-9 cursor-pointer rounded-full border px-3.5 text-sm font-medium transition-colors",
              activePreset === preset.label
                ? "border-primary bg-primary text-primary-foreground"
                : "hover:bg-accent/10",
            )}
          >
            {preset.label}
          </button>
        ))}
        <div className="ml-auto flex items-center gap-2">
          <Input
            type="date"
            value={from}
            max={to}
            onChange={(e) => setFrom(e.target.value)}
            aria-label="From date"
            className="w-36"
          />
          <span className="text-sm text-muted-foreground">to</span>
          <Input
            type="date"
            value={to}
            min={from}
            onChange={(e) => setTo(e.target.value)}
            aria-label="To date"
            className="w-36"
          />
        </div>
      </div>

      {/* Summary strip */}
      {isLoading || !summary ? (
        <Skeleton className="h-24 w-full rounded-xl" />
      ) : (
        <Card className="py-4">
          <CardContent className="grid grid-cols-2 gap-x-6 gap-y-4 px-4 sm:grid-cols-3 lg:grid-cols-6">
            {[
              { label: "Net sales", value: formatCentavos(summary.net_sales) },
              { label: "Profit", value: formatCentavos(summary.profit), highlight: true },
              { label: "COGS", value: formatCentavos(summary.cogs) },
              { label: "Expenses", value: formatCentavos(summary.expenses) },
              { label: "Refunds", value: formatCentavos(summary.refunds) },
              { label: "Avg order", value: formatCentavos(summary.aov) },
            ].map((item) => (
              <div key={item.label}>
                <p className="text-xs text-muted-foreground">{item.label}</p>
                <p
                  className={cn(
                    "text-lg font-bold tabular-nums tracking-tight",
                    item.highlight && (summary.profit >= 0
                      ? "text-emerald-700 dark:text-emerald-400"
                      : "text-rose-700 dark:text-rose-400"),
                  )}
                >
                  {item.value}
                </p>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Charts */}
      <div className="grid gap-4 lg:grid-cols-5">
        <div className="lg:col-span-3"><HourlyChart hourly={dashboard?.hourly} /></div>
        <div className="lg:col-span-2"><PaymentDonut mix={dashboard?.payment_mix} /></div>
      </div>

      <SalesHeatmap cells={dashboard?.heatmap} />

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <TopList title="Top products" entries={dashboard?.top_products} unit="qty" />
        <TopList title="Top categories" entries={dashboard?.top_categories} unit="qty" />
        <TopList title="Best employees" entries={dashboard?.top_employees} unit="orders" />
      </div>

      <ExpensesCard from={from} to={to} />
    </motion.div>
  );
}
