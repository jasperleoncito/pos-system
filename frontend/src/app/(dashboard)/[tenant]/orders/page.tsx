"use client";

import { useState } from "react";

import { useOrders } from "@/hooks/use-orders";
import { formatCentavos } from "@/lib/currency";
import { cn } from "@/lib/utils";
import { OrderDetailDialog } from "@/components/orders/order-detail-dialog";
import { ReceiptDialog } from "@/components/pos/receipt-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const STATUS_FILTERS = [
  { value: "", label: "All" },
  { value: "completed", label: "Completed" },
  { value: "open", label: "Open" },
  { value: "held", label: "Held" },
  { value: "voided", label: "Voided" },
  { value: "refunded", label: "Refunded" },
  { value: "partially_refunded", label: "Partial refund" },
];

const STATUS_COLORS: Record<string, string> = {
  completed: "bg-emerald-600",
  open: "bg-sky-600",
  held: "bg-amber-600",
  voided: "bg-neutral-500",
  refunded: "bg-rose-600",
  partially_refunded: "bg-orange-600",
};

export default function OrdersPage() {
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [detailOrderId, setDetailOrderId] = useState<string | null>(null);
  const [receiptOrderId, setReceiptOrderId] = useState<string | null>(null);

  const { data, isLoading } = useOrders({ status, page, limit: 25 });
  const total = data?.meta?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / (data?.meta?.limit ?? 25)));

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Orders</h1>
        <p className="text-muted-foreground">Sales history — tap an order for details</p>
      </header>

      <div className="flex gap-1.5 overflow-x-auto pb-1">
        {STATUS_FILTERS.map((f) => (
          <button
            key={f.value}
            type="button"
            onClick={() => {
              setStatus(f.value);
              setPage(1);
            }}
            className={cn(
              "min-h-10 shrink-0 cursor-pointer rounded-full border px-4 text-sm font-medium transition-colors",
              status === f.value
                ? "border-primary bg-primary text-primary-foreground"
                : "hover:bg-accent/10",
            )}
          >
            {f.label}
          </button>
        ))}
      </div>

      <Card className="py-0">
        <CardContent className="overflow-x-auto p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>#</TableHead>
                <TableHead>Items</TableHead>
                <TableHead className="hidden sm:table-cell">Type</TableHead>
                <TableHead className="hidden sm:table-cell">When</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Total</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading &&
                Array.from({ length: 6 }, (_, i) => (
                  <TableRow key={i}>
                    <TableCell colSpan={6}>
                      <Skeleton className="h-9 w-full" />
                    </TableCell>
                  </TableRow>
                ))}

              {data?.orders.map((o) => (
                <TableRow
                  key={o.id}
                  className="cursor-pointer"
                  onClick={() => setDetailOrderId(o.id)}
                >
                  <TableCell className="font-semibold tabular-nums">#{o.order_number}</TableCell>
                  <TableCell className="max-w-56">
                    <p className="truncate text-muted-foreground">
                      {(o.items ?? []).map((i) => `${i.qty}× ${i.name}`).join(", ")}
                    </p>
                  </TableCell>
                  <TableCell className="hidden capitalize sm:table-cell">
                    {o.order_type.replace("_", " ")}
                    {o.table_number && ` · T${o.table_number}`}
                  </TableCell>
                  <TableCell className="hidden text-muted-foreground sm:table-cell">
                    {new Date(o.created_at).toLocaleString("en-PH", {
                      month: "short",
                      day: "numeric",
                      hour: "numeric",
                      minute: "2-digit",
                    })}
                  </TableCell>
                  <TableCell>
                    <Badge className={STATUS_COLORS[o.status]}>{o.status.replace("_", " ")}</Badge>
                  </TableCell>
                  <TableCell className="text-right font-medium tabular-nums">
                    {formatCentavos(o.total)}
                  </TableCell>
                </TableRow>
              ))}

              {data && data.orders.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="py-10 text-center text-muted-foreground">
                    No orders yet.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {pageCount > 1 && (
        <div className="flex items-center justify-between">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {page} of {pageCount} · {total} orders
          </span>
          <Button variant="outline" size="sm" disabled={page >= pageCount} onClick={() => setPage((p) => p + 1)}>
            Next
          </Button>
        </div>
      )}

      <OrderDetailDialog
        orderId={detailOrderId}
        onClose={() => setDetailOrderId(null)}
        onPrintReceipt={(id) => {
          setDetailOrderId(null);
          setReceiptOrderId(id);
        }}
      />
      <ReceiptDialog orderId={receiptOrderId} onClose={() => setReceiptOrderId(null)} />
    </div>
  );
}
