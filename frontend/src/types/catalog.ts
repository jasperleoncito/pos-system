export interface Category {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  sort_order: number;
  image_key: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Variant {
  id?: string;
  product_id?: string;
  name: string;
  price_delta: number;
  sku?: string;
  sort_order?: number;
}

export interface Modifier {
  id?: string;
  group_id?: string;
  name: string;
  price_delta: number;
  is_active?: boolean;
  sort_order?: number;
}

export interface ModifierGroup {
  id: string;
  tenant_id?: string;
  name: string;
  min_select: number;
  max_select: number;
  is_required: boolean;
  sort_order: number;
  modifiers?: Modifier[];
}

export interface Tax {
  id: string;
  tenant_id?: string;
  name: string;
  rate_percent: number;
  is_inclusive: boolean;
  is_default: boolean;
  is_active: boolean;
}

export interface Product {
  id: string;
  tenant_id: string;
  category_id: string;
  tax_id: string | null;
  name: string;
  description: string;
  sku: string;
  base_price: number;
  cost_price: number;
  image_key: string;
  thumb_key: string;
  is_active: boolean;
  track_inventory: boolean;
  sort_order: number;
  variants?: Variant[];
  modifier_groups?: ModifierGroup[];
  category_name?: string;
  image_url: string;
  thumb_url: string;
}

export interface ProductInput {
  category_id: string;
  tax_id: string | null;
  name: string;
  description: string;
  sku: string;
  base_price: number;
  cost_price: number;
  is_active: boolean;
  track_inventory: boolean;
  sort_order: number;
  variants: { name: string; price_delta: number; sku?: string }[];
  modifier_groups: string[];
}
