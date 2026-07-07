"use client";

import { useMemo, useState } from "react";
import { CheckCircle2, Loader2 } from "lucide-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import { formatCentavos, pesosToCentavos } from "@/lib/currency";
import { cn } from "@/lib/utils";
import type { Order, PaymentMethod } from "@/types/order";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

const METHODS: { value: PaymentMethod; label: string }[] = [
  { value: "cash", label: "Cash" },
  { value: "gcash", label: "GCash" },
  { value: "card", label: "Card" },
  { value: "maya", label: "Maya" },
  { value: "bank_transfer", label: "Bank" },
];

/** Divides total into n parts; the first part absorbs the remainder. */
export function splitEvenly(total: number, parts: number): number[] {
  const base = Math.floor(total / parts);
  const amounts = Array.from({ length: parts }, () => base);
  amounts[0] += total - base * parts;
  return amounts;
}

interface SplitBillDialogProps {
  order: Order | null; // the unpaid order being split
  onClose: () => void;
  onCompleted: (orderId: string) => void; // all splits paid
}

export function SplitBillDialog({ order, onClose, onCompleted }: SplitBillDialogProps) {
  const queryClient = useQueryClient();
  const [current, setCurrent] = useState<Order | null>(null);
  const [parts, setParts] = useState(2);
  const [payingSplitId, setPayingSplitId] = useState<string | null>(null);
  const [method, setMethod] = useState<PaymentMethod>("cash");
  const [amountPesos, setAmountPesos] = useState("");

  const active = current ?? order;
  const splits = useMemo(() => active?.splits ?? [], [active]);
  const hasSplits = splits.length > 0;

  const createSplits = useMutation({
    mutationFn: async () => {
      const res = await api.post<ApiEnvelope<Order>>(`/orders/${active!.id}/splits`, {
        amounts: splitEvenly(active!.total, parts),
      });
      return res.data.data;
    },
    onSuccess: (o) => setCurrent(o),
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });

  const paySplit = useMutation({
    mutationFn: async ({ splitId, amount }: { splitId: string; amount: number }) => {
      const res = await api.post<ApiEnvelope<Order>>(
        `/orders/${active!.id}/splits/${splitId}/payments`,
        { payments: [{ method, amount }] },
      );
      return res.data.data;
    },
    onSuccess: (o) => {
      setCurrent(o);
      setPayingSplitId(null);
      setAmountPesos("");
      setMethod("cash");
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["drawer"] });
      if (o.status === "completed") {
        onCompleted(o.id);
      }
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });

  if (!active) return null;

  const payingSplit = splits.find((s) => s.id === payingSplitId);

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Split bill — order #{active.order_number}</DialogTitle>
          <DialogDescription>
            Total {formatCentavos(active.total)}
          </DialogDescription>
        </DialogHeader>

        {!hasSplits ? (
          <div className="space-y-4">
            <p className="text-sm font-medium">How many ways?</p>
            <div className="grid grid-cols-5 gap-2">
              {[2, 3, 4, 5, 6].map((n) => (
                <button
                  key={n}
                  type="button"
                  onClick={() => setParts(n)}
                  className={cn(
                    "min-h-12 cursor-pointer rounded-lg border text-lg font-semibold transition-colors",
                    parts === n
                      ? "border-primary bg-primary text-primary-foreground"
                      : "hover:bg-accent/10",
                  )}
                >
                  {n}
                </button>
              ))}
            </div>
            <p className="text-sm text-muted-foreground">
              {splitEvenly(active.total, parts)
                .map((a) => formatCentavos(a))
                .join(" + ")}
            </p>
            <Button
              className="min-h-12 w-full"
              onClick={() => createSplits.mutate()}
              disabled={createSplits.isPending}
            >
              {createSplits.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              Split {parts} ways
            </Button>
          </div>
        ) : (
          <div className="space-y-3">
            {splits.map((split) => (
              <div
                key={split.id}
                className={cn(
                  "rounded-lg border p-3",
                  split.status === "paid" && "border-emerald-300 bg-emerald-50 dark:border-emerald-900 dark:bg-emerald-950/40",
                )}
              >
                <div className="flex items-center justify-between">
                  <p className="text-sm font-medium">Split {split.split_number}</p>
                  <p className="text-sm font-semibold tabular-nums">{formatCentavos(split.amount)}</p>
                </div>

                {split.status === "paid" ? (
                  <p className="mt-1 flex items-center gap-1 text-xs text-emerald-700 dark:text-emerald-400">
                    <CheckCircle2 className="size-3.5" aria-hidden /> Paid
                  </p>
                ) : payingSplitId === split.id ? (
                  <div className="mt-2 space-y-2">
                    <div className="flex flex-wrap gap-1.5">
                      {METHODS.map(({ value, label }) => (
                        <button
                          key={value}
                          type="button"
                          onClick={() => setMethod(value)}
                          className={cn(
                            "min-h-9 cursor-pointer rounded-md border px-3 text-sm font-medium transition-colors",
                            method === value
                              ? "border-primary bg-primary text-primary-foreground"
                              : "hover:bg-accent/10",
                          )}
                        >
                          {label}
                        </button>
                      ))}
                    </div>
                    <div className="flex gap-2">
                      <Input
                        type="number"
                        min="0"
                        step="0.01"
                        inputMode="decimal"
                        placeholder={`${(split.amount / 100).toFixed(2)}`}
                        aria-label={`Split ${split.split_number} amount`}
                        value={amountPesos}
                        onChange={(e) => setAmountPesos(e.target.value)}
                      />
                      <Button
                        disabled={paySplit.isPending}
                        onClick={() => {
                          const amount = amountPesos
                            ? pesosToCentavos(Number(amountPesos))
                            : split.amount;
                          if (amount < split.amount && method !== "cash") {
                            toast.error("Amount does not cover this split");
                            return;
                          }
                          paySplit.mutate({ splitId: split.id, amount });
                        }}
                      >
                        {paySplit.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
                        Pay
                      </Button>
                    </div>
                  </div>
                ) : (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2"
                    onClick={() => {
                      setPayingSplitId(split.id);
                      setAmountPesos("");
                    }}
                  >
                    Take payment
                  </Button>
                )}
              </div>
            ))}

            {payingSplit && method === "cash" && (
              <p className="text-xs text-muted-foreground">
                Cash overpayment is returned as change.
              </p>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
