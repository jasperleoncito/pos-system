"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";

export interface InventoryUnit {
  id: string;
  name: string;
  abbreviation: string;
}

export interface InventoryItem {
  id: string;
  name: string;
  type: "ingredient" | "finished_good";
  unit_id: string;
  unit_abbr: string;
  current_stock: number;
  reorder_level: number;
  cost_per_unit: number;
  is_active: boolean;
}

export interface InventoryMovement {
  id: string;
  item_id: string;
  item_name: string;
  movement_type: string;
  qty_delta: number;
  qty_before: number;
  qty_after: number;
  reference_type: string;
  notes: string;
  created_at: string;
}

export interface RecipeItem {
  id?: string;
  inventory_item_id: string;
  item_name?: string;
  unit_abbr?: string;
  qty: number;
}

export function useUnits() {
  return useQuery({
    queryKey: ["inventory", "units"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<InventoryUnit[] | null>>("/units");
      return res.data.data ?? [];
    },
  });
}

export function useInventoryItems(search = "") {
  return useQuery({
    queryKey: ["inventory", "items", search],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<InventoryItem[] | null>>("/inventory/items", {
        params: search ? { search } : undefined,
      });
      return res.data.data ?? [];
    },
  });
}

export interface InventoryItemInput {
  name: string;
  type: string;
  unit_id: string;
  current_stock: number;
  reorder_level: number;
  cost_per_unit: number;
  is_active: boolean;
}

export function useSaveInventoryItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: InventoryItemInput }) => {
      if (id) {
        const res = await api.put<ApiEnvelope<InventoryItem>>(`/inventory/items/${id}`, input);
        return res.data.data;
      }
      const res = await api.post<ApiEnvelope<InventoryItem>>("/inventory/items", input);
      return res.data.data;
    },
    onSuccess: (_, { id }) => {
      toast.success(id ? "Item updated" : "Item created");
      queryClient.invalidateQueries({ queryKey: ["inventory"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useApplyMovement() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: {
      item_id: string;
      movement_type: string;
      qty: number;
      notes?: string;
    }) => {
      const res = await api.post<ApiEnvelope<InventoryMovement>>("/inventory/movements", input);
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Movement recorded");
      queryClient.invalidateQueries({ queryKey: ["inventory"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useMovements(itemId: string | null) {
  return useQuery({
    queryKey: ["inventory", "movements", itemId],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<InventoryMovement[] | null>>("/inventory/movements", {
        params: { item_id: itemId, limit: 30 },
      });
      return res.data.data ?? [];
    },
    enabled: Boolean(itemId),
  });
}

export function useRecipe(productId: string | null) {
  return useQuery({
    queryKey: ["inventory", "recipe", productId],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<RecipeItem[] | null>>(`/products/${productId}/recipe`);
      return res.data.data ?? [];
    },
    enabled: Boolean(productId),
  });
}

export function useSaveRecipe() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ productId, items }: { productId: string; items: { inventory_item_id: string; qty: number }[] }) => {
      const res = await api.put<ApiEnvelope<RecipeItem[]>>(`/products/${productId}/recipe`, { items });
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Recipe saved");
      queryClient.invalidateQueries({ queryKey: ["inventory", "recipe"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
