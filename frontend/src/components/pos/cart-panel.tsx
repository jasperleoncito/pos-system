"use client";

import { Minus, PauseCircle, Plus, ShoppingCart, Trash2 } from "lucide-react";

import { formatCentavos } from "@/lib/currency";
import { cartTotal, setLineQty, type CartLine } from "@/lib/pos-cart";
import { cn } from "@/lib/utils";
import type { OrderType } from "@/types/order";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";

const ORDER_TYPES: { value: OrderType; label: string }[] = [
  { value: "dine_in", label: "Dine-in" },
  { value: "takeout", label: "Take out" },
  { value: "delivery", label: "Delivery" },
];

interface CartPanelProps {
  lines: CartLine[];
  onLinesChange: (lines: CartLine[]) => void;
  orderType: OrderType;
  onOrderTypeChange: (t: OrderType) => void;
  tableNumber: string;
  onTableNumberChange: (t: string) => void;
  onCharge: () => void;
  onHold: () => void;
  isBusy: boolean;
}

export function CartPanel({
  lines,
  onLinesChange,
  orderType,
  onOrderTypeChange,
  tableNumber,
  onTableNumberChange,
  onCharge,
  onHold,
  isBusy,
}: CartPanelProps) {
  const total = cartTotal(lines);

  return (
    <div className="flex h-full flex-col">
      {/* Order type */}
      <div className="grid grid-cols-3 gap-1.5 p-3">
        {ORDER_TYPES.map(({ value, label }) => (
          <button
            key={value}
            type="button"
            onClick={() => onOrderTypeChange(value)}
            className={cn(
              "min-h-10 cursor-pointer rounded-lg border text-sm font-medium transition-colors",
              orderType === value
                ? "border-primary bg-primary text-primary-foreground"
                : "hover:bg-accent/10",
            )}
          >
            {label}
          </button>
        ))}
      </div>
      {orderType === "dine_in" && (
        <div className="px-3 pb-2">
          <Input
            placeholder="Table number"
            value={tableNumber}
            onChange={(e) => onTableNumberChange(e.target.value)}
            aria-label="Table number"
          />
        </div>
      )}

      <Separator />

      {/* Lines */}
      <div className="flex-1 overflow-y-auto p-3">
        {lines.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
            <ShoppingCart className="size-8" aria-hidden />
            <p className="text-sm">Tap products to start an order</p>
          </div>
        ) : (
          <ul className="space-y-3">
            {lines.map((line) => (
              <li key={line.key} className="rounded-lg border p-2.5">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="text-sm font-medium leading-tight">
                      {line.name}
                      {line.variantName && (
                        <span className="text-muted-foreground"> · {line.variantName}</span>
                      )}
                    </p>
                    {line.modifierNames.length > 0 && (
                      <p className="text-xs text-muted-foreground">
                        {line.modifierNames.join(", ")}
                      </p>
                    )}
                    {line.notes && (
                      <p className="text-xs italic text-muted-foreground">“{line.notes}”</p>
                    )}
                  </div>
                  <p className="shrink-0 text-sm font-semibold tabular-nums">
                    {formatCentavos(line.unitPrice * line.qty)}
                  </p>
                </div>
                <div className="mt-2 flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="icon"
                    className="size-8"
                    aria-label={`Decrease ${line.name}`}
                    onClick={() => onLinesChange(setLineQty(lines, line.key, line.qty - 1))}
                  >
                    <Minus className="size-3.5" aria-hidden />
                  </Button>
                  <span className="w-8 text-center text-sm font-semibold tabular-nums">
                    {line.qty}
                  </span>
                  <Button
                    variant="outline"
                    size="icon"
                    className="size-8"
                    aria-label={`Increase ${line.name}`}
                    onClick={() => onLinesChange(setLineQty(lines, line.key, line.qty + 1))}
                  >
                    <Plus className="size-3.5" aria-hidden />
                  </Button>
                  <div className="flex-1" />
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-8"
                    aria-label={`Remove ${line.name}`}
                    onClick={() => onLinesChange(setLineQty(lines, line.key, 0))}
                  >
                    <Trash2 className="size-3.5 text-destructive" aria-hidden />
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Totals + actions */}
      <div className="space-y-3 border-t p-3">
        <div className="flex items-center justify-between text-lg font-bold">
          <span>Total</span>
          <span className="tabular-nums">{formatCentavos(total)}</span>
        </div>
        <div className="grid grid-cols-[auto_1fr] gap-2">
          <Button
            variant="outline"
            className="min-h-12"
            disabled={lines.length === 0 || isBusy}
            onClick={onHold}
          >
            <PauseCircle className="size-4" aria-hidden />
            Hold
          </Button>
          <Button
            className="min-h-12 text-base font-semibold"
            disabled={lines.length === 0 || isBusy}
            onClick={onCharge}
          >
            Charge {total > 0 ? formatCentavos(total) : ""}
          </Button>
        </div>
      </div>
    </div>
  );
}
