"use client";

import { Star } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { api, type ApiEnvelope } from "@/lib/api";
import { useLoyaltyHistory, type Customer } from "@/hooks/use-customers";
import { formatCentavos } from "@/lib/currency";
import { cn } from "@/lib/utils";
import type { Order } from "@/types/order";
import { TierBadge } from "@/components/pos/customer-dialog";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

function usePurchaseHistory(customerId: string | null) {
  return useQuery({
    queryKey: ["customers", "orders", customerId],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Order[] | null>>("/orders", {
        params: { customer_id: customerId, limit: 20 },
      });
      return res.data.data ?? [];
    },
    enabled: Boolean(customerId),
  });
}

interface CustomerProfileDialogProps {
  customer: Customer | null;
  onClose: () => void;
}

/** Profile: balance + tier, loyalty ledger, and purchase history. */
export function CustomerProfileDialog({ customer, onClose }: CustomerProfileDialogProps) {
  const { data: history, isLoading: historyLoading } = useLoyaltyHistory(customer?.id ?? null);
  const { data: orders, isLoading: ordersLoading } = usePurchaseHistory(customer?.id ?? null);

  return (
    <Dialog open={customer !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {customer?.full_name}
            {customer && <TierBadge tier={customer.tier} />}
          </DialogTitle>
        </DialogHeader>

        <div className="grid grid-cols-3 gap-2 text-center">
          <div className="rounded-lg border p-3">
            <p className="text-2xl font-bold tabular-nums">{customer?.points_balance ?? 0}</p>
            <p className="text-xs text-muted-foreground">Points balance</p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-2xl font-bold tabular-nums">{customer?.lifetime_points ?? 0}</p>
            <p className="text-xs text-muted-foreground">Lifetime points</p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-2xl font-bold tabular-nums">{orders?.length ?? 0}</p>
            <p className="text-xs text-muted-foreground">Recent orders</p>
          </div>
        </div>

        {(customer?.phone || customer?.email || customer?.birthday) && (
          <p className="text-sm text-muted-foreground">
            {[
              customer?.phone,
              customer?.email,
              customer?.birthday &&
                `🎂 ${new Date(customer.birthday).toLocaleDateString("en-PH", { month: "long", day: "numeric" })}`,
            ]
              .filter(Boolean)
              .join(" · ")}
          </p>
        )}

        <Tabs defaultValue="loyalty">
          <TabsList className="w-full">
            <TabsTrigger value="loyalty" className="flex-1">Loyalty history</TabsTrigger>
            <TabsTrigger value="orders" className="flex-1">Purchases</TabsTrigger>
          </TabsList>

          <TabsContent value="loyalty" className="space-y-2">
            {historyLoading && <Skeleton className="h-32 w-full" />}
            {(history ?? []).map((tx) => (
              <div key={tx.id} className="flex items-center justify-between rounded-lg border p-2.5 text-sm">
                <div>
                  <p className="font-medium capitalize">
                    {tx.type}
                    {tx.order_number ? ` · order #${tx.order_number}` : ""}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {new Date(tx.created_at).toLocaleString("en-PH")} · balance {tx.balance_after}
                  </p>
                </div>
                <span
                  className={cn(
                    "flex items-center gap-1 font-semibold tabular-nums",
                    tx.points < 0 ? "text-rose-600" : "text-emerald-600",
                  )}
                >
                  {tx.points > 0 ? "+" : ""}
                  {tx.points}
                  <Star className="size-3.5" aria-hidden />
                </span>
              </div>
            ))}
            {history && history.length === 0 && (
              <p className="py-6 text-center text-sm text-muted-foreground">No loyalty activity yet.</p>
            )}
          </TabsContent>

          <TabsContent value="orders" className="space-y-2">
            {ordersLoading && <Skeleton className="h-32 w-full" />}
            {(orders ?? []).map((o) => (
              <div key={o.id} className="flex items-center justify-between rounded-lg border p-2.5 text-sm">
                <div>
                  <p className="font-medium">Order #{o.order_number}</p>
                  <p className="text-xs text-muted-foreground">
                    {new Date(o.created_at).toLocaleString("en-PH")}
                  </p>
                </div>
                <div className="text-right">
                  <p className="font-semibold tabular-nums">{formatCentavos(o.total)}</p>
                  <Badge variant="secondary" className="capitalize">{o.status.replace("_", " ")}</Badge>
                </div>
              </div>
            ))}
            {orders && orders.length === 0 && (
              <p className="py-6 text-center text-sm text-muted-foreground">No purchases yet.</p>
            )}
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
