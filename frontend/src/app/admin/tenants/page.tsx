"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";

import { useAdminTenants, useSetTenantStatus } from "@/hooks/use-tenant";
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
    <div className="flex items-center gap-3 rounded-lg border p-3">
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

  const total = data?.meta?.total ?? 0;
  const limit = data?.meta?.limit ?? 20;
  const pageCount = Math.max(1, Math.ceil(total / limit));

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Tenants</h1>
        <p className="text-muted-foreground">All businesses on the platform</p>
      </header>

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
