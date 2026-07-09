"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import type {
  AdminSubscription,
  BillingPlan,
  BillingStats,
  CheckoutResult,
  PlatformOwner,
  PlatformPlans,
  Subscription,
  SubscriptionPayment,
} from "@/types/billing";

/** Public price sheet — used by the register page before any login. */
export function usePlans() {
  return useQuery({
    queryKey: ["billing", "plans"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<PlatformPlans>>("/billing/plans");
      return res.data.data;
    },
    staleTime: 5 * 60_000,
  });
}

/**
 * The active tenant's subscription (any member). By default it re-polls
 * every 30s while not active, so paying in another tab lifts the
 * blocked screen / modal without a re-login.
 */
export function useSubscription(refetchInterval?: number) {
  const { auth } = useAuth();
  return useQuery({
    queryKey: ["billing", "subscription", auth?.activeTenant?.tenant_id],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Subscription>>("/billing/subscription");
      return res.data.data;
    },
    enabled: Boolean(auth?.activeTenant),
    refetchInterval:
      refetchInterval ??
      ((query) => (query.state.data && query.state.data.status !== "active" ? 30_000 : false)),
  });
}

/** Creates (or reuses) a Xendit invoice; caller redirects to invoice_url. */
export function useCheckout() {
  return useMutation({
    mutationFn: async (plan: BillingPlan) => {
      const res = await api.post<ApiEnvelope<CheckoutResult>>("/billing/checkout", { plan });
      return res.data.data;
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useBillingPayments(page: number) {
  const { auth } = useAuth();
  return useQuery({
    queryKey: ["billing", "payments", auth?.activeTenant?.tenant_id, page],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<SubscriptionPayment[]>>("/billing/payments", {
        params: { page, limit: 20 },
      });
      return { payments: res.data.data ?? [], meta: res.data.meta };
    },
    enabled: Boolean(auth?.activeTenant),
  });
}

// ---- super-admin console ----

export function useAdminSubscriptions(page: number, status: string) {
  return useQuery({
    queryKey: ["admin", "subscriptions", page, status],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<AdminSubscription[]>>("/admin/subscriptions", {
        params: { page, limit: 20, status: status || undefined },
      });
      return { subscriptions: res.data.data ?? [], meta: res.data.meta };
    },
  });
}

export function useAdminOwners(page: number) {
  return useQuery({
    queryKey: ["admin", "owners", page],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<PlatformOwner[]>>("/admin/owners", {
        params: { page, limit: 20 },
      });
      return { owners: res.data.data ?? [], meta: res.data.meta };
    },
  });
}

export function useAdminBillingStats() {
  return useQuery({
    queryKey: ["admin", "billing", "stats"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<BillingStats>>("/admin/billing/stats");
      return res.data.data;
    },
  });
}

export function useAdminBillingSettings() {
  return useQuery({
    queryKey: ["admin", "billing", "settings"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<PlatformPlans>>("/admin/billing/settings");
      return res.data.data;
    },
  });
}

function invalidateAdminBilling(queryClient: ReturnType<typeof useQueryClient>) {
  queryClient.invalidateQueries({ queryKey: ["admin", "subscriptions"] });
  queryClient.invalidateQueries({ queryKey: ["admin", "billing", "stats"] });
  queryClient.invalidateQueries({ queryKey: ["admin", "owners"] });
}

export function useAdminMarkPaid() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ tenantId, note }: { tenantId: string; note: string }) => {
      const res = await api.post<ApiEnvelope<Subscription>>(
        `/admin/subscriptions/${tenantId}/mark-paid`,
        { note },
      );
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Payment recorded — subscription extended");
      invalidateAdminBilling(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useAdminSetSubscriptionStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ tenantId, status }: { tenantId: string; status: "active" | "inactive" }) => {
      const res = await api.patch<ApiEnvelope<Subscription>>(
        `/admin/subscriptions/${tenantId}/status`,
        { status },
      );
      return res.data.data;
    },
    onSuccess: (sub) => {
      toast.success(`Subscription is now ${sub.status}`);
      invalidateAdminBilling(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useAdminUpdatePrices() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { monthly_price: number; yearly_price: number }) => {
      const res = await api.put<ApiEnvelope<PlatformPlans>>("/admin/billing/settings", input);
      return res.data.data;
    },
    onSuccess: (settings) => {
      toast.success("Prices updated");
      queryClient.setQueryData(["admin", "billing", "settings"], settings);
      queryClient.invalidateQueries({ queryKey: ["billing", "plans"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
