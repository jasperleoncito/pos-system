"use client";

import { useEffect, useState } from "react";
import { BadgeCheck, CalendarPlus, Loader2, Plus, Power, UserRound } from "lucide-react";
import { toast } from "sonner";

import {
  useAdminGrantMonths,
  useAdminMarkPaid,
  useAdminSetSubscriptionStatus,
  useAdminSubscriptions,
} from "@/hooks/use-billing";
import { useAdminCreateTenant, useAdminStats, useSetTenantStatus } from "@/hooks/use-tenant";
import { formatCentavos } from "@/lib/currency";
import type { AdminSubscription } from "@/types/billing";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";

const SUB_BADGE: Record<string, string> = {
  active: "bg-emerald-600 text-white",
  pending: "bg-amber-500 text-white",
  inactive: "bg-destructive text-white",
};

function slugify(input: string): string {
  return input.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");
}

function fmtDate(value?: string | null) {
  if (!value) return "—";
  return new Date(value).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

function dueTone(periodEnd: string, status: string): string {
  if (status !== "active") return "text-muted-foreground";
  const days = (new Date(periodEnd).getTime() - Date.now()) / 86_400_000;
  if (days < 0) return "text-destructive font-medium";
  if (days <= 3) return "text-amber-600 font-medium";
  return "";
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
              <Input id="nb-email" type="email" value={ownerEmail} onChange={(e) => setOwnerEmail(e.target.value)} />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
          <Button onClick={submit} disabled={createTenant.isPending}>
            {createTenant.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Create business
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function MarkPaidDialog({ sub, open, onOpenChange }: { sub: AdminSubscription | null; open: boolean; onOpenChange: (o: boolean) => void }) {
  const markPaid = useAdminMarkPaid();
  const [note, setNote] = useState("");
  useEffect(() => {
    if (open) setNote("");
  }, [open]);
  if (!sub) return null;
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Record a payment for {sub.tenant_name}</DialogTitle>
          <DialogDescription>
            Extends the {sub.plan} subscription by one period from {fmtDate(sub.current_period_end)} (or
            from today if past due) — like a Xendit payment. Use for bank transfers or comps.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="mp-note">Note (optional)</Label>
          <Textarea id="mp-note" rows={2} placeholder="e.g. Paid via bank transfer, ref #12345" value={note} onChange={(e) => setNote(e.target.value)} />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button disabled={markPaid.isPending} onClick={() => markPaid.mutate({ tenantId: sub.tenant_id, note: note.trim() }, { onSuccess: () => onOpenChange(false) })}>
            {markPaid.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Record payment
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function GrantMonthsDialog({ sub, open, onOpenChange }: { sub: AdminSubscription | null; open: boolean; onOpenChange: (o: boolean) => void }) {
  const grant = useAdminGrantMonths();
  const [months, setMonths] = useState("1");
  useEffect(() => {
    if (open) setMonths("1");
  }, [open]);
  if (!sub) return null;
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Grant free months to {sub.tenant_name}</DialogTitle>
          <DialogDescription>
            Extends the subscription and activates it — no payment. Comps a trial or covers an issue.
            Adds to the current end date ({fmtDate(sub.current_period_end)}).
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label>Months to grant</Label>
          <Select value={months} onValueChange={setMonths}>
            <SelectTrigger className="w-full" aria-label="Months to grant">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {[1, 2, 3, 4, 5, 6].map((m) => (
                <SelectItem key={m} value={String(m)}>
                  {m} month{m === 1 ? "" : "s"}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button disabled={grant.isPending} onClick={() => grant.mutate({ tenantId: sub.tenant_id, months: Number(months) }, { onSuccess: () => onOpenChange(false) })}>
            {grant.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Grant {months} month{months === "1" ? "" : "s"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function BusinessRow({ sub, onMarkPaid, onGrant }: { sub: AdminSubscription; onMarkPaid: (s: AdminSubscription) => void; onGrant: (s: AdminSubscription) => void }) {
  const setSubStatus = useAdminSetSubscriptionStatus();
  const setTenantStatus = useSetTenantStatus();
  const subActive = sub.status === "active";
  const tenantActive = sub.tenant_status === "active";

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-lg border p-3">
      <div className="min-w-0 flex-1">
        <p className="flex flex-wrap items-center gap-2 text-sm font-medium">
          <span className="truncate">{sub.tenant_name}</span>
          {!tenantActive && <Badge variant="destructive">suspended</Badge>}
          <Badge className={SUB_BADGE[sub.status]}>sub: {sub.status}</Badge>
          <Badge variant="outline" className="capitalize">{sub.plan}</Badge>
        </p>
        <p className="truncate text-xs text-muted-foreground">
          <UserRound className="mr-1 inline size-3" aria-hidden />
          {sub.owner_name || "—"} · {sub.owner_email || "no owner"} · /{sub.tenant_slug}
        </p>
        <p className="text-xs">
          <span className={dueTone(sub.current_period_end, sub.status)}>
            {sub.status === "pending" ? "never paid" : `due ${fmtDate(sub.current_period_end)}`}
          </span>
          <span className="text-muted-foreground">
            {" · last paid "}
            {sub.last_paid_at ? `${fmtDate(sub.last_paid_at)} (${formatCentavos(sub.last_paid_amount ?? 0)})` : "never"}
          </span>
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-1">
        <Button variant="outline" size="sm" onClick={() => onMarkPaid(sub)}>
          <BadgeCheck className="size-4" aria-hidden />
          Mark paid
        </Button>
        <Button variant="outline" size="sm" onClick={() => onGrant(sub)}>
          <CalendarPlus className="size-4" aria-hidden />
          Grant months
        </Button>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="ghost" size="sm" disabled={setSubStatus.isPending}>
              <Power className="size-4" aria-hidden />
              {subActive ? "Deactivate" : "Reactivate"}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{subActive ? "Deactivate" : "Reactivate"} subscription for {sub.tenant_name}?</AlertDialogTitle>
              <AlertDialogDescription>
                {subActive
                  ? "Members are locked out until payment or reactivation. Doesn't change the due date."
                  : "The business regains access without a payment. The due date is unchanged, so the sweep may deactivate it again if still past due."}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction onClick={() => setSubStatus.mutate({ tenantId: sub.tenant_id, status: subActive ? "inactive" : "active" })}>
                {subActive ? "Deactivate" : "Reactivate"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="ghost" size="sm" disabled={setTenantStatus.isPending}>
              {tenantActive ? "Suspend" : "Unsuspend"}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>{tenantActive ? "Suspend" : "Unsuspend"} {sub.tenant_name}?</AlertDialogTitle>
              <AlertDialogDescription>
                {tenantActive
                  ? "Suspends the whole business regardless of billing — members lose access until you unsuspend."
                  : "Lifts the suspension. Billing state is separate and still applies."}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction onClick={() => setTenantStatus.mutate({ tenantId: sub.tenant_id, status: tenantActive ? "suspended" : "active" })}>
                {tenantActive ? "Suspend" : "Unsuspend"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  );
}

export default function AdminBusinessesPage() {
  const { data: stats } = useAdminStats();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("all");
  const { data, isLoading } = useAdminSubscriptions(page, statusFilter === "all" ? "" : statusFilter);
  const [markPaidTarget, setMarkPaidTarget] = useState<AdminSubscription | null>(null);
  const [grantTarget, setGrantTarget] = useState<AdminSubscription | null>(null);

  const total = data?.meta?.total ?? 0;
  const limit = data?.meta?.limit ?? 20;
  const pageCount = Math.max(1, Math.ceil(total / limit));

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">Businesses</h1>
          <p className="text-muted-foreground">Every business, its subscription, and platform analytics</p>
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
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <div>
            <CardTitle className="text-base">All businesses</CardTitle>
            <CardDescription>{total} total — soonest due first</CardDescription>
          </div>
          <Select
            value={statusFilter}
            onValueChange={(v) => {
              setStatusFilter(v);
              setPage(1);
            }}
          >
            <SelectTrigger className="w-[170px]" aria-label="Filter by subscription status">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All subscriptions</SelectItem>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="pending">Awaiting payment</SelectItem>
              <SelectItem value="inactive">Inactive</SelectItem>
            </SelectContent>
          </Select>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading && Array.from({ length: 4 }, (_, i) => <Skeleton key={i} className="h-20 w-full" />)}

          {data?.subscriptions.map((s) => (
            <BusinessRow key={s.id} sub={s} onMarkPaid={setMarkPaidTarget} onGrant={setGrantTarget} />
          ))}

          {data && data.subscriptions.length === 0 && (
            <p className="py-6 text-center text-sm text-muted-foreground">No businesses match this filter.</p>
          )}

          {pageCount > 1 && (
            <div className="flex items-center justify-between pt-2">
              <Button variant="outline" size="sm" disabled={page <= 1 || isLoading} onClick={() => setPage((p) => p - 1)}>
                Previous
              </Button>
              <span className="text-sm text-muted-foreground">Page {page} of {pageCount}</span>
              <Button variant="outline" size="sm" disabled={page >= pageCount || isLoading} onClick={() => setPage((p) => p + 1)}>
                Next
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      <MarkPaidDialog sub={markPaidTarget} open={Boolean(markPaidTarget)} onOpenChange={(o) => !o && setMarkPaidTarget(null)} />
      <GrantMonthsDialog sub={grantTarget} open={Boolean(grantTarget)} onOpenChange={(o) => !o && setGrantTarget(null)} />
    </div>
  );
}
