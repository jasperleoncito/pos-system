"use client";

import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, ChefHat, Database, HardDrive, Loader2, XCircle, Zap } from "lucide-react";

import { api, type ApiEnvelope } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface DependencyStatus {
  status: "ok" | "down";
  error?: string;
}

interface HealthData {
  status: "healthy" | "degraded";
  time: string;
  checks: {
    database: DependencyStatus;
    redis: DependencyStatus;
    storage: DependencyStatus;
  };
}

const HEALTH_REFETCH_MS = 10_000;

const CHECK_LABELS: { key: keyof HealthData["checks"]; label: string; icon: typeof Database }[] = [
  { key: "database", label: "PostgreSQL", icon: Database },
  { key: "redis", label: "Redis", icon: Zap },
  { key: "storage", label: "MinIO Storage", icon: HardDrive },
];

export default function Home() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<HealthData>>("/health");
      return res.data.data;
    },
    refetchInterval: HEALTH_REFETCH_MS,
  });

  return (
    <main className="flex min-h-dvh items-center justify-center bg-background p-6">
      <div className="w-full max-w-md space-y-8">
        <header className="space-y-3 text-center">
          <div className="mx-auto flex size-16 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-lg">
            <ChefHat className="size-8" aria-hidden />
          </div>
          <h1 className="text-3xl font-bold tracking-tight">POS System</h1>
          <p className="text-muted-foreground">
            Multi-tenant restaurant point of sale &amp; business management
          </p>
        </header>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-base">Platform status</CardTitle>
            {isLoading ? (
              <Loader2 className="size-4 animate-spin text-muted-foreground" aria-label="Checking" />
            ) : isError ? (
              <Badge variant="destructive">API unreachable</Badge>
            ) : (
              <Badge
                variant={data?.status === "healthy" ? "default" : "destructive"}
                className={data?.status === "healthy" ? "bg-emerald-600" : undefined}
              >
                {data?.status === "healthy" ? "All systems go" : "Degraded"}
              </Badge>
            )}
          </CardHeader>
          <CardContent className="space-y-3">
            {CHECK_LABELS.map(({ key, label, icon: Icon }) => {
              const check = data?.checks[key];
              return (
                <div
                  key={key}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div className="flex items-center gap-3">
                    <Icon className="size-4 text-muted-foreground" aria-hidden />
                    <span className="text-sm font-medium">{label}</span>
                  </div>
                  {isLoading || isError || !check ? (
                    <XCircle className="size-5 text-muted-foreground/40" aria-label="Unknown" />
                  ) : check.status === "ok" ? (
                    <CheckCircle2 className="size-5 text-emerald-600" aria-label="Online" />
                  ) : (
                    <XCircle className="size-5 text-destructive" aria-label="Offline" />
                  )}
                </div>
              );
            })}
          </CardContent>
        </Card>

        <p className="text-center text-xs text-muted-foreground">
          Phase 0 — infrastructure scaffold. Authentication &amp; tenants arrive in Phase 1.
        </p>
      </div>
    </main>
  );
}
