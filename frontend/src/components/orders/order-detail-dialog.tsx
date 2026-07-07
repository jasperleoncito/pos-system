"use client";

import { useState } from "react";
import { Loader2, Printer, RotateCcw, XCircle } from "lucide-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import { formatCentavos } from "@/lib/currency";
import { can } from "@/lib/rbac";
import { useAuth } from "@/hooks/use-auth";
import type { Order } from "@/types/order";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

const STATUS_COLORS: Record<string, string> = {
  completed: "bg-emerald-600",
  open: "bg-sky-600",
  held: "bg-amber-600",
  voided: "bg-neutral-500",
  refunded: "bg-rose-600",
  partially_refunded: "bg-orange-600",
};

interface OrderDetailDialogProps {
  orderId: string | null;
  onClose: () => void;
  onPrintReceipt: (orderId: string) => void;
}

export function OrderDetailDialog({ orderId, onClose, onPrintReceipt }: OrderDetailDialogProps) {
  const { auth } = useAuth();
  const queryClient = useQueryClient();
  const role = auth?.activeTenant?.role;

  const [action, setAction] = useState<"refund" | "void" | null>(null);
  const [reason, setReason] = useState("");
  const [refundPesos, setRefundPesos] = useState("");

  const { data: order, isLoading } = useQuery({
    queryKey: ["orders", "detail", orderId],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Order>>(`/orders/${orderId}`);
      return res.data.data;
    },
    enabled: Boolean(orderId),
  });

  const mutate = useMutation({
    mutationFn: async () => {
      if (action === "void") {
        const res = await api.post<ApiEnvelope<Order>>(`/orders/${orderId}/void`, { reason });
        return res.data.data;
      }
      const amount = refundPesos ? Math.round(Number(refundPesos) * 100) : 0;
      const res = await api.post<ApiEnvelope<Order>>(`/orders/${orderId}/refunds`, {
        reason,
        amount,
      });
      return res.data.data;
    },
    onSuccess: (o) => {
      toast.success(action === "void" ? "Order voided" : "Refund issued");
      queryClient.setQueryData(["orders", "detail", orderId], o);
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["drawer"] });
      setAction(null);
      setReason("");
      setRefundPesos("");
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });

  if (!orderId) return null;

  const canRefund =
    can(role, "orders:refund") &&
    (order?.status === "completed" || order?.status === "partially_refunded");
  const canVoid =
    can(role, "orders:void") &&
    order &&
    !["voided", "refunded", "partially_refunded"].includes(order.status);

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-md">
        {isLoading || !order ? (
          <Skeleton className="h-72 w-full" />
        ) : (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                Order #{order.order_number}
                <Badge className={STATUS_COLORS[order.status]}>
                  {order.status.replace("_", " ")}
                </Badge>
              </DialogTitle>
              <DialogDescription>
                {new Date(order.created_at).toLocaleString("en-PH")} ·{" "}
                {order.order_type.replace("_", " ")}
                {order.table_number && ` · Table ${order.table_number}`}
                {order.cashier_name && ` · ${order.cashier_name}`}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-3 text-sm">
              {(order.items ?? []).map((item) => (
                <div key={item.id} className="flex justify-between gap-2">
                  <span>
                    {item.qty} × {item.name}
                    {item.variant_name && ` (${item.variant_name})`}
                    {(item.modifiers ?? []).length > 0 && (
                      <span className="block text-xs text-muted-foreground">
                        {(item.modifiers ?? []).map((m) => m.name).join(", ")}
                      </span>
                    )}
                  </span>
                  <span className="tabular-nums">{formatCentavos(item.line_total)}</span>
                </div>
              ))}

              <Separator />

              {order.discount_total > 0 && (
                <div className="flex justify-between text-emerald-700 dark:text-emerald-400">
                  <span>Discount</span>
                  <span className="tabular-nums">−{formatCentavos(order.discount_total)}</span>
                </div>
              )}
              <div className="flex justify-between font-semibold">
                <span>Total</span>
                <span className="tabular-nums">{formatCentavos(order.total)}</span>
              </div>

              {(order.payments ?? []).map((p) => (
                <div key={p.id} className="flex justify-between text-muted-foreground">
                  <span className="capitalize">
                    {p.method.replace("_", " ")}
                    {p.reference_no && ` · ${p.reference_no}`}
                  </span>
                  <span className="tabular-nums">{formatCentavos(p.amount)}</span>
                </div>
              ))}
              {order.change > 0 && (
                <div className="flex justify-between text-muted-foreground">
                  <span>Change</span>
                  <span className="tabular-nums">{formatCentavos(order.change)}</span>
                </div>
              )}

              {(order.refunds ?? []).length > 0 && (
                <>
                  <Separator />
                  {(order.refunds ?? []).map((r) => (
                    <div key={r.id} className="flex justify-between text-rose-600">
                      <span>Refund #{r.refund_number} — {r.reason}</span>
                      <span className="tabular-nums">−{formatCentavos(r.amount)}</span>
                    </div>
                  ))}
                </>
              )}

              {order.void_reason && (
                <p className="text-xs text-muted-foreground">Void reason: {order.void_reason}</p>
              )}
            </div>

            {action === null ? (
              <DialogFooter className="flex-wrap gap-2">
                <Button variant="outline" size="sm" onClick={() => onPrintReceipt(order.id)}>
                  <Printer className="size-4" aria-hidden />
                  Receipt
                </Button>
                {canRefund && (
                  <Button variant="outline" size="sm" onClick={() => setAction("refund")}>
                    <RotateCcw className="size-4" aria-hidden />
                    Refund
                  </Button>
                )}
                {canVoid && (
                  <Button variant="destructive" size="sm" onClick={() => setAction("void")}>
                    <XCircle className="size-4" aria-hidden />
                    Void
                  </Button>
                )}
              </DialogFooter>
            ) : (
              <div className="space-y-3 rounded-lg border p-3">
                <p className="text-sm font-semibold">
                  {action === "void" ? "Void this order" : "Refund this order"}
                </p>
                {action === "refund" && (
                  <div className="space-y-2">
                    <Label htmlFor="refund-amount">Amount (PHP) — blank for full refund</Label>
                    <Input
                      id="refund-amount"
                      type="number"
                      min="0"
                      step="0.01"
                      inputMode="decimal"
                      placeholder={(order.total / 100).toFixed(2)}
                      value={refundPesos}
                      onChange={(e) => setRefundPesos(e.target.value)}
                    />
                  </div>
                )}
                <div className="space-y-2">
                  <Label htmlFor="action-reason">Reason</Label>
                  <Input
                    id="action-reason"
                    placeholder={action === "void" ? "e.g. wrong order entered" : "e.g. food quality issue"}
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                  />
                </div>
                <div className="flex justify-end gap-2">
                  <Button variant="outline" size="sm" onClick={() => setAction(null)}>
                    Cancel
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={mutate.isPending || reason.trim().length < 3}
                    onClick={() => mutate.mutate()}
                  >
                    {mutate.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
                    Confirm {action}
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
