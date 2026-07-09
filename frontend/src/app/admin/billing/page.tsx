"use client";

import { useEffect, useState } from "react";
import { BadgeCheck, Loader2, PhilippinePeso, Power, ReceiptText, UserRound } from "lucide-react";

import {
  useAdminBillingSettings,
  useAdminBillingStats,
  useAdminMarkPaid,
  useAdminOwners,
  useAdminSetSubscriptionStatus,
  useAdminSubscriptions,
  useAdminUpdatePrices,
} from "@/hooks/use-billing";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";

const STATUS_BADGE: Record<string, string> = {
  active: "bg-emerald-600 text-white",
  pending: "bg-amber-500 text-white",
  inactive: "bg-destructive text-white",
};

function fmtDate(value?: string | null) {
  if (!value) return "—";
  return new Date(value).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

/** Days until due, for urgency coloring. */
function dueTone(periodEnd: string, status: string): string {
  if (status !== "active") return "text-muted-foreground";
  const days = (new Date(periodEnd).getTime() - Date.now()) / 86_400_000;
  if (days < 0) return "text-destructive font-medium";
  if (days <= 3) return "text-amber-600 font-medium";
  return "";
}

function Pager({ page, setPage, total, limit }: { page: number; setPage: (fn: (p: number) => number) => void; total: number; limit: number }) {
  const pageCount = Math.max(1, Math.ceil(total / limit));
  if (pageCount <= 1) return null;
  return (
    <div className="flex items-center justify-between pt-2">
      <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
        Previous
      </Button>
      <span className="text-sm text-muted-foreground">Page {page} of {pageCount}</span>
      <Button variant="outline" size="sm" disabled={page >= pageCount} onClick={() => setPage((p) => p + 1)}>
        Next
      </Button>
    </div>
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
            Extends the {sub.plan} subscription by one period from{" "}
            {fmtDate(sub.current_period_end)} (or from today if already past due) — exactly like a
            Xendit payment. Use for bank transfers or comps.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="mp-note">Note (optional)</Label>
          <Textarea
            id="mp-note"
            rows={2}
            placeholder="e.g. Paid via bank transfer, ref #12345"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button
            disabled={markPaid.isPending}
            onClick={() =>
              markPaid.mutate(
                { tenantId: sub.tenant_id, note: note.trim() },
                { onSuccess: () => onOpenChange(false) },
              )
            }
          >
            {markPaid.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Record payment
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SubscriptionRow({ sub, onMarkPaid }: { sub: AdminSubscription; onMarkPaid: (s: AdminSubscription) => void }) {
  const setStatus = useAdminSetSubscriptionStatus();
  const isActive = sub.status === "active";
  const nextStatus = isActive ? "inactive" : "active";

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-lg border p-3">
      <div className="min-w-0 flex-1">
        <p className="flex items-center gap-2 text-sm font-medium">
          <span className="truncate">{sub.tenant_name}</span>
          <Badge className={STATUS_BADGE[sub.status]}>{sub.status}</Badge>
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

      <div className="flex items-center gap-1">
        <Button variant="outline" size="sm" onClick={() => onMarkPaid(sub)}>
          <BadgeCheck className="size-4" aria-hidden />
          Mark paid
        </Button>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="ghost" size="sm" disabled={setStatus.isPending}>
              <Power className="size-4" aria-hidden />
              {isActive ? "Deactivate" : "Reactivate"}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {isActive ? "Deactivate" : "Reactivate"} {sub.tenant_name}?
              </AlertDialogTitle>
              <AlertDialogDescription>
                {isActive
                  ? "Members will be locked out until payment or reactivation. This does not change the due date."
                  : "The business regains access immediately without a payment. The due date stays as-is, so the hourly sweep may deactivate it again if it's still past due."}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                onClick={() => setStatus.mutate({ tenantId: sub.tenant_id, status: nextStatus })}
              >
                {isActive ? "Deactivate" : "Reactivate"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  );
}

function PricesCard() {
  const { data: settings } = useAdminBillingSettings();
  const update = useAdminUpdatePrices();
  const [monthly, setMonthly] = useState("");
  const [yearly, setYearly] = useState("");

  useEffect(() => {
    if (!settings) return;
    setMonthly((settings.monthly_price / 100).toString());
    setYearly((settings.yearly_price / 100).toString());
  }, [settings]);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Plan prices</CardTitle>
        <CardDescription>
          Applied to every new invoice and renewal notice platform-wide. In pesos.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="price-monthly">Monthly (PHP)</Label>
            <Input
              id="price-monthly"
              type="number"
              min="1"
              step="0.01"
              value={monthly}
              onChange={(e) => setMonthly(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="price-yearly">Yearly (PHP)</Label>
            <Input
              id="price-yearly"
              type="number"
              min="1"
              step="0.01"
              value={yearly}
              onChange={(e) => setYearly(e.target.value)}
            />
          </div>
        </div>
        <Button
          disabled={update.isPending || !monthly || !yearly}
          onClick={() =>
            update.mutate({
              monthly_price: Math.round(Number(monthly) * 100),
              yearly_price: Math.round(Number(yearly) * 100),
            })
          }
        >
          {update.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
          Save prices
        </Button>
      </CardContent>
    </Card>
  );
}

function OwnersTab() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminOwners(page);
  const total = data?.meta?.total ?? 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Owners</CardTitle>
        <CardDescription>{total} owner{total === 1 ? "" : "s"} across the platform</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {isLoading && Array.from({ length: 3 }, (_, i) => <Skeleton key={i} className="h-16 w-full" />)}

        {data?.owners.map((o) => (
          <div key={o.user_id} className="rounded-lg border p-3">
            <p className="flex items-center gap-2 text-sm font-medium">
              {o.full_name}
              {o.user_status !== "active" && <Badge variant="destructive">disabled</Badge>}
            </p>
            <p className="text-xs text-muted-foreground">
              {o.email} · joined {fmtDate(o.created_at)}
            </p>
            <div className="mt-2 flex flex-wrap gap-2">
              {o.businesses.map((b) => (
                <span key={b.tenant_id} className="inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs">
                  {b.name}
                  {b.sub_status && (
                    <Badge className={`${STATUS_BADGE[b.sub_status] ?? ""} px-1.5 py-0 text-[10px]`}>
                      {b.sub_status}
                    </Badge>
                  )}
                  {b.sub_status === "active" && (
                    <span className="text-muted-foreground">due {fmtDate(b.period_end)}</span>
                  )}
                </span>
              ))}
            </div>
          </div>
        ))}

        {data && data.owners.length === 0 && (
          <p className="py-6 text-center text-sm text-muted-foreground">No owners yet.</p>
        )}
        <Pager page={page} setPage={(fn) => setPage(fn)} total={total} limit={data?.meta?.limit ?? 20} />
      </CardContent>
    </Card>
  );
}

export default function AdminBillingPage() {
  const { data: stats } = useAdminBillingStats();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("all");
  const { data, isLoading } = useAdminSubscriptions(page, statusFilter === "all" ? "" : statusFilter);
  const [markPaidTarget, setMarkPaidTarget] = useState<AdminSubscription | null>(null);

  const total = data?.meta?.total ?? 0;

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Billing</h1>
        <p className="text-muted-foreground">Subscriptions, owners, and plan pricing</p>
      </header>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {[
          { label: "Active", value: stats ? String(stats.subs_active) : "—", icon: BadgeCheck },
          { label: "Awaiting payment", value: stats ? String(stats.subs_pending) : "—", icon: ReceiptText },
          { label: "Inactive", value: stats ? String(stats.subs_inactive) : "—", icon: Power },
          { label: "Collected (30d)", value: stats ? formatCentavos(stats.collected_30d) : "—", icon: PhilippinePeso },
        ].map((s) => (
          <Card key={s.label} className="py-4">
            <CardContent className="px-4">
              <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <s.icon className="size-3.5" aria-hidden />
                {s.label}
              </p>
              <p className="text-xl font-bold tabular-nums tracking-tight">{s.value}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Tabs defaultValue="subscriptions">
        <TabsList>
          <TabsTrigger value="subscriptions">Subscriptions</TabsTrigger>
          <TabsTrigger value="owners">Owners</TabsTrigger>
          <TabsTrigger value="prices">Prices</TabsTrigger>
        </TabsList>

        <TabsContent value="subscriptions" className="mt-4">
          <Card>
            <CardHeader className="flex-row items-center justify-between space-y-0">
              <div>
                <CardTitle className="text-base">Subscriptions</CardTitle>
                <CardDescription>{total} total — soonest due first</CardDescription>
              </div>
              <Select
                value={statusFilter}
                onValueChange={(v) => {
                  setStatusFilter(v);
                  setPage(1);
                }}
              >
                <SelectTrigger className="w-[160px]" aria-label="Filter by status">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="pending">Awaiting payment</SelectItem>
                  <SelectItem value="inactive">Inactive</SelectItem>
                </SelectContent>
              </Select>
            </CardHeader>
            <CardContent className="space-y-3">
              {isLoading &&
                Array.from({ length: 4 }, (_, i) => <Skeleton key={i} className="h-20 w-full" />)}

              {data?.subscriptions.map((s) => (
                <SubscriptionRow key={s.id} sub={s} onMarkPaid={setMarkPaidTarget} />
              ))}

              {data && data.subscriptions.length === 0 && (
                <p className="py-6 text-center text-sm text-muted-foreground">
                  No subscriptions match this filter.
                </p>
              )}
              <Pager page={page} setPage={(fn) => setPage(fn)} total={total} limit={data?.meta?.limit ?? 20} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="owners" className="mt-4">
          <OwnersTab />
        </TabsContent>

        <TabsContent value="prices" className="mt-4">
          <PricesCard />
        </TabsContent>
      </Tabs>

      <MarkPaidDialog
        sub={markPaidTarget}
        open={Boolean(markPaidTarget)}
        onOpenChange={(open) => !open && setMarkPaidTarget(null)}
      />
    </div>
  );
}
