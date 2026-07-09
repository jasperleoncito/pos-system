import { Minus, Plus, Search } from "lucide-react";

import { formatCentavos } from "@/lib/currency";

const CATEGORIES = ["All", "Rice Meals", "Silog", "Drinks", "Add-ons"];

const PRODUCTS = [
  { name: "Pork Sisig", price: 12000, tone: "bg-chart-1/15" },
  { name: "Chicken Adobo", price: 11000, tone: "bg-accent/20" },
  { name: "Beef Tapa", price: 13500, tone: "bg-chart-2/20" },
  { name: "Garlic Rice", price: 3000, tone: "bg-chart-3/15" },
  { name: "Iced Tea", price: 4500, tone: "bg-chart-4/15" },
  { name: "Leche Flan", price: 6000, tone: "bg-chart-5/15" },
];

const CART = [
  { name: "Pork Sisig", qty: 2, price: 12000 },
  { name: "Garlic Rice", qty: 2, price: 3000 },
  { name: "Iced Tea", qty: 1, price: 4500 },
];

/**
 * A stylized, non-interactive mock of the POS terminal built from the
 * app's real design vocabulary (product tiles + cart panel + ₱ totals).
 * Purely decorative — stands in for a screenshot on the landing hero.
 */
export function AppMockup() {
  const subtotal = CART.reduce((sum, i) => sum + i.qty * i.price, 0);
  const vat = Math.round((subtotal * 12) / 112);

  return (
    <div
      aria-hidden
      className="w-full overflow-hidden rounded-2xl border bg-card shadow-xl ring-1 ring-black/5"
    >
      {/* window chrome */}
      <div className="flex items-center gap-1.5 border-b bg-muted/40 px-4 py-3">
        <span className="size-2.5 rounded-full bg-chart-1/60" />
        <span className="size-2.5 rounded-full bg-accent/70" />
        <span className="size-2.5 rounded-full bg-chart-3/60" />
        <div className="ml-3 flex h-6 flex-1 items-center gap-2 rounded-md bg-background px-2 text-[11px] text-muted-foreground">
          <Search className="size-3" />
          Search menu…
        </div>
      </div>

      <div className="grid gap-3 p-3 sm:grid-cols-[1.6fr_1fr] sm:p-4">
        {/* menu side */}
        <div className="space-y-3">
          <div className="flex flex-wrap gap-1.5">
            {CATEGORIES.map((c, i) => (
              <span
                key={c}
                className={
                  i === 0
                    ? "rounded-full bg-primary px-2.5 py-1 text-[11px] font-medium text-primary-foreground"
                    : "rounded-full border px-2.5 py-1 text-[11px] text-muted-foreground"
                }
              >
                {c}
              </span>
            ))}
          </div>
          <div className="grid grid-cols-3 gap-2">
            {PRODUCTS.map((p) => (
              <div key={p.name} className="rounded-xl border p-2 transition-colors">
                <div className={`mb-2 aspect-square rounded-lg ${p.tone}`} />
                <p className="truncate text-[11px] font-medium leading-tight">{p.name}</p>
                <p className="text-[11px] font-bold text-primary tabular-nums">{formatCentavos(p.price)}</p>
              </div>
            ))}
          </div>
        </div>

        {/* cart side */}
        <div className="flex flex-col rounded-xl border bg-background p-3">
          <p className="mb-2 text-xs font-semibold">Current order</p>
          <div className="flex-1 space-y-2">
            {CART.map((i) => (
              <div key={i.name} className="flex items-center gap-2 text-[11px]">
                <div className="flex items-center gap-1 rounded-md border px-1 py-0.5">
                  <Minus className="size-2.5 text-muted-foreground" />
                  <span className="w-3 text-center font-medium tabular-nums">{i.qty}</span>
                  <Plus className="size-2.5 text-muted-foreground" />
                </div>
                <span className="flex-1 truncate">{i.name}</span>
                <span className="font-medium tabular-nums">{formatCentavos(i.qty * i.price)}</span>
              </div>
            ))}
          </div>
          <div className="mt-3 space-y-1 border-t pt-2 text-[11px]">
            <div className="flex justify-between text-muted-foreground">
              <span>VAT (12% incl.)</span>
              <span className="tabular-nums">{formatCentavos(vat)}</span>
            </div>
            <div className="flex justify-between text-sm font-bold">
              <span>Total</span>
              <span className="tabular-nums text-primary">{formatCentavos(subtotal)}</span>
            </div>
          </div>
          <div className="mt-3 rounded-lg bg-primary py-2 text-center text-[11px] font-semibold text-primary-foreground">
            Charge {formatCentavos(subtotal)}
          </div>
        </div>
      </div>
    </div>
  );
}
