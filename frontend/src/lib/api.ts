import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";

import {
  clearAuth,
  getAccessToken,
  getRefreshToken,
  updateTokens,
} from "@/lib/auth-store";
import type { AuthResult } from "@/types/auth";

export interface ApiEnvelope<T> {
  success: boolean;
  message: string;
  data: T;
  meta?: {
    total: number;
    page: number;
    limit: number;
  };
}

export interface ApiErrorEnvelope {
  success: false;
  message: string;
  errors: string[];
}

const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "/api/v1";

/** Shared Axios instance with bearer attachment and silent refresh. */
export const api = axios.create({
  baseURL: BASE_URL,
  headers: { "Content-Type": "application/json" },
  timeout: 30_000,
});

api.interceptors.request.use((config) => {
  const token = getAccessToken();
  if (token && !config.headers.Authorization) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

/** Single-flight refresh: concurrent 401s share one refresh request. */
let refreshPromise: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return null;
  try {
    // Bare axios call: must not recurse through the interceptors.
    const res = await axios.post<ApiEnvelope<AuthResult>>(
      `${BASE_URL}/auth/refresh`,
      { refresh_token: refreshToken },
      { timeout: 15_000 },
    );
    const { access_token, refresh_token } = res.data.data;
    updateTokens(access_token, refresh_token);
    return access_token;
  } catch {
    clearAuth();
    return null;
  }
}

interface RetriableConfig extends InternalAxiosRequestConfig {
  _retried?: boolean;
}

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as RetriableConfig | undefined;
    const isAuthRoute = config?.url?.startsWith("/auth/") ?? false;

    if (error.response?.status === 401 && config && !config._retried && !isAuthRoute) {
      config._retried = true;
      refreshPromise ??= refreshAccessToken().finally(() => {
        refreshPromise = null;
      });
      const newToken = await refreshPromise;
      if (newToken) {
        config.headers.Authorization = `Bearer ${newToken}`;
        return api(config);
      }
      if (typeof window !== "undefined") {
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
  },
);

/** Extracts a user-friendly message from any thrown API error. */
export function getApiErrorMessage(error: unknown): string {
  if (axios.isAxiosError<ApiErrorEnvelope>(error)) {
    return error.response?.data?.message ?? error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "Unexpected error";
}
