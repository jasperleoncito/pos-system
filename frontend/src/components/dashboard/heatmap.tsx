"use client";

import { useMemo } from "react";

import { formatCentavos } from "@/lib/currency";
import type { HeatCell } from "@/hooks/use-analytics";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

/**
 * Day × hour sales heatmap. Sequential single hue: the brand color at
 * opacity steps scaled to the busiest cell (light → dark = low → high).
 */
export function SalesHeatmap({ cells }: { cells?: HeatCell[] | null }) {
  const { grid, max } = useMemo(() => {
    const grid = new Map<string, HeatCell>();
    let max = 0;
    for (const c of cells ?? []) {
      grid.set(`${c.day_of_week}-${c.hour}`, c);
      if (c.sales > max) max = c.sales;
    }
    return { grid, max };
  }, [cells]);

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Busy times</CardTitle>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <div className="min-w-[560px]">
          <div className="grid grid-cols-[2.5rem_repeat(24,1fr)] gap-0.5">
            <div />
            {Array.from({ length: 24 }, (_, h) => (
              <div key={h} className="pb-1 text-center text-[10px] text-muted-foreground">
                {h % 3 === 0 ? `${h}`.padStart(2, "0") : ""}
              </div>
            ))}
            {DAYS.map((day, dow) => (
              <div key={day} className="contents">
                <div className="pr-2 text-right text-[11px] leading-4 text-muted-foreground">{day}</div>
                {Array.from({ length: 24 }, (_, h) => {
                  const cell = grid.get(`${dow}-${h}`);
                  const intensity = cell && max > 0 ? Math.max(0.15, cell.sales / max) : 0;
                  return (
                    <div
                      key={h}
                      className="aspect-square rounded-[3px] bg-muted"
                      title={
                        cell
                          ? `${day} ${`${h}`.padStart(2, "0")}:00 — ${formatCentavos(cell.sales)} · ${cell.orders} orders`
                          : `${day} ${`${h}`.padStart(2, "0")}:00 — no sales`
                      }
                      style={intensity > 0 ? { backgroundColor: "var(--chart-1)", opacity: intensity } : undefined}
                    />
                  );
                })}
              </div>
            ))}
          </div>
          <div className="mt-2 flex items-center justify-end gap-1.5 text-[10px] text-muted-foreground">
            less
            {[0.15, 0.4, 0.65, 1].map((o) => (
              <span
                key={o}
                className="size-2.5 rounded-[3px]"
                style={{ backgroundColor: "var(--chart-1)", opacity: o }}
                aria-hidden
              />
            ))}
            more
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
