"use client";

import { useEffect, useMemo, useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { formatCentavos, pesosToCentavos } from "@/lib/currency";
import { cn } from "@/lib/utils";
import type { PaymentLineInput, PaymentMethod } from "@/types/order";
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

const METHODS: { value: PaymentMethod; label: string }[] = [
  { value: "cash", label: "Cash" },
  { value: "gcash", label: "GCash" },
  { value: "card", label: "Card" },
  { value: "maya", label: "Maya" },
  { value: "bank_transfer", label: "Bank" },
];

const QUICK_CASH_PESOS = [20, 50, 100, 200, 500, 1000];

interface TenderRow {
  method: PaymentMethod;
  amountPesos: string;
  referenceNo: string;
}

interface PaymentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  totalCentavos: number;
  isPaying: boolean;
  onConfirm: (payments: PaymentLineInput[]) => void;
}

export function PaymentDialog({ open, onOpenChange, totalCentavos, isPaying, onConfirm }: PaymentDialogProps) {
  const [rows, setRows] = useState<TenderRow[]>([]);

  useEffect(() => {
    if (open) {
      setRows([{ method: "cash", amountPesos: "", referenceNo: "" }]);
    }
  }, [open]);

  const paidCentavos = useMemo(
    () => rows.reduce((sum, r) => sum + pesosToCentavos(Number(r.amountPesos) || 0), 0),
    [rows],
  );
  const remaining = Math.max(0, totalCentavos - paidCentavos);
  const change = Math.max(0, paidCentavos - totalCentavos);
  const cashPaid = rows
    .filter((r) => r.method === "cash")
    .reduce((sum, r) => sum + pesosToCentavos(Number(r.amountPesos) || 0), 0);

  const updateRow = (i: number, patch: Partial<TenderRow>) => {
    setRows((prev) => prev.map((r, j) => (j === i ? { ...r, ...patch } : r)));
  };

  const setExact = (i: number) => {
    const otherPaid = rows.reduce(
      (sum, r, j) => (j === i ? sum : sum + pesosToCentavos(Number(r.amountPesos) || 0)),
      0,
    );
    const due = Math.max(0, totalCentavos - otherPaid);
    updateRow(i, { amountPesos: (due / 100).toString() });
  };

  const onSubmit = () => {
    if (paidCentavos < totalCentavos) {
      toast.error("Payments do not cover the total");
      return;
    }
    if (change > cashPaid) {
      toast.error("Change cannot exceed the cash received — lower non-cash amounts");
      return;
    }
    const payments: PaymentLineInput[] = rows
      .filter((r) => Number(r.amountPesos) > 0)
      .map((r) => ({
        method: r.method,
        amount: pesosToCentavos(Number(r.amountPesos)),
        reference_no: r.referenceNo || undefined,
      }));
    onConfirm(payments);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Take payment</DialogTitle>
          <DialogDescription>
            Amount due: <span className="font-semibold text-foreground">{formatCentavos(totalCentavos)}</span>
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Quick cash for the first cash row */}
          <div className="grid grid-cols-3 gap-2">
            {QUICK_CASH_PESOS.map((pesos) => (
              <Button
                key={pesos}
                type="button"
                variant="outline"
                className="min-h-11 tabular-nums"
                onClick={() => {
                  const i = rows.findIndex((r) => r.method === "cash");
                  if (i >= 0) {
                    updateRow(i, { amountPesos: String(pesos) });
                  }
                }}
              >
                ₱{pesos}
              </Button>
            ))}
          </div>

          {rows.map((row, i) => (
            <div key={i} className="space-y-2 rounded-lg border p-3">
              <div className="flex flex-wrap gap-1.5">
                {METHODS.map(({ value, label }) => (
                  <button
                    key={value}
                    type="button"
                    onClick={() => updateRow(i, { method: value })}
                    className={cn(
                      "min-h-9 cursor-pointer rounded-md border px-3 text-sm font-medium transition-colors",
                      row.method === value
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
                  placeholder="Amount (PHP)"
                  aria-label={`Payment ${i + 1} amount`}
                  value={row.amountPesos}
                  onChange={(e) => updateRow(i, { amountPesos: e.target.value })}
                />
                <Button type="button" variant="outline" onClick={() => setExact(i)}>
                  Exact
                </Button>
                {rows.length > 1 && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    aria-label="Remove payment"
                    onClick={() => setRows((prev) => prev.filter((_, j) => j !== i))}
                  >
                    <Trash2 className="size-4" aria-hidden />
                  </Button>
                )}
              </div>
              {row.method !== "cash" && (
                <Input
                  placeholder="Reference number"
                  aria-label={`Payment ${i + 1} reference`}
                  value={row.referenceNo}
                  onChange={(e) => updateRow(i, { referenceNo: e.target.value })}
                />
              )}
            </div>
          ))}

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setRows((prev) => [...prev, { method: "gcash", amountPesos: "", referenceNo: "" }])}
          >
            <Plus className="size-4" aria-hidden />
            Split payment
          </Button>

          <div className="space-y-1 rounded-lg bg-muted p-3 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Paid</span>
              <span className="font-medium tabular-nums">{formatCentavos(paidCentavos)}</span>
            </div>
            {remaining > 0 ? (
              <div className="flex justify-between text-destructive">
                <span>Remaining</span>
                <span className="font-semibold tabular-nums">{formatCentavos(remaining)}</span>
              </div>
            ) : (
              <div className="flex justify-between text-emerald-700 dark:text-emerald-400">
                <span>Change</span>
                <span className="font-semibold tabular-nums">{formatCentavos(change)}</span>
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button
            className="min-h-12 w-full text-base font-semibold"
            disabled={isPaying || paidCentavos < totalCentavos}
            onClick={onSubmit}
          >
            {isPaying && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Complete sale
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
