"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import type { Coupon, Discount, PromoType } from "@/types/promo";

export interface DiscountInput {
  name: string;
  type: PromoType;
  percent_value: number;
  amount_value: number;
  requires_approval: boolean;
  is_active: boolean;
}

export interface CouponInput {
  code: string;
  discount_type: PromoType;
  percent_value: number;
  amount_value: number;
  min_order_amount: number;
  max_uses: number;
  valid_from: string | null;
  valid_to: string | null;
  is_active: boolean;
}

export function useDiscounts() {
  return useQuery({
    queryKey: ["promos", "discounts"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Discount[] | null>>("/discounts");
      return res.data.data ?? [];
    },
  });
}

export function useSaveDiscount() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: DiscountInput }) => {
      if (id) {
        const res = await api.put<ApiEnvelope<Discount>>(`/discounts/${id}`, input);
        return res.data.data;
      }
      const res = await api.post<ApiEnvelope<Discount>>("/discounts", input);
      return res.data.data;
    },
    onSuccess: (_, { id }) => {
      toast.success(id ? "Discount updated" : "Discount created");
      queryClient.invalidateQueries({ queryKey: ["promos"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useDeleteDiscount() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/discounts/${id}`);
    },
    onSuccess: () => {
      toast.success("Discount deleted");
      queryClient.invalidateQueries({ queryKey: ["promos"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useCoupons() {
  return useQuery({
    queryKey: ["promos", "coupons"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Coupon[] | null>>("/coupons");
      return res.data.data ?? [];
    },
  });
}

export function useSaveCoupon() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: CouponInput }) => {
      if (id) {
        const res = await api.put<ApiEnvelope<Coupon>>(`/coupons/${id}`, input);
        return res.data.data;
      }
      const res = await api.post<ApiEnvelope<Coupon>>("/coupons", input);
      return res.data.data;
    },
    onSuccess: (_, { id }) => {
      toast.success(id ? "Coupon updated" : "Coupon created");
      queryClient.invalidateQueries({ queryKey: ["promos"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useDeleteCoupon() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/coupons/${id}`);
    },
    onSuccess: () => {
      toast.success("Coupon deleted");
      queryClient.invalidateQueries({ queryKey: ["promos"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useValidateCoupon() {
  return useMutation({
    mutationFn: async ({ code, subtotal }: { code: string; subtotal: number }) => {
      const res = await api.post<ApiEnvelope<{ coupon: Coupon; discount: number }>>(
        "/coupons/validate",
        { code, subtotal },
      );
      return res.data.data;
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
