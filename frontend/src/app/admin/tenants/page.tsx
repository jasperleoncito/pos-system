"use client";

import { useState } from "react";
import { Loader2, Plus } from "lucide-react";
import { toast } from "sonner";

import {
  useAdminCreateTenant,
  useAdminStats,
  useAdminTenants,
  useSetTenantStatus,
} from "@/hooks/use-tenant";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";

function slugify(input: string): string {
  return input
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function NewBusinessDialog() {
  const createTenant = useAdminCreateTenant();
  const [open, setOpen] = useState(false);
  const [businessName, setBusinessName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugTouched, setSlugTouched] = useState(false);
  const [ownerName, setOwnerName] = useState("");
  const [ownerEmail, setOwnerEmail] = useState("");

  const reset = () => {
    setBusinessName("");
    setSlug("");
    setSlugTouched(false);
    setOwnerName("");
    setOwnerEmail("");
  };

  const submit = () => {
    if (businessName.trim().length < 2 || slug.trim().length < 2) {
      toast.error("Business name and URL slug are required");
      return;
    }
    if (ownerName.trim().length < 2 || !ownerEmail.trim()) {
      toast.error("Owner name and email are required");
      return;
    }
    createTenant.mutate(
      {
        business_name: businessName.trim(),
        business_slug: slug.trim(),
        owner_full_name: ownerName.trim(),
        owner_email: ownerEmail.trim(),
      },
      {
        onSuccess: () => {
          setOpen(false);
          reset();
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Button onClick={() => setOpen(true)}>
        <Plus className="size-4" aria-hidden />
        New business
      </Button>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create a business</DialogTitle>
          <DialogDescription>
            The owner gets an email to set their password (or keeps their existing login if the
            email is already registered).
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="nb-name">Business name</Label>
            <Input
              id="nb-name"
              value={businessName}
              onChange={(e) => {
                setBusinessName(e.target.value);
                if (!slugTouched) setSlug(slugify(e.target.value));
              }}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="nb-slug">URL slug</Label>
            <Input
              id="nb-slug"
              placeholder="my-restaurant"
              value={slug}
              onChange={(e) => {
                setSlugTouched(true);
                setSlug(slugify(e.target.value));
              }}
            />
            <p className="text-xs text-muted-foreground">Used in the app URL: /{slug || "my-restaurant"}/…</p>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="nb-owner">Owner name</Label>
              <Input id="nb-owner" value={ownerName} onChange={(e) => setOwnerName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="nb-email">Owner email</Label>
              <Input
                id="nb-email"
                type="email"
                value={ownerEmail}
                onChange={(e) => setOwnerEmail(e.target.value)}
              />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={submit} disabled={createTenant.isPending}>
            {createTenant.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Create business
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

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
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">Platform</h1>
          <p className="text-muted-foreground">System analytics and every business on the platform</p>
        </div>
        <NewBusinessDialog />
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
