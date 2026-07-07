"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import {
  clearAuth,
  getAuth,
  getRefreshToken,
  saveAuthResult,
  subscribeAuth,
  updateActiveTenant,
  type AuthState,
} from "@/lib/auth-store";
import type { AuthResult, DeviceSession, Membership } from "@/types/auth";
import type { LoginInput, RegisterInput } from "@/schemas/auth";

/** Reactive view of the persisted auth state. */
export function useAuth(): { auth: AuthState | null; isReady: boolean } {
  const [auth, setAuth] = useState<AuthState | null>(null);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    setAuth(getAuth());
    setIsReady(true);
    return subscribeAuth(setAuth);
  }, []);

  return { auth, isReady };
}

export function useLogin() {
  return useMutation({
    mutationFn: async (input: LoginInput) => {
      const res = await api.post<ApiEnvelope<AuthResult>>("/auth/login", input);
      return res.data.data;
    },
    onSuccess: (result) => saveAuthResult(result),
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useRegister() {
  return useMutation({
    mutationFn: async (input: Omit<RegisterInput, "confirm_password">) => {
      const res = await api.post<ApiEnvelope<AuthResult>>("/auth/register", input);
      return res.data.data;
    },
    onSuccess: (result) => {
      saveAuthResult(result);
      toast.success("Account created! Check your inbox to verify your email.");
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useLogout() {
  const router = useRouter();
  const queryClient = useQueryClient();

  return useCallback(async () => {
    const refreshToken = getRefreshToken();
    try {
      if (refreshToken) {
        await api.post("/auth/logout", { refresh_token: refreshToken });
      }
    } catch {
      // Logout must always succeed locally even if the API call fails.
    } finally {
      clearAuth();
      queryClient.clear();
      router.replace("/login");
    }
  }, [router, queryClient]);
}

export function useSwitchTenant() {
  const router = useRouter();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (tenantId: string) => {
      const res = await api.post<
        ApiEnvelope<{ access_token: string; active_tenant: Membership }>
      >("/auth/switch-tenant", { tenant_id: tenantId });
      return res.data.data;
    },
    onSuccess: (data) => {
      updateActiveTenant(data.access_token, data.active_tenant);
      queryClient.clear();
      toast.success(`Switched to ${data.active_tenant.tenant_name}`);
      router.push(`/${data.active_tenant.tenant_slug}/dashboard`);
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useSessions() {
  return useQuery({
    queryKey: ["auth", "sessions"],
    queryFn: async () => {
      const res = await api.get<
        ApiEnvelope<{ sessions: DeviceSession[]; current_session_id: string }>
      >("/auth/sessions");
      return res.data.data;
    },
  });
}

export function useRevokeSession() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (sessionId: string) => {
      await api.delete(`/auth/sessions/${sessionId}`);
    },
    onSuccess: () => {
      toast.success("Session revoked");
      queryClient.invalidateQueries({ queryKey: ["auth", "sessions"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useLogoutAll() {
  const router = useRouter();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.post("/auth/logout-all");
    },
    onSuccess: () => {
      clearAuth();
      queryClient.clear();
      toast.success("Logged out from all devices");
      router.replace("/login");
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
