"use client";

import { formatCentavos } from "@/lib/currency";
import type { TopEntry } from "@/hooks/use-analytics";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface TopListProps {
  title: string;
  entries?: TopEntry[] | null;
  unit: "qty" | "orders";
}

/** Ranked list with proportional bars — magnitude in one hue. */
export function TopList({ title, entries, unit }: TopListProps) {
  const list = entries ?? [];
  const max = list.reduce((m, e) => Math.max(m, e.revenue), 0);

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {list.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">No sales in range.</p>
        ) : (
          <ul className="space-y-2.5">
            {list.map((entry) => (
              <li key={entry.name} className="space-y-1">
                <div className="flex items-baseline justify-between gap-2 text-sm">
                  <span className="min-w-0 truncate font-medium">{entry.name}</span>
                  <span className="shrink-0 tabular-nums text-muted-foreground">
                    {formatCentavos(entry.revenue)}
                    <span className="ml-1 text-xs">
                      · {unit === "qty" ? `${entry.qty ?? 0}×` : `${entry.orders ?? 0} orders`}
                    </span>
                  </span>
                </div>
                <div className="h-1.5 overflow-hidden rounded-full bg-muted">
                  <div
                    className="h-full rounded-full"
                    style={{
                      width: max > 0 ? `${Math.max(2, (entry.revenue / max) * 100)}%` : 0,
                      backgroundColor: "var(--chart-1)",
                    }}
                  />
                </div>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
