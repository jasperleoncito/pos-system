import type { Role } from "@/types/auth";

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

export interface TeamMember {
  id: string;
  tenant_id: string;
  user_id: string;
  role: Role;
  full_name: string;
  email: string;
  user_status: "active" | "disabled";
  email_verified_at: string | null;
  joined_at: string;
  is_owner: boolean;
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
