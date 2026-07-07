import type { Product } from "@/types/catalog";
import type { OrderItemInput } from "@/types/order";

/**
 * Client-side cart line. Displayed totals are previews; the server
 * re-prices everything from the catalog on order creation.
 */
export interface CartLine {
  key: string;
  productId: string;
  name: string;
  variantId?: string;
  variantName?: string;
  modifierIds: string[];
  modifierNames: string[];
  unitPrice: number; // centavos preview
  qty: number;
  notes?: string;
}

/** Identical selections merge into one line. */
export function lineKey(productId: string, variantId: string | undefined, modifierIds: string[], notes?: string): string {
  return [productId, variantId ?? "", [...modifierIds].sort().join(","), notes ?? ""].join("|");
}

export function buildCartLine(
  product: Product,
  opts: { variantId?: string; modifierIds?: string[]; qty?: number; notes?: string },
): CartLine {
  const modifierIds = opts.modifierIds ?? [];
  let unitPrice = product.base_price;
  let variantName: string | undefined;

  if (opts.variantId) {
    const variant = (product.variants ?? []).find((v) => v.id === opts.variantId);
    if (variant) {
      unitPrice += variant.price_delta;
      variantName = variant.name;
    }
  }

  const modifierNames: string[] = [];
  for (const group of product.modifier_groups ?? []) {
    for (const mod of group.modifiers ?? []) {
      if (mod.id && modifierIds.includes(mod.id)) {
        unitPrice += mod.price_delta;
        modifierNames.push(mod.name);
      }
    }
  }

  return {
    key: lineKey(product.id, opts.variantId, modifierIds, opts.notes),
    productId: product.id,
    name: product.name,
    variantId: opts.variantId,
    variantName,
    modifierIds,
    modifierNames,
    unitPrice,
    qty: opts.qty ?? 1,
    notes: opts.notes,
  };
}

/** Adds a line, merging with an identical existing selection. */
export function addLine(lines: CartLine[], line: CartLine): CartLine[] {
  const existing = lines.findIndex((l) => l.key === line.key);
  if (existing >= 0) {
    return lines.map((l, i) => (i === existing ? { ...l, qty: l.qty + line.qty } : l));
  }
  return [...lines, line];
}

export function setLineQty(lines: CartLine[], key: string, qty: number): CartLine[] {
  if (qty <= 0) {
    return lines.filter((l) => l.key !== key);
  }
  return lines.map((l) => (l.key === key ? { ...l, qty } : l));
}

export function cartTotal(lines: CartLine[]): number {
  return lines.reduce((sum, l) => sum + l.unitPrice * l.qty, 0);
}

export function toOrderItems(lines: CartLine[]): OrderItemInput[] {
  return lines.map((l) => ({
    product_id: l.productId,
    variant_id: l.variantId || undefined,
    qty: l.qty,
    modifier_ids: l.modifierIds,
    notes: l.notes || undefined,
  }));
}

/** True when the product needs the options dialog before adding. */
export function needsOptions(product: Product): boolean {
  return (product.variants?.length ?? 0) > 0 || (product.modifier_groups?.length ?? 0) > 0;
}
