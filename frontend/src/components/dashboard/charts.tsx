"use client";

import {
  Bar,
  BarChart,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { formatCentavos } from "@/lib/currency";
import type { HourPoint, PaymentSlice } from "@/hooks/use-analytics";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

/**
 * Fixed categorical colors per payment method (Okabe–Ito, CVD-safe).
 * Color follows the entity — a method keeps its hue whatever its rank.
 */
const METHOD_COLORS: Record<string, string> = {
  cash: "#0072B2",
  gcash: "#009E73",
  card: "#E69F00",
  maya: "#56B4E9",
  bank_transfer: "#CC79A7",
  points: "#D55E00",
};

const METHOD_LABELS: Record<string, string> = {
  cash: "Cash",
  gcash: "GCash",
  card: "Card",
  maya: "Maya",
  bank_transfer: "Bank",
  points: "Points",
};

interface TooltipPayload {
  value?: number | string;
  payload?: HourPoint;
}

function PesoTooltip({ active, payload, label }: { active?: boolean; payload?: TooltipPayload[]; label?: string | number }) {
  if (!active || !payload?.length) return null;
  const point = payload[0]?.payload;
  return (
    <div className="rounded-lg border bg-popover px-3 py-2 text-xs shadow-md">
      <p className="font-medium">{String(label).padStart(2, "0")}:00</p>
      <p className="tabular-nums">{formatCentavos(Number(payload[0]?.value ?? 0))}</p>
      {point && <p className="text-muted-foreground">{point.orders} orders</p>}
    </div>
  );
}

/** Hourly sales — one series, brand hue, per-bar hover tooltip. */
export function HourlyChart({ hourly }: { hourly?: HourPoint[] }) {
  const data = hourly ?? [];
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Sales by hour</CardTitle>
      </CardHeader>
      <CardContent className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: 4 }}>
            <XAxis
              dataKey="hour"
              tickFormatter={(h: number) => `${h}`.padStart(2, "0")}
              tickLine={false}
              axisLine={false}
              fontSize={11}
              interval={2}
              stroke="var(--muted-foreground)"
            />
            <YAxis
              tickFormatter={(v: number) =>
                v >= 100_000 ? `₱${Math.round(v / 100_000)}k` : `₱${Math.round(v / 100)}`
              }
              tickLine={false}
              axisLine={false}
              fontSize={11}
              width={44}
              stroke="var(--muted-foreground)"
            />
            <Tooltip content={<PesoTooltip />} cursor={{ fill: "var(--muted)", opacity: 0.5 }} />
            <Bar dataKey="sales" fill="var(--chart-1)" radius={[4, 4, 0, 0]} maxBarSize={22} />
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}

/** Payment mix donut — fixed hue per method + legend with amounts. */
export function PaymentDonut({ mix }: { mix?: PaymentSlice[] | null }) {
  const data = (mix ?? []).filter((s) => s.amount > 0);
  const total = data.reduce((sum, s) => sum + s.amount, 0);

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Payment mix</CardTitle>
      </CardHeader>
      <CardContent>
        {data.length === 0 ? (
          <p className="py-12 text-center text-sm text-muted-foreground">No payments in range.</p>
        ) : (
          <div className="flex items-center gap-4">
            <div className="h-44 w-44 shrink-0">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={data}
                    dataKey="amount"
                    nameKey="method"
                    innerRadius="62%"
                    outerRadius="100%"
                    paddingAngle={2}
                    strokeWidth={0}
                  >
                    {data.map((slice) => (
                      <Cell key={slice.method} fill={METHOD_COLORS[slice.method] ?? "#999999"} />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value) => formatCentavos(Number(value))}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <ul className="min-w-0 flex-1 space-y-1.5 text-sm">
              {data.map((slice) => (
                <li key={slice.method} className="flex items-center justify-between gap-2">
                  <span className="flex min-w-0 items-center gap-2">
                    <span
                      className="size-2.5 shrink-0 rounded-full"
                      style={{ backgroundColor: METHOD_COLORS[slice.method] ?? "#999999" }}
                      aria-hidden
                    />
                    <span className="truncate">{METHOD_LABELS[slice.method] ?? slice.method}</span>
                  </span>
                  <span className="shrink-0 tabular-nums text-muted-foreground">
                    {formatCentavos(slice.amount)}
                    {total > 0 && (
                      <span className="ml-1 text-xs">({Math.round((slice.amount / total) * 100)}%)</span>
                    )}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
