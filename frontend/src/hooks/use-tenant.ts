"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import type { Tenant, TenantSettings } from "@/types/tenant";

export interface UpdateSettingsInput {
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
}

export function useTenantSettings() {
  const { auth } = useAuth();
  return useQuery({
    queryKey: ["tenant", "settings", auth?.activeTenant?.tenant_id],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<TenantSettings>>("/tenant/settings");
      return res.data.data;
    },
    enabled: Boolean(auth?.activeTenant),
    staleTime: 5 * 60_000,
  });
}

export function useUpdateTenantSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateSettingsInput) => {
      const res = await api.put<ApiEnvelope<TenantSettings>>("/tenant/settings", input);
      return res.data.data;
    },
    onSuccess: (data) => {
      queryClient.setQueryData(["tenant", "settings", data.tenant_id], data);
      toast.success("Branding updated");
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useUploadLogo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (file: File) => {
      const form = new FormData();
      form.append("logo", file);
      const res = await api.post<ApiEnvelope<TenantSettings>>("/tenant/logo", form, {
        headers: { "Content-Type": "multipart/form-data" },
        timeout: 120_000,
      });
      return res.data.data;
    },
    onSuccess: (data) => {
      queryClient.setQueryData(["tenant", "settings", data.tenant_id], data);
      toast.success("Logo updated");
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useAdminTenants(page: number) {
  return useQuery({
    queryKey: ["admin", "tenants", page],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Tenant[]>>("/admin/tenants", {
        params: { page, limit: 20 },
      });
      return { tenants: res.data.data ?? [], meta: res.data.meta };
    },
  });
}

export interface PlatformStats {
  tenants_total: number;
  tenants_active: number;
  users_total: number;
  orders_30d: number;
  gmv_30d: number;
}

export function useAdminStats() {
  return useQuery({
    queryKey: ["admin", "stats"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<PlatformStats>>("/admin/stats");
      return res.data.data;
    },
  });
}

export interface AdminCreateTenantInput {
  business_name: string;
  business_slug: string;
  owner_full_name: string;
  owner_email: string;
}

export function useAdminCreateTenant() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: AdminCreateTenantInput) => {
      const res = await api.post<ApiEnvelope<Tenant>>("/admin/tenants", input);
      return res.data.data;
    },
    onSuccess: (t) => {
      toast.success(`${t.name} created — the owner has been emailed`);
      queryClient.invalidateQueries({ queryKey: ["admin", "tenants"] });
      queryClient.invalidateQueries({ queryKey: ["admin", "stats"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useSetTenantPlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ tenantId, plan }: { tenantId: string; plan: string }) => {
      const res = await api.patch<ApiEnvelope<Tenant>>(`/admin/tenants/${tenantId}/plan`, { plan });
      return res.data.data;
    },
    onSuccess: (t) => {
      toast.success(`${t.name} moved to the ${t.plan} plan`);
      queryClient.invalidateQueries({ queryKey: ["admin", "tenants"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useSetTenantStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ tenantId, status }: { tenantId: string; status: "active" | "suspended" }) => {
      const res = await api.patch<ApiEnvelope<Tenant>>(`/admin/tenants/${tenantId}/status`, { status });
      return res.data.data;
    },
    onSuccess: (t) => {
      toast.success(`${t.name} is now ${t.status}`);
      queryClient.invalidateQueries({ queryKey: ["admin", "tenants"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
