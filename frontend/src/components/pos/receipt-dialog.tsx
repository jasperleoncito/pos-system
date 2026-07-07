"use client";

import { Printer } from "lucide-react";

import { formatCentavos } from "@/lib/currency";
import { useReceipt } from "@/hooks/use-orders";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

const METHOD_LABELS: Record<string, string> = {
  cash: "Cash",
  gcash: "GCash",
  card: "Card",
  maya: "Maya",
  bank_transfer: "Bank Transfer",
};

interface ReceiptDialogProps {
  orderId: string | null;
  onClose: () => void;
}

/**
 * Receipt preview + print. The #receipt-print block is the only thing
 * visible during window.print() (80mm thermal layout via print CSS).
 */
export function ReceiptDialog({ orderId, onClose }: ReceiptDialogProps) {
  const { data: receipt, isLoading } = useReceipt(orderId);

  if (!orderId) return null;

  const order = receipt?.order;
  const business = receipt?.business;

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Receipt</DialogTitle>
        </DialogHeader>

        {isLoading || !order || !business ? (
          <Skeleton className="h-72 w-full" />
        ) : (
          <div id="receipt-print" className="mx-auto w-full max-w-72 bg-white p-2 font-mono text-xs text-black">
            <div className="space-y-0.5 text-center">
              {business.logo_url && (
                // eslint-disable-next-line @next/next/no-img-element -- receipt printer asset
                <img src={business.logo_url} alt="" className="mx-auto mb-1 size-12 object-contain" />
              )}
              <p className="text-sm font-bold">{business.name}</p>
              {business.receipt_header && <p>{business.receipt_header}</p>}
              {business.address && <p>{business.address}</p>}
              {business.contact_number && <p>{business.contact_number}</p>}
              {business.tax_id && (
                <p>
                  {business.tax_label || "TIN"}: {business.tax_id}
                </p>
              )}
            </div>

            <Separator className="my-2 bg-black/20" />

            <div className="space-y-0.5">
              <div className="flex justify-between">
                <span>Order #{order.order_number}</span>
                <span className="uppercase">{order.order_type.replace("_", " ")}</span>
              </div>
              {order.table_number && <p>Table {order.table_number}</p>}
              <p>{new Date(order.completed_at ?? order.created_at).toLocaleString("en-PH")}</p>
              {order.cashier_name && <p>Cashier: {order.cashier_name}</p>}
            </div>

            <Separator className="my-2 bg-black/20" />

            <div className="space-y-1">
              {(order.items ?? []).map((item) => (
                <div key={item.id}>
                  <div className="flex justify-between gap-2">
                    <span>
                      {item.qty} × {item.name}
                      {item.variant_name ? ` (${item.variant_name})` : ""}
                    </span>
                    <span className="tabular-nums">{formatCentavos(item.line_total)}</span>
                  </div>
                  {(item.modifiers ?? []).map((mod) => (
                    <p key={mod.id} className="pl-4 text-[10px]">
                      + {mod.name}
                      {mod.price_delta > 0 ? ` (${formatCentavos(mod.price_delta)})` : ""}
                    </p>
                  ))}
                </div>
              ))}
            </div>

            <Separator className="my-2 bg-black/20" />

            <div className="space-y-0.5">
              <div className="flex justify-between text-sm font-bold">
                <span>TOTAL</span>
                <span className="tabular-nums">{formatCentavos(order.total)}</span>
              </div>
              {order.tax_total > 0 && (
                <div className="flex justify-between text-[10px]">
                  <span>VAT included</span>
                  <span className="tabular-nums">{formatCentavos(order.tax_total)}</span>
                </div>
              )}
              {(order.payments ?? []).map((p) => (
                <div key={p.id} className="flex justify-between">
                  <span>{METHOD_LABELS[p.method] ?? p.method}</span>
                  <span className="tabular-nums">{formatCentavos(p.amount)}</span>
                </div>
              ))}
              {order.change > 0 && (
                <div className="flex justify-between">
                  <span>Change</span>
                  <span className="tabular-nums">{formatCentavos(order.change)}</span>
                </div>
              )}
            </div>

            {business.receipt_footer && (
              <>
                <Separator className="my-2 bg-black/20" />
                <p className="text-center">{business.receipt_footer}</p>
              </>
            )}
          </div>
        )}

        <DialogFooter className="gap-2">
          <Button variant="outline" onClick={onClose}>
            New sale
          </Button>
          <Button onClick={() => window.print()} disabled={isLoading}>
            <Printer className="size-4" aria-hidden />
            Print
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
