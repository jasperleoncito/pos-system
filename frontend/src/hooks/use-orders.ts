"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import type {
  CashMovement,
  CreateOrderInput,
  DrawerSession,
  Order,
  PaymentLineInput,
  Receipt,
} from "@/types/order";

export function useCreateOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateOrderInput) => {
      const res = await api.post<ApiEnvelope<Order>>("/orders", input);
      return res.data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function usePayOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ orderId, payments }: { orderId: string; payments: PaymentLineInput[] }) => {
      const res = await api.post<ApiEnvelope<Order>>(`/orders/${orderId}/payments`, { payments });
      return res.data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["orders"] });
      queryClient.invalidateQueries({ queryKey: ["drawer"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useSetHold() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ orderId, hold }: { orderId: string; hold: boolean }) => {
      const res = await api.post<ApiEnvelope<Order>>(`/orders/${orderId}/hold`, { hold });
      return res.data.data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["orders"] }),
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useOrders(params: { status?: string; page?: number; limit?: number }) {
  return useQuery({
    queryKey: ["orders", params],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Order[] | null>>("/orders", {
        params: {
          status: params.status || undefined,
          page: params.page ?? 1,
          limit: params.limit ?? 25,
        },
      });
      return { orders: res.data.data ?? [], meta: res.data.meta };
    },
  });
}

export function useReceipt(orderId: string | null) {
  return useQuery({
    queryKey: ["orders", "receipt", orderId],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Receipt>>(`/orders/${orderId}/receipt`);
      return res.data.data;
    },
    enabled: Boolean(orderId),
  });
}

// ---- cash drawer ----

export function useCurrentDrawer() {
  return useQuery({
    queryKey: ["drawer", "current"],
    queryFn: async () => {
      try {
        const res = await api.get<
          ApiEnvelope<{ session: DrawerSession; movements: CashMovement[] }>
        >("/cash-drawer/current");
        return res.data.data;
      } catch {
        return null; // 404 = no open drawer
      }
    },
    staleTime: 15_000,
  });
}

export function useOpenDrawer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (openingFloat: number) => {
      const res = await api.post<ApiEnvelope<DrawerSession>>("/cash-drawer/open", {
        opening_float: openingFloat,
      });
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Drawer opened");
      queryClient.invalidateQueries({ queryKey: ["drawer"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useCloseDrawer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (countedCash: number) => {
      const res = await api.post<ApiEnvelope<DrawerSession>>("/cash-drawer/close", {
        counted_cash: countedCash,
      });
      return res.data.data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["drawer"] }),
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
