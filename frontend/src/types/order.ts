export type OrderType = "dine_in" | "takeout" | "delivery";
export type PaymentMethod = "cash" | "gcash" | "card" | "maya" | "bank_transfer";

export interface OrderItemModifier {
  id: string;
  modifier_id: string;
  group_name: string;
  name: string;
  price_delta: number;
}

export interface OrderItem {
  id: string;
  product_id: string;
  variant_id: string | null;
  name: string;
  variant_name: string;
  unit_price: number;
  qty: number;
  discount_amount: number;
  tax_amount: number;
  line_total: number;
  notes: string;
  status: string;
  modifiers?: OrderItemModifier[];
}

export interface Payment {
  id: string;
  order_id: string;
  method: PaymentMethod;
  amount: number;
  reference_no: string;
  status: string;
  created_at: string;
}

export interface OrderSplit {
  id: string;
  order_id: string;
  split_number: number;
  amount: number;
  status: "pending" | "paid";
  created_at: string;
}

export interface OrderRefund {
  id: string;
  order_id: string;
  refund_number: number;
  reason: string;
  amount: number;
  created_at: string;
}

export interface Order {
  id: string;
  order_number: number;
  order_type: OrderType;
  table_number: string;
  cashier_user_id: string;
  cashier_name?: string;
  status: string;
  kitchen_status: string;
  priority: boolean;
  subtotal: number;
  discount_total: number;
  tax_total: number;
  total: number;
  tendered: number;
  change: number;
  notes: string;
  void_reason?: string;
  completed_at: string | null;
  created_at: string;
  items?: OrderItem[];
  payments?: Payment[];
  splits?: OrderSplit[];
  refunds?: OrderRefund[];
}

export interface ReceiptBusiness {
  name: string;
  logo_url: string;
  receipt_header: string;
  receipt_footer: string;
  address: string;
  contact_number: string;
  tax_label: string;
  tax_id: string;
}

export interface Receipt {
  order: Order;
  business: ReceiptBusiness;
}

export interface DrawerSession {
  id: string;
  opened_by: string;
  opening_float: number;
  expected_cash: number;
  counted_cash: number | null;
  variance: number | null;
  status: "open" | "closed";
  opened_at: string;
  closed_at: string | null;
}

export interface CashMovement {
  id: string;
  type: string;
  amount: number;
  order_id: string | null;
  reason: string;
  created_at: string;
}

// ---- request payloads ----

export interface OrderItemInput {
  product_id: string;
  variant_id?: string;
  qty: number;
  modifier_ids: string[];
  notes?: string;
}

export interface CreateOrderInput {
  order_type: OrderType;
  table_number: string;
  notes: string;
  hold: boolean;
  items: OrderItemInput[];
}

export interface PaymentLineInput {
  method: PaymentMethod;
  amount: number;
  reference_no?: string;
}
