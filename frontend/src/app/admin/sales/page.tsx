"use client";

import { useState } from "react";
import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { PhilippinePeso, Receipt, ShoppingBag, TrendingUp } from "lucide-react";

import { useAdminSales, type PlatformSalesPoint } from "@/hooks/use-tenant";
import { formatCentavos } from "@/lib/currency";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

const RANGES = [
  { days: 7, label: "7 days" },
  { days: 30, label: "30 days" },
  { days: 90, label: "90 days" },
];

function shortDate(d: string) {
  return new Date(`${d}T00:00:00`).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

interface TooltipPayload {
  value?: number;
  payload?: PlatformSalesPoint;
}

function SalesTooltip({ active, payload, label }: { active?: boolean; payload?: TooltipPayload[]; label?: string }) {
  if (!active || !payload?.length) return null;
  const point = payload[0]?.payload;
  return (
    <div className="rounded-lg border bg-popover px-3 py-2 text-xs shadow-md">
      <p className="font-medium">{label ? shortDate(label) : ""}</p>
      <p className="tabular-nums">{formatCentavos(Number(payload[0]?.value ?? 0))}</p>
      {point && <p className="text-muted-foreground">{point.orders} orders</p>}
    </div>
  );
}

export default function AdminSalesPage() {
  const [days, setDays] = useState(30);
  const { data, isLoading } = useAdminSales(days);

  const aov = data && data.orders > 0 ? Math.round(data.gross_sales / data.orders) : 0;
  const maxSales = Math.max(1, ...(data?.top_businesses.map((b) => b.sales) ?? [1]));

  const tiles = [
    { label: "Gross sales", value: data ? formatCentavos(data.gross_sales) : "—", icon: PhilippinePeso },
    { label: "Orders", value: data ? data.orders.toLocaleString() : "—", icon: ShoppingBag },
    { label: "Avg order value", value: data ? formatCentavos(aov) : "—", icon: Receipt },
    { label: "Subscription revenue", value: data ? formatCentavos(data.subscription_revenue) : "—", icon: TrendingUp },
  ];

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">Sales analytics</h1>
          <p className="text-muted-foreground">Platform-wide sales across every business</p>
        </div>
        <div className="flex gap-1 rounded-lg border p-1">
          {RANGES.map((r) => (
            <Button
              key={r.days}
              variant={days === r.days ? "default" : "ghost"}
              size="sm"
              onClick={() => setDays(r.days)}
            >
              {r.label}
            </Button>
          ))}
        </div>
      </header>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {tiles.map((t) => (
          <Card key={t.label} className="py-4">
            <CardContent className="px-4">
              <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <t.icon className="size-3.5" aria-hidden />
                {t.label}
              </p>
              <p className="text-xl font-bold tabular-nums tracking-tight">{t.value}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Sales over time</CardTitle>
          <CardDescription>Gross sales per day · last {days} days</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <Skeleton className="h-64 w-full" />
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <AreaChart data={data?.series ?? []} margin={{ top: 4, right: 8, left: 8, bottom: 0 }}>
                <defs>
                  <linearGradient id="salesFill" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#16a34a" stopOpacity={0.35} />
                    <stop offset="100%" stopColor="#16a34a" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis
                  dataKey="date"
                  tickFormatter={shortDate}
                  tick={{ fontSize: 11 }}
                  interval="preserveStartEnd"
                  minTickGap={24}
                  stroke="currentColor"
                  className="text-muted-foreground"
                />
                <YAxis
                  tickFormatter={(v) => `₱${Math.round(Number(v) / 100).toLocaleString()}`}
                  tick={{ fontSize: 11 }}
                  width={56}
                  stroke="currentColor"
                  className="text-muted-foreground"
                />
                <Tooltip content={<SalesTooltip />} />
                <Area type="monotone" dataKey="sales" stroke="#16a34a" strokeWidth={2} fill="url(#salesFill)" />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Top businesses</CardTitle>
          <CardDescription>By gross sales · last {days} days</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          {isLoading && Array.from({ length: 4 }, (_, i) => <Skeleton key={i} className="h-12 w-full" />)}
          {data?.top_businesses.map((b, i) => (
            <div key={b.tenant_id} className="relative overflow-hidden rounded-lg border p-3">
              <div className="absolute inset-y-0 left-0 bg-primary/5" style={{ width: `${(b.sales / maxSales) * 100}%` }} aria-hidden />
              <div className="relative flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">
                    <span className="text-muted-foreground">{i + 1}.</span> {b.name}
                  </p>
                  <p className="truncate text-xs text-muted-foreground">/{b.slug} · {b.orders} orders</p>
                </div>
                <p className="shrink-0 text-sm font-semibold tabular-nums">{formatCentavos(b.sales)}</p>
              </div>
            </div>
          ))}
          {data && data.top_businesses.length === 0 && (
            <p className="py-8 text-center text-sm text-muted-foreground">No sales in this period yet.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
