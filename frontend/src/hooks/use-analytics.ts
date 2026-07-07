"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";

export interface PeriodStat {
  label: "today" | "wtd" | "mtd" | "ytd";
  sales: number;
  orders: number;
  prev_sales: number;
  prev_orders: number;
}

export interface AnalyticsSummary {
  gross_sales: number;
  orders: number;
  aov: number;
  refunds: number;
  expenses: number;
  cogs: number;
  profit: number;
  net_sales: number;
  items_sold: number;
}

export interface TopEntry {
  name: string;
  qty?: number;
  orders?: number;
  revenue: number;
}

export interface HourPoint {
  hour: number;
  sales: number;
  orders: number;
}

export interface HeatCell {
  day_of_week: number;
  hour: number;
  sales: number;
  orders: number;
}

export interface PaymentSlice {
  method: string;
  amount: number;
  count: number;
}

export interface DashboardData {
  summary: AnalyticsSummary;
  top_products: TopEntry[] | null;
  top_categories: TopEntry[] | null;
  top_employees: TopEntry[] | null;
  hourly: HourPoint[];
  heatmap: HeatCell[] | null;
  payment_mix: PaymentSlice[] | null;
}

export interface Expense {
  id: string;
  category: string;
  description: string;
  amount: number;
  expense_date: string;
  created_at: string;
}

export function useOverview() {
  return useQuery({
    queryKey: ["analytics", "overview"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<PeriodStat[]>>("/analytics/overview");
      return res.data.data ?? [];
    },
    refetchInterval: 120_000,
  });
}

export function useDashboard(from: string, to: string) {
  return useQuery({
    queryKey: ["analytics", "dashboard", from, to],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<DashboardData>>("/analytics/dashboard", {
        params: { from, to },
      });
      return res.data.data;
    },
    refetchInterval: 120_000,
  });
}

export function useExpenses(from: string, to: string) {
  return useQuery({
    queryKey: ["analytics", "expenses", from, to],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Expense[] | null>>("/expenses", {
        params: { from, to },
      });
      return res.data.data ?? [];
    },
  });
}

export function useCreateExpense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: {
      category: string;
      description: string;
      amount: number;
      expense_date?: string;
    }) => {
      const res = await api.post<ApiEnvelope<Expense>>("/expenses", input);
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Expense recorded");
      queryClient.invalidateQueries({ queryKey: ["analytics"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useDeleteExpense() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/expenses/${id}`),
    onSuccess: () => {
      toast.success("Expense deleted");
      queryClient.invalidateQueries({ queryKey: ["analytics"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
