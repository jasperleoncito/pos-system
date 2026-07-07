"use client";

import { TrendingDown, TrendingUp } from "lucide-react";

import { formatCentavos } from "@/lib/currency";
import { cn } from "@/lib/utils";
import type { PeriodStat } from "@/hooks/use-analytics";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

const LABELS: Record<PeriodStat["label"], { title: string; versus: string }> = {
  today: { title: "Today", versus: "vs yesterday" },
  wtd: { title: "This week", versus: "vs last week" },
  mtd: { title: "This month", versus: "vs last month" },
  ytd: { title: "This year", versus: "vs last year" },
};

function Delta({ current, previous, versus }: { current: number; previous: number; versus: string }) {
  if (previous <= 0) {
    return <p className="text-xs text-muted-foreground">no prior data</p>;
  }
  const pct = ((current - previous) / previous) * 100;
  const up = pct >= 0;
  return (
    <p
      className={cn(
        "flex items-center gap-1 text-xs font-medium",
        up ? "text-emerald-700 dark:text-emerald-400" : "text-rose-700 dark:text-rose-400",
      )}
    >
      {up ? <TrendingUp className="size-3.5" aria-hidden /> : <TrendingDown className="size-3.5" aria-hidden />}
      {up ? "+" : ""}
      {pct.toFixed(1)}% <span className="font-normal text-muted-foreground">{versus}</span>
    </p>
  );
}

/** Today / WTD / MTD / YTD sales cards with previous-period deltas. */
export function StatCards({ stats }: { stats?: PeriodStat[] }) {
  if (!stats) {
    return (
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => <Skeleton key={i} className="h-28 w-full rounded-xl" />)}
      </div>
    );
  }
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {stats.map((stat) => {
        const meta = LABELS[stat.label];
        return (
          <Card key={stat.label} className="py-4">
            <CardContent className="space-y-1.5 px-4">
              <p className="text-sm text-muted-foreground">{meta.title}</p>
              <p className="text-2xl font-bold tabular-nums tracking-tight">
                {formatCentavos(stat.sales)}
              </p>
              <div className="flex items-center justify-between gap-2">
                <Delta current={stat.sales} previous={stat.prev_sales} versus={meta.versus} />
                <p className="text-xs tabular-nums text-muted-foreground">{stat.orders} orders</p>
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
