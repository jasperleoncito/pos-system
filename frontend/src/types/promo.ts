export type PromoType = "percent" | "fixed";

export interface Discount {
  id: string;
  name: string;
  type: PromoType;
  percent_value: number;
  amount_value: number;
  requires_approval: boolean;
  is_active: boolean;
  created_at: string;
}

export interface Coupon {
  id: string;
  code: string;
  discount_type: PromoType;
  percent_value: number;
  amount_value: number;
  min_order_amount: number;
  max_uses: number;
  uses_count: number;
  valid_from: string | null;
  valid_to: string | null;
  is_active: boolean;
  created_at: string;
}

/** Mirror of the backend promo.Apply, for cart previews only. */
export function applyPromo(
  type: PromoType,
  percentValue: number,
  amountValue: number,
  subtotal: number,
): number {
  let discount = 0;
  if (type === "percent") {
    discount = Math.floor((subtotal * percentValue) / 100 + 0.5);
  } else {
    discount = amountValue;
  }
  return Math.max(0, Math.min(discount, subtotal));
}
