export interface TenantSettings {
  id: string;
  tenant_id: string;
  logo_key: string;
  logo_thumb_key: string;
  favicon_keys: Record<string, string>;
  primary_color: string;
  secondary_color: string;
  accent_color: string;
  receipt_header: string;
  receipt_footer: string;
  contact_number: string;
  facebook: string;
  website: string;
  address: string;
  tax_label: string;
  tax_id: string;
  updated_at: string;
  logo_url: string;
  logo_thumb_url: string;
  favicon_urls: Record<string, string>;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  owner_user_id: string;
  status: "active" | "suspended";
  currency: string;
  timezone: string;
  plan: "free" | "standard" | "premium";
  created_at: string;
  updated_at: string;
}
