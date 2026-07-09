export type BillingPlan = "monthly" | "yearly";
export type SubscriptionStatus = "pending" | "active" | "inactive";

export interface PlatformPlans {
  monthly_price: number; // centavos
  yearly_price: number; // centavos
  updated_at: string;
}

export interface Subscription {
  id: string;
  tenant_id: string;
  plan: BillingPlan;
  status: SubscriptionStatus;
  current_period_start: string;
  current_period_end: string;
  due_notice_sent_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface SubscriptionPayment {
  id: string;
  tenant_id: string;
  subscription_id: string;
  plan: BillingPlan;
  amount: number; // centavos
  status: "pending" | "paid" | "expired";
  method: "xendit" | "manual";
  external_id: string;
  xendit_invoice_id: string;
  xendit_invoice_url: string;
  payment_channel: string;
  paid_at?: string | null;
  note: string;
  created_at: string;
}

export interface CheckoutResult {
  payment_id: string;
  plan: BillingPlan;
  amount: number;
  invoice_url: string;
}

export interface AdminSubscription extends Subscription {
  tenant_name: string;
  tenant_slug: string;
  tenant_status: "active" | "suspended";
  owner_name: string;
  owner_email: string;
  last_paid_at?: string | null;
  last_paid_amount?: number | null;
}

export interface OwnedBusiness {
  tenant_id: string;
  name: string;
  slug: string;
  plan: string;
  sub_status: string;
  period_end: string;
}

export interface PlatformOwner {
  user_id: string;
  full_name: string;
  email: string;
  user_status: "active" | "disabled";
  created_at: string;
  businesses: OwnedBusiness[];
}

export interface BillingStats {
  subs_active: number;
  subs_pending: number;
  subs_inactive: number;
  collected_this_month: number;
  collected_30d: number;
}
