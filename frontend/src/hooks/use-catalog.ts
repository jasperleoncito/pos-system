"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import type { Category, ModifierGroup, Product, ProductInput, Tax } from "@/types/catalog";

// ---- categories ----

export function useCategories(activeOnly = false) {
  return useQuery({
    queryKey: ["catalog", "categories", activeOnly],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Category[] | null>>("/categories", {
        params: activeOnly ? { active: true } : undefined,
      });
      return res.data.data ?? [];
    },
  });
}

export interface CategoryInput {
  name: string;
  description: string;
  sort_order: number;
  is_active: boolean;
}

function invalidateCatalog(queryClient: ReturnType<typeof useQueryClient>) {
  queryClient.invalidateQueries({ queryKey: ["catalog"] });
}

export function useSaveCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: CategoryInput }) => {
      if (id) {
        const res = await api.put<ApiEnvelope<Category>>(`/categories/${id}`, input);
        return res.data.data;
      }
      const res = await api.post<ApiEnvelope<Category>>("/categories", input);
      return res.data.data;
    },
    onSuccess: (_, { id }) => {
      toast.success(id ? "Category updated" : "Category created");
      invalidateCatalog(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useDeleteCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/categories/${id}`);
    },
    onSuccess: () => {
      toast.success("Category deleted");
      invalidateCatalog(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

// ---- products ----

export interface ProductListParams {
  categoryId?: string;
  search?: string;
  page?: number;
  limit?: number;
  activeOnly?: boolean;
}

export function useProducts(params: ProductListParams) {
  return useQuery({
    queryKey: ["catalog", "products", params],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Product[] | null>>("/products", {
        params: {
          category_id: params.categoryId || undefined,
          search: params.search || undefined,
          active: params.activeOnly ? true : undefined,
          page: params.page ?? 1,
          limit: params.limit ?? 50,
        },
      });
      return { products: res.data.data ?? [], meta: res.data.meta };
    },
  });
}

export function useSaveProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: ProductInput }) => {
      if (id) {
        const res = await api.put<ApiEnvelope<Product>>(`/products/${id}`, input);
        return res.data.data;
      }
      const res = await api.post<ApiEnvelope<Product>>("/products", input);
      return res.data.data;
    },
    onSuccess: (_, { id }) => {
      toast.success(id ? "Product updated" : "Product created");
      invalidateCatalog(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useDeleteProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/products/${id}`);
    },
    onSuccess: () => {
      toast.success("Product deleted");
      invalidateCatalog(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useUploadProductImage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, file }: { id: string; file: File }) => {
      const form = new FormData();
      form.append("image", file);
      const res = await api.post<ApiEnvelope<Product>>(`/products/${id}/image`, form, {
        headers: { "Content-Type": "multipart/form-data" },
        timeout: 120_000,
      });
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Product image updated");
      invalidateCatalog(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

// ---- modifier groups ----

export interface ModifierGroupInput {
  name: string;
  min_select: number;
  max_select: number;
  is_required: boolean;
  sort_order: number;
  modifiers: { name: string; price_delta: number; is_active?: boolean }[];
}

export function useModifierGroups() {
  return useQuery({
    queryKey: ["catalog", "modifier-groups"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<ModifierGroup[] | null>>("/modifier-groups");
      return res.data.data ?? [];
    },
  });
}

export function useSaveModifierGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: ModifierGroupInput }) => {
      if (id) {
        const res = await api.put<ApiEnvelope<ModifierGroup>>(`/modifier-groups/${id}`, input);
        return res.data.data;
      }
      const res = await api.post<ApiEnvelope<ModifierGroup>>("/modifier-groups", input);
      return res.data.data;
    },
    onSuccess: (_, { id }) => {
      toast.success(id ? "Modifier group updated" : "Modifier group created");
      invalidateCatalog(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useDeleteModifierGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/modifier-groups/${id}`);
    },
    onSuccess: () => {
      toast.success("Modifier group deleted");
      invalidateCatalog(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

// ---- taxes ----

export interface TaxInput {
  name: string;
  rate_percent: number;
  is_inclusive: boolean;
  is_default: boolean;
  is_active: boolean;
}

export function useTaxes() {
  return useQuery({
    queryKey: ["catalog", "taxes"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Tax[] | null>>("/taxes");
      return res.data.data ?? [];
    },
  });
}

export function useSaveTax() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: TaxInput }) => {
      if (id) {
        const res = await api.put<ApiEnvelope<Tax>>(`/taxes/${id}`, input);
        return res.data.data;
      }
      const res = await api.post<ApiEnvelope<Tax>>("/taxes", input);
      return res.data.data;
    },
    onSuccess: (_, { id }) => {
      toast.success(id ? "Tax updated" : "Tax created");
      invalidateCatalog(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useDeleteTax() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/taxes/${id}`);
    },
    onSuccess: () => {
      toast.success("Tax deleted");
      invalidateCatalog(queryClient);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
