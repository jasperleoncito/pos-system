"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";

import { useAdminStats, useAdminTenants, useSetTenantStatus } from "@/hooks/use-tenant";
import { formatCentavos } from "@/lib/currency";
import type { Tenant } from "@/types/tenant";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

function TenantRow({ tenant }: { tenant: Tenant }) {
  const setStatus = useSetTenantStatus();
  const isActive = tenant.status === "active";
  const nextStatus = isActive ? "suspended" : "active";

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-lg border p-3">
      <div className="min-w-0 flex-1">
        <p className="flex items-center gap-2 text-sm font-medium">
          {tenant.name}
          <Badge
            variant={isActive ? "default" : "destructive"}
            className={isActive ? "bg-emerald-600" : undefined}
          >
            {tenant.status}
          </Badge>
        </p>
        <p className="truncate text-xs text-muted-foreground">
          /{tenant.slug} · {tenant.currency} · created{" "}
          {new Date(tenant.created_at).toLocaleDateString()}
        </p>
      </div>

      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button
            variant={isActive ? "outline" : "default"}
            size="sm"
            disabled={setStatus.isPending}
          >
            {isActive ? "Suspend" : "Activate"}
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {isActive ? "Suspend" : "Activate"} {tenant.name}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              {isActive
                ? "Members will lose access until the business is reactivated."
                : "Members will regain access to this business."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => setStatus.mutate({ tenantId: tenant.id, status: nextStatus })}
            >
              {isActive ? "Suspend" : "Activate"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

export default function AdminTenantsPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminTenants(page);
  const { data: stats } = useAdminStats();

  const total = data?.meta?.total ?? 0;
  const limit = data?.meta?.limit ?? 20;
  const pageCount = Math.max(1, Math.ceil(total / limit));

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Platform</h1>
        <p className="text-muted-foreground">System analytics and every business on the platform</p>
      </header>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {[
          { label: "Businesses", value: stats ? `${stats.tenants_active}/${stats.tenants_total}` : "—", hint: "active / total" },
          { label: "Users", value: stats ? String(stats.users_total) : "—", hint: "all accounts" },
          { label: "Orders (30d)", value: stats ? String(stats.orders_30d) : "—", hint: "platform-wide" },
          { label: "GMV (30d)", value: stats ? formatCentavos(stats.gmv_30d) : "—", hint: "gross sales" },
        ].map((s) => (
          <Card key={s.label} className="py-4">
            <CardContent className="px-4">
              <p className="text-xs text-muted-foreground">{s.label}</p>
              <p className="text-xl font-bold tabular-nums tracking-tight">{s.value}</p>
              <p className="text-xs text-muted-foreground">{s.hint}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Businesses</CardTitle>
          <CardDescription>{total} total</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading &&
            Array.from({ length: 3 }, (_, i) => <Skeleton key={i} className="h-16 w-full" />)}

          {data?.tenants.map((t) => <TenantRow key={t.id} tenant={t} />)}

          {data && data.tenants.length === 0 && (
            <p className="py-6 text-center text-sm text-muted-foreground">No tenants yet.</p>
          )}

          {pageCount > 1 && (
            <div className="flex items-center justify-between pt-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1 || isLoading}
                onClick={() => setPage((p) => p - 1)}
              >
                Previous
              </Button>
              <span className="text-sm text-muted-foreground">
                Page {page} of {pageCount}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= pageCount || isLoading}
                onClick={() => setPage((p) => p + 1)}
              >
                {isLoading && <Loader2 className="size-4 animate-spin" aria-hidden />}
                Next
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
