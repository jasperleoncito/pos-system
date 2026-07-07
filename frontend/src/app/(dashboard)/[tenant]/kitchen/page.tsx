"use client";

import { useEffect, useState } from "react";
import { CheckCheck, ChefHat, Flame } from "lucide-react";

import {
  useKitchenChime,
  useKitchenOrders,
  useKitchenStream,
  useSetItemStatus,
  useSetKitchenStatus,
} from "@/hooks/use-kitchen";
import { cn } from "@/lib/utils";
import type { Order, OrderItem } from "@/types/order";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

const COLUMNS = [
  { status: "pending", title: "New", accent: "border-t-sky-500" },
  { status: "preparing", title: "Preparing", accent: "border-t-amber-500" },
  { status: "ready", title: "Ready", accent: "border-t-emerald-500" },
] as const;

const NEXT_STATUS: Record<string, { to: string; label: string }> = {
  pending: { to: "preparing", label: "Start" },
  preparing: { to: "ready", label: "Ready" },
  ready: { to: "completed", label: "Done" },
};

const WARN_MINUTES = 5;
const LATE_MINUTES = 10;

function ElapsedBadge({ createdAt, now }: { createdAt: string; now: number }) {
  const minutes = Math.floor((now - new Date(createdAt).getTime()) / 60_000);
  return (
    <span
      className={cn(
        "rounded-md px-1.5 py-0.5 text-xs font-bold tabular-nums",
        minutes >= LATE_MINUTES
          ? "bg-rose-600 text-white"
          : minutes >= WARN_MINUTES
            ? "bg-amber-500 text-white"
            : "bg-muted text-muted-foreground",
      )}
    >
      {minutes}m
    </span>
  );
}

function TicketItem({ order, item }: { order: Order; item: OrderItem }) {
  const setItemStatus = useSetItemStatus();
  const isDone = item.status === "ready" || item.status === "completed";

  return (
    <button
      type="button"
      onClick={() =>
        setItemStatus.mutate({
          orderId: order.id,
          itemId: item.id,
          status: isDone ? "pending" : "ready",
        })
      }
      className={cn(
        "w-full cursor-pointer rounded-md px-2 py-1.5 text-left transition-colors hover:bg-accent/10",
        isDone && "opacity-50",
      )}
    >
      <p className={cn("text-sm font-semibold leading-tight", isDone && "line-through")}>
        {item.qty} × {item.name}
        {item.variant_name && <span className="font-normal"> ({item.variant_name})</span>}
      </p>
      {(item.modifiers ?? []).length > 0 && (
        <p className="text-xs text-muted-foreground">
          {(item.modifiers ?? []).map((m) => m.name).join(" · ")}
        </p>
      )}
      {item.notes && <p className="text-xs font-medium text-amber-600">“{item.notes}”</p>}
    </button>
  );
}

export default function KitchenPage() {
  const { data: orders, isLoading } = useKitchenOrders();
  const setStatus = useSetKitchenStatus();
  const chime = useKitchenChime();
  useKitchenStream(chime);

  // Ticking clock for elapsed-time badges.
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 15_000);
    return () => clearInterval(timer);
  }, []);

  return (
    <div className="flex h-[calc(100dvh-3.5rem-2rem)] flex-col gap-4 sm:h-[calc(100dvh-3.5rem-3rem)]">
      <header className="flex items-center gap-3">
        <div className="flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground">
          <ChefHat className="size-5" aria-hidden />
        </div>
        <div>
          <h1 className="text-xl font-bold tracking-tight">Kitchen Display</h1>
          <p className="text-sm text-muted-foreground">
            {(orders ?? []).length} active {(orders ?? []).length === 1 ? "ticket" : "tickets"}
          </p>
        </div>
      </header>

      <div className="grid flex-1 grid-cols-1 gap-3 overflow-y-auto md:grid-cols-3 md:overflow-hidden">
        {COLUMNS.map((column) => {
          const columnOrders = (orders ?? []).filter((o) => o.kitchen_status === column.status);
          return (
            <section
              key={column.status}
              aria-label={column.title}
              className={cn(
                "flex min-h-40 flex-col rounded-xl border border-t-4 bg-card md:overflow-hidden",
                column.accent,
              )}
            >
              <h2 className="flex items-center justify-between px-3 py-2 text-sm font-bold uppercase tracking-wide">
                {column.title}
                <Badge variant="secondary">{columnOrders.length}</Badge>
              </h2>

              <div className="flex-1 space-y-2 overflow-y-auto p-2 pt-0">
                {isLoading && <Skeleton className="h-32 w-full" />}

                {columnOrders.map((order) => {
                  const next = NEXT_STATUS[order.kitchen_status];
                  return (
                    <article
                      key={order.id}
                      className={cn(
                        "rounded-lg border bg-background p-2.5 shadow-sm",
                        order.priority && "border-rose-500 ring-1 ring-rose-500",
                      )}
                    >
                      <div className="mb-1 flex items-center gap-2">
                        <p className="text-base font-extrabold tabular-nums">#{order.order_number}</p>
                        {order.priority && (
                          <Badge className="bg-rose-600">
                            <Flame className="size-3" aria-hidden /> Rush
                          </Badge>
                        )}
                        <span className="flex-1 text-xs capitalize text-muted-foreground">
                          {order.order_type.replace("_", " ")}
                          {order.table_number && ` · T${order.table_number}`}
                        </span>
                        <ElapsedBadge createdAt={order.created_at} now={now} />
                      </div>

                      <div className="divide-y">
                        {(order.items ?? []).map((item) => (
                          <TicketItem key={item.id} order={order} item={item} />
                        ))}
                      </div>

                      {order.notes && (
                        <p className="mt-1 text-xs font-medium text-amber-600">“{order.notes}”</p>
                      )}

                      {next && (
                        <Button
                          className="mt-2 min-h-11 w-full font-semibold"
                          disabled={setStatus.isPending}
                          onClick={() => setStatus.mutate({ orderId: order.id, status: next.to })}
                        >
                          <CheckCheck className="size-4" aria-hidden />
                          {next.label}
                        </Button>
                      )}
                    </article>
                  );
                })}

                {!isLoading && columnOrders.length === 0 && (
                  <p className="py-8 text-center text-sm text-muted-foreground">Empty</p>
                )}
              </div>
            </section>
          );
        })}
      </div>
    </div>
  );
}
