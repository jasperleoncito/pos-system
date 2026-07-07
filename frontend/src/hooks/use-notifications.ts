"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";

export interface AppNotification {
  id: string;
  type: "low_stock" | "attendance" | "daily_summary" | "system";
  title: string;
  body: string;
  link: string;
  read_at: string | null;
  created_at: string;
}

export interface NotificationFeed {
  items: AppNotification[] | null;
  unread: number;
}

export interface NotificationPrefs {
  email_low_stock: boolean;
  email_attendance: boolean;
  email_daily_summary: boolean;
}

export function useNotificationFeed(limit = 30) {
  return useQuery({
    queryKey: ["notifications", "feed", limit],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<NotificationFeed>>("/notifications", {
        params: { limit },
      });
      return res.data.data;
    },
    refetchInterval: 30_000,
  });
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.post(`/notifications/${id}/read`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["notifications"] }),
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => api.post("/notifications/read-all"),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["notifications"] }),
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useNotificationPrefs() {
  return useQuery({
    queryKey: ["notifications", "prefs"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<NotificationPrefs>>("/notifications/preferences");
      return res.data.data;
    },
  });
}

export function useSaveNotificationPrefs() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (prefs: NotificationPrefs) => {
      const res = await api.put<ApiEnvelope<NotificationPrefs>>("/notifications/preferences", prefs);
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Preferences saved");
      queryClient.invalidateQueries({ queryKey: ["notifications", "prefs"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
