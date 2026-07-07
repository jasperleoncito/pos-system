import type { AuthResult, Membership, User } from "@/types/auth";

/**
 * Client-side auth state persisted in localStorage. Module-level so the
 * axios interceptors can read tokens without React context.
 */
const STORAGE_KEY = "pos.auth.v1";

export interface AuthState {
  user: User;
  memberships: Membership[];
  activeTenant: Membership | null;
  accessToken: string;
  refreshToken: string;
}

type Listener = (state: AuthState | null) => void;

let cached: AuthState | null | undefined;
const listeners = new Set<Listener>();

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

export function getAuth(): AuthState | null {
  if (!isBrowser()) return null;
  if (cached !== undefined) return cached;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    cached = raw ? (JSON.parse(raw) as AuthState) : null;
  } catch {
    cached = null;
  }
  return cached;
}

function persist(state: AuthState | null): void {
  cached = state;
  if (isBrowser()) {
    if (state === null) {
      window.localStorage.removeItem(STORAGE_KEY);
    } else {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    }
  }
  listeners.forEach((fn) => fn(state));
}

export function saveAuthResult(result: AuthResult): AuthState {
  const state: AuthState = {
    user: result.user,
    memberships: result.memberships ?? [],
    activeTenant: result.active_tenant ?? null,
    accessToken: result.access_token,
    refreshToken: result.refresh_token,
  };
  persist(state);
  return state;
}

/** Updates tokens after a silent refresh, keeping user/tenant intact. */
export function updateTokens(accessToken: string, refreshToken: string): void {
  const current = getAuth();
  if (!current) return;
  persist({ ...current, accessToken, refreshToken });
}

/** Updates the access token + active tenant after a tenant switch. */
export function updateActiveTenant(accessToken: string, membership: Membership): void {
  const current = getAuth();
  if (!current) return;
  persist({ ...current, accessToken, activeTenant: membership });
}

export function clearAuth(): void {
  persist(null);
}

export function getAccessToken(): string | null {
  return getAuth()?.accessToken ?? null;
}

export function getRefreshToken(): string | null {
  return getAuth()?.refreshToken ?? null;
}

/** Subscribe to auth changes; returns an unsubscribe function. */
export function subscribeAuth(fn: Listener): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
