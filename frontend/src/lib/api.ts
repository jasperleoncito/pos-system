import axios from "axios";

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

/**
 * Shared Axios instance. The auth phase adds interceptors for attaching
 * access tokens and transparently refreshing them on 401 responses.
 */
export const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL ?? "/api/v1",
  headers: { "Content-Type": "application/json" },
  timeout: 30_000,
});

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
