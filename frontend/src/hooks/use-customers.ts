"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";

export type CustomerTier = "regular" | "silver" | "gold" | "vip";

export interface Customer {
  id: string;
  full_name: string;
  phone: string;
  email: string;
  birthday: string | null;
  notes: string;
  points_balance: number;
  lifetime_points: number;
  tier: CustomerTier;
  is_active: boolean;
  created_at: string;
}

export interface CustomerInput {
  full_name: string;
  phone: string;
  email: string;
  birthday?: string;
  notes: string;
  is_active: boolean;
}

export interface LoyaltySettings {
  is_enabled: boolean;
  earn_rate: number; // centavos spent per point
  redeem_value: number; // centavos of value per point
  silver_threshold: number;
  gold_threshold: number;
  vip_threshold: number;
  silver_multiplier: number;
  gold_multiplier: number;
  vip_multiplier: number;
}

export interface LoyaltyTransaction {
  id: string;
  customer_id: string;
  order_id: string | null;
  order_number?: number;
  type: "earn" | "redeem" | "adjust";
  points: number;
  balance_after: number;
  notes: string;
  created_at: string;
}

export function useCustomers(search = "") {
  return useQuery({
    queryKey: ["customers", "list", search],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Customer[] | null>>("/customers", {
        params: search ? { search } : undefined,
      });
      return res.data.data ?? [];
    },
  });
}

export function useSaveCustomer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: CustomerInput }) => {
      if (id) {
        const res = await api.put<ApiEnvelope<Customer>>(`/customers/${id}`, input);
        return res.data.data;
      }
      const res = await api.post<ApiEnvelope<Customer>>("/customers", input);
      return res.data.data;
    },
    onSuccess: (_, { id }) => {
      toast.success(id ? "Customer updated" : "Customer created");
      queryClient.invalidateQueries({ queryKey: ["customers"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useDeleteCustomer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/customers/${id}`),
    onSuccess: () => {
      toast.success("Customer removed");
      queryClient.invalidateQueries({ queryKey: ["customers"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useLoyaltyHistory(customerId: string | null) {
  return useQuery({
    queryKey: ["customers", "loyalty", customerId],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<LoyaltyTransaction[] | null>>(`/customers/${customerId}/loyalty`);
      return res.data.data ?? [];
    },
    enabled: Boolean(customerId),
  });
}

export function useLoyaltySettings() {
  return useQuery({
    queryKey: ["loyalty", "settings"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<LoyaltySettings>>("/loyalty/settings");
      return res.data.data;
    },
  });
}

export function useSaveLoyaltySettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: LoyaltySettings) => {
      const res = await api.put<ApiEnvelope<LoyaltySettings>>("/loyalty/settings", input);
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Loyalty settings saved");
      queryClient.invalidateQueries({ queryKey: ["loyalty"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
