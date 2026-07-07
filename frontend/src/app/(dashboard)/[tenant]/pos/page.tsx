"use client";

import { useMemo, useState } from "react";
import { Banknote, ImageOff, Search, ShoppingCart } from "lucide-react";
import { toast } from "sonner";

import { useCategories, useProducts } from "@/hooks/use-catalog";
import { useCreateOrder, usePayOrder, useCurrentDrawer } from "@/hooks/use-orders";
import { useLoyaltySettings, type Customer } from "@/hooks/use-customers";
import { formatCentavos } from "@/lib/currency";
import {
  addLine,
  cartTotal,
  needsOptions,
  buildCartLine,
  toOrderItems,
  type CartLine,
} from "@/lib/pos-cart";
import { cn } from "@/lib/utils";
import type { Product } from "@/types/catalog";
import type { Order, OrderType, PaymentLineInput } from "@/types/order";
import { applyPromo } from "@/types/promo";
import { CartPanel } from "@/components/pos/cart-panel";
import { CustomerDialog } from "@/components/pos/customer-dialog";
import { DrawerDialog } from "@/components/pos/drawer-dialog";
import { HeldOrdersSheet } from "@/components/pos/held-orders-sheet";
import { PaymentDialog } from "@/components/pos/payment-dialog";
import { ProductOptionsDialog } from "@/components/pos/product-options-dialog";
import { PromoDialog, type AppliedPromo } from "@/components/pos/promo-dialog";
import { ReceiptDialog } from "@/components/pos/receipt-dialog";
import { SplitBillDialog } from "@/components/pos/split-bill-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";

const ALL = "all";

export default function POSPage() {
  const [categoryId, setCategoryId] = useState(ALL);
  const [search, setSearch] = useState("");
  const [lines, setLines] = useState<CartLine[]>([]);
  const [orderType, setOrderType] = useState<OrderType>("dine_in");
  const [tableNumber, setTableNumber] = useState("");
  const [optionsProduct, setOptionsProduct] = useState<Product | null>(null);
  const [payingOrder, setPayingOrder] = useState<Order | null>(null); // resumed held order
  const [paymentOpen, setPaymentOpen] = useState(false);
  const [receiptOrderId, setReceiptOrderId] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [mobileCartOpen, setMobileCartOpen] = useState(false);
  const [promoOpen, setPromoOpen] = useState(false);
  const [promo, setPromo] = useState<AppliedPromo>({});
  const [splittingOrder, setSplittingOrder] = useState<Order | null>(null);
  const [customerOpen, setCustomerOpen] = useState(false);
  const [customer, setCustomer] = useState<Customer | null>(null);

  const { data: categories } = useCategories(true);
  const { data: productData, isLoading } = useProducts({
    categoryId: categoryId === ALL ? "" : categoryId,
    search,
    activeOnly: true,
    limit: 200,
  });
  const { data: drawerData } = useCurrentDrawer();
  const { data: loyaltySettings } = useLoyaltySettings();
  const createOrder = useCreateOrder();
  const payOrder = usePayOrder();

  // Redeemable value on this sale (centavos) for the payment dialog.
  const pointsAvailable =
    customer && loyaltySettings?.is_enabled
      ? customer.points_balance * loyaltySettings.redeem_value
      : 0;

  const products = useMemo(() => productData?.products ?? [], [productData]);
  const subtotal = cartTotal(lines);

  // Client-side promo preview; the server recomputes on order creation.
  const discountPreview = useMemo(() => {
    let discount = 0;
    if (promo.discount) {
      discount += applyPromo(
        promo.discount.type,
        promo.discount.percent_value,
        promo.discount.amount_value,
        subtotal,
      );
    }
    if (promo.couponAmount) {
      discount += Math.min(promo.couponAmount, subtotal - discount);
    }
    return discount;
  }, [promo, subtotal]);

  const promoLabel = [promo.discount?.name, promo.couponCode].filter(Boolean).join(" + ");
  const total = payingOrder ? payingOrder.total : Math.max(0, subtotal - discountPreview);
  const itemCount = lines.reduce((n, l) => n + l.qty, 0);
  const isBusy = createOrder.isPending || payOrder.isPending;

  const onProductTap = (product: Product) => {
    if (needsOptions(product)) {
      setOptionsProduct(product);
    } else {
      setLines((prev) => addLine(prev, buildCartLine(product, {})));
    }
  };

  const resetSale = () => {
    setLines([]);
    setTableNumber("");
    setPayingOrder(null);
    setMobileCartOpen(false);
    setPromo({});
    setCustomer(null);
  };

  const orderPayload = (hold: boolean) => ({
    order_type: orderType,
    table_number: tableNumber.trim(),
    notes: "",
    hold,
    discount_id: promo.discount?.id,
    coupon_code: promo.couponCode,
    customer_id: customer?.id,
    items: toOrderItems(lines),
  });

  const onCharge = () => {
    if (orderType === "dine_in" && !tableNumber.trim() && !payingOrder) {
      toast.error("Enter the table number for dine-in orders");
      return;
    }
    setPaymentOpen(true);
  };

  const onHold = () => {
    createOrder.mutate(orderPayload(true), {
      onSuccess: (order) => {
        toast.success(`Order #${order.order_number} held`);
        resetSale();
      },
    });
  };

  const onSplit = () => {
    if (orderType === "dine_in" && !tableNumber.trim()) {
      toast.error("Enter the table number for dine-in orders");
      return;
    }
    // The order is created first; splits are attached in the dialog.
    createOrder.mutate(orderPayload(false), {
      onSuccess: (order) => {
        setSplittingOrder(order);
        setMobileCartOpen(false);
      },
    });
  };

  const settle = (orderId: string, payments: PaymentLineInput[]) => {
    payOrder.mutate(
      { orderId, payments },
      {
        onSuccess: (completed) => {
          setPaymentOpen(false);
          resetSale();
          setReceiptOrderId(completed.id);
        },
      },
    );
  };

  const onConfirmPayment = (payments: PaymentLineInput[]) => {
    if (payingOrder) {
      settle(payingOrder.id, payments);
      return;
    }
    createOrder.mutate(orderPayload(false), {
      onSuccess: (order) => settle(order.id, payments),
    });
  };

  const cartPanel = (
    <CartPanel
      lines={lines}
      onLinesChange={setLines}
      orderType={orderType}
      onOrderTypeChange={setOrderType}
      tableNumber={tableNumber}
      onTableNumberChange={setTableNumber}
      discountPreview={discountPreview}
      promoLabel={promoLabel}
      customerLabel={
        customer ? `${customer.full_name} · ${customer.points_balance} pts` : ""
      }
      onOpenCustomer={() => setCustomerOpen(true)}
      onOpenPromo={() => setPromoOpen(true)}
      onSplit={onSplit}
      onCharge={onCharge}
      onHold={onHold}
      isBusy={isBusy}
    />
  );

  return (
    <div className="flex h-[calc(100dvh-3.5rem-2rem)] flex-col gap-3 sm:h-[calc(100dvh-3.5rem-3rem)] lg:flex-row">
      {/* ---- left: catalog ---- */}
      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
            <Input
              placeholder="Search menu…"
              className="min-h-11 pl-9"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <HeldOrdersSheet
            onResume={(order) => {
              setPayingOrder(order);
              setPaymentOpen(true);
            }}
          />
          <Button
            variant={drawerData?.session ? "outline" : "default"}
            className="min-h-11"
            onClick={() => setDrawerOpen(true)}
          >
            <Banknote className="size-4" aria-hidden />
            <span className="hidden sm:inline">
              {drawerData?.session ? "Drawer" : "Open drawer"}
            </span>
          </Button>
        </div>

        {/* category chips */}
        <div className="flex gap-1.5 overflow-x-auto pb-1">
          <button
            type="button"
            onClick={() => setCategoryId(ALL)}
            className={cn(
              "min-h-10 shrink-0 cursor-pointer rounded-full border px-4 text-sm font-medium transition-colors",
              categoryId === ALL
                ? "border-primary bg-primary text-primary-foreground"
                : "hover:bg-accent/10",
            )}
          >
            All
          </button>
          {(categories ?? []).map((c) => (
            <button
              key={c.id}
              type="button"
              onClick={() => setCategoryId(c.id)}
              className={cn(
                "min-h-10 shrink-0 cursor-pointer rounded-full border px-4 text-sm font-medium transition-colors",
                categoryId === c.id
                  ? "border-primary bg-primary text-primary-foreground"
                  : "hover:bg-accent/10",
              )}
            >
              {c.name}
            </button>
          ))}
        </div>

        {/* product grid */}
        <div className="grid flex-1 auto-rows-min grid-cols-2 gap-2 overflow-y-auto pb-24 sm:grid-cols-3 xl:grid-cols-4 lg:pb-2">
          {isLoading &&
            Array.from({ length: 8 }, (_, i) => <Skeleton key={i} className="h-32 w-full" />)}

          {products.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => onProductTap(p)}
              className="flex min-h-32 cursor-pointer flex-col overflow-hidden rounded-xl border bg-card text-left shadow-sm transition-all hover:border-primary hover:shadow-md active:scale-[0.98]"
            >
              <div className="flex h-16 items-center justify-center bg-muted">
                {p.thumb_url ? (
                  // eslint-disable-next-line @next/next/no-img-element -- MinIO-served
                  <img src={p.thumb_url} alt="" className="size-full object-cover" />
                ) : (
                  <ImageOff className="size-5 text-muted-foreground/40" aria-hidden />
                )}
              </div>
              <div className="flex flex-1 flex-col justify-between gap-1 p-2.5">
                <p className="line-clamp-2 text-sm font-medium leading-tight">{p.name}</p>
                <div className="flex items-center justify-between">
                  <p className="text-sm font-bold tabular-nums text-primary">
                    {formatCentavos(p.base_price)}
                  </p>
                  {needsOptions(p) && (
                    <Badge variant="secondary" className="text-[10px]">
                      options
                    </Badge>
                  )}
                </div>
              </div>
            </button>
          ))}

          {!isLoading && products.length === 0 && (
            <p className="col-span-full py-16 text-center text-sm text-muted-foreground">
              No products match.
            </p>
          )}
        </div>
      </div>

      {/* ---- right: cart (desktop/tablet) ---- */}
      <aside className="hidden w-80 shrink-0 overflow-hidden rounded-xl border bg-card lg:block">
        {cartPanel}
      </aside>

      {/* ---- mobile cart bar ---- */}
      <div className="fixed inset-x-4 bottom-4 z-40 lg:hidden">
        <Button
          className="min-h-14 w-full justify-between text-base font-semibold shadow-lg"
          onClick={() => setMobileCartOpen(true)}
          disabled={itemCount === 0}
        >
          <span className="flex items-center gap-2">
            <ShoppingCart className="size-5" aria-hidden />
            {itemCount} {itemCount === 1 ? "item" : "items"}
          </span>
          <span className="tabular-nums">{formatCentavos(cartTotal(lines))}</span>
        </Button>
      </div>
      <Sheet open={mobileCartOpen} onOpenChange={setMobileCartOpen}>
        <SheetContent side="bottom" className="h-[85dvh] p-0">
          <SheetHeader className="border-b px-4 py-3">
            <SheetTitle>Current order</SheetTitle>
          </SheetHeader>
          <div className="h-[calc(85dvh-3.5rem)]">{cartPanel}</div>
        </SheetContent>
      </Sheet>

      {/* ---- dialogs ---- */}
      <ProductOptionsDialog
        product={optionsProduct}
        onClose={() => setOptionsProduct(null)}
        onAdd={(line) => setLines((prev) => addLine(prev, line))}
      />
      <PaymentDialog
        open={paymentOpen}
        onOpenChange={(open) => {
          setPaymentOpen(open);
          if (!open) setPayingOrder(null);
        }}
        totalCentavos={total}
        isPaying={isBusy}
        onConfirm={onConfirmPayment}
        pointsAvailableCentavos={payingOrder ? 0 : pointsAvailable}
        pointsCustomerName={customer?.full_name}
      />
      <CustomerDialog
        open={customerOpen}
        onOpenChange={setCustomerOpen}
        selected={customer}
        onSelect={setCustomer}
      />
      <ReceiptDialog orderId={receiptOrderId} onClose={() => setReceiptOrderId(null)} />
      <DrawerDialog open={drawerOpen} onOpenChange={setDrawerOpen} />
      <PromoDialog
        open={promoOpen}
        onOpenChange={setPromoOpen}
        subtotal={subtotal}
        applied={promo}
        onApply={setPromo}
      />
      <SplitBillDialog
        order={splittingOrder}
        onClose={() => setSplittingOrder(null)}
        onCompleted={(orderId) => {
          setSplittingOrder(null);
          resetSale();
          setReceiptOrderId(orderId);
        }}
      />
    </div>
  );
}
