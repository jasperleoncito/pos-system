"use client";

import { PauseCircle } from "lucide-react";

import { formatCentavos } from "@/lib/currency";
import { useOrders } from "@/hooks/use-orders";
import type { Order } from "@/types/order";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

interface HeldOrdersSheetProps {
  onResume: (order: Order) => void;
}

export function HeldOrdersSheet({ onResume }: HeldOrdersSheetProps) {
  const { data, isLoading } = useOrders({ status: "held", limit: 50 });
  const held = data?.orders ?? [];

  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button variant="outline" className="relative min-h-11">
          <PauseCircle className="size-4" aria-hidden />
          Held
          {held.length > 0 && (
            <Badge className="absolute -right-2 -top-2 size-5 justify-center rounded-full p-0">
              {held.length}
            </Badge>
          )}
        </Button>
      </SheetTrigger>
      <SheetContent side="right" className="w-full sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Held orders</SheetTitle>
        </SheetHeader>
        <div className="space-y-2 overflow-y-auto px-4 pb-4">
          {isLoading &&
            Array.from({ length: 2 }, (_, i) => <Skeleton key={i} className="h-20 w-full" />)}

          {held.map((order) => (
            <button
              key={order.id}
              type="button"
              onClick={() => onResume(order)}
              className="w-full cursor-pointer rounded-lg border p-3 text-left transition-colors hover:bg-accent/10"
            >
              <div className="flex items-center justify-between">
                <p className="text-sm font-semibold">#{order.order_number}</p>
                <p className="text-sm font-semibold tabular-nums">{formatCentavos(order.total)}</p>
              </div>
              <p className="text-xs text-muted-foreground">
                {(order.items ?? []).map((i) => `${i.qty}× ${i.name}`).join(", ")}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {new Date(order.created_at).toLocaleTimeString()} ·{" "}
                {order.order_type.replace("_", " ")}
                {order.table_number && ` · Table ${order.table_number}`}
              </p>
            </button>
          ))}

          {!isLoading && held.length === 0 && (
            <p className="py-10 text-center text-sm text-muted-foreground">No held orders.</p>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
