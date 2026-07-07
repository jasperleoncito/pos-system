"use client";

import { useCallback, useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import { getAccessToken } from "@/lib/auth-store";
import type { Order } from "@/types/order";

const POLL_FALLBACK_MS = 10_000;

export function useKitchenOrders() {
  return useQuery({
    queryKey: ["kitchen", "orders"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Order[] | null>>("/kitchen/orders");
      return res.data.data ?? [];
    },
    refetchInterval: POLL_FALLBACK_MS, // fallback when SSE is down
  });
}

export function useSetKitchenStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ orderId, status }: { orderId: string; status: string }) => {
      const res = await api.patch<ApiEnvelope<Order>>(`/kitchen/orders/${orderId}/status`, { status });
      return res.data.data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["kitchen"] }),
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useSetItemStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ orderId, itemId, status }: { orderId: string; itemId: string; status: string }) => {
      const res = await api.patch<ApiEnvelope<Order>>(
        `/kitchen/orders/${orderId}/items/${itemId}/status`,
        { status },
      );
      return res.data.data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["kitchen"] }),
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

/** Two-tone chime via Web Audio — no asset file needed. */
export function useKitchenChime() {
  return useCallback(() => {
    try {
      const ctx = new AudioContext();
      const play = (freq: number, start: number) => {
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = "sine";
        osc.frequency.value = freq;
        gain.gain.setValueAtTime(0.4, ctx.currentTime + start);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + start + 0.35);
        osc.connect(gain).connect(ctx.destination);
        osc.start(ctx.currentTime + start);
        osc.stop(ctx.currentTime + start + 0.4);
      };
      play(880, 0);
      play(1175, 0.18);
      setTimeout(() => ctx.close(), 1000);
    } catch {
      // Audio blocked until user interaction — safe to ignore.
    }
  }, []);
}

/**
 * Subscribes to the kitchen SSE stream. Any event refreshes the queue;
 * order_fired additionally triggers onNewOrder (sound). EventSource
 * cannot send headers, so the access token rides the query string;
 * the stream is recreated on error with a fresh token.
 */
export function useKitchenStream(onNewOrder: () => void) {
  const queryClient = useQueryClient();
  const onNewOrderRef = useRef(onNewOrder);
  onNewOrderRef.current = onNewOrder;

  useEffect(() => {
    let source: EventSource | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let closed = false;

    const connect = () => {
      const token = getAccessToken();
      if (!token || closed) return;

      const base = process.env.NEXT_PUBLIC_API_URL ?? "/api/v1";
      source = new EventSource(`${base}/kitchen/stream?token=${encodeURIComponent(token)}`);

      source.addEventListener("kitchen", (e) => {
        queryClient.invalidateQueries({ queryKey: ["kitchen"] });
        try {
          const event = JSON.parse((e as MessageEvent).data) as { type: string };
          if (event.type === "order_fired") {
            onNewOrderRef.current();
          }
        } catch {
          // Malformed event — the refetch above still keeps us fresh.
        }
      });

      source.onerror = () => {
        source?.close();
        if (!closed) {
          // Token may have expired; reconnect with the current one.
          retryTimer = setTimeout(connect, 3000);
        }
      };
    };

    connect();
    return () => {
      closed = true;
      source?.close();
      if (retryTimer) clearTimeout(retryTimer);
    };
  }, [queryClient]);
}
