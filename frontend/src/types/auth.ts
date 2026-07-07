export type Role = "owner" | "manager" | "cashier" | "kitchen" | "employee";

export interface User {
  id: string;
  email: string;
  full_name: string;
  phone: string;
  avatar_key: string;
  is_super_admin: boolean;
  email_verified_at: string | null;
  status: string;
}

export interface Membership {
  id: string;
  tenant_id: string;
  user_id: string;
  role: Role;
  tenant_name?: string;
  tenant_slug?: string;
}

export interface AuthResult {
  user: User;
  memberships: Membership[];
  active_tenant?: Membership | null;
  access_token: string;
  refresh_token: string;
}

export interface DeviceSession {
  id: string;
  user_id: string;
  device_name: string;
  user_agent: string;
  ip: string;
  last_used_at: string;
  expires_at: string;
  created_at: string;
}
