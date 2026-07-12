"use client";

import { useState } from "react";
import { ExternalLink, Loader2 } from "lucide-react";

import { useBillingPayments, useCheckout, usePlans, useSubscription } from "@/hooks/use-billing";
import { formatCentavos } from "@/lib/currency";
import type { BillingPlan } from "@/types/billing";
import { PlanCards } from "@/components/billing/plan-cards";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

const STATUS_BADGE: Record<string, { label: string; className: string }> = {
  active: { label: "Active", className: "bg-emerald-600 text-white" },
  pending: { label: "Awaiting payment", className: "bg-amber-500 text-white" },
  inactive: { label: "Inactive", className: "bg-destructive text-white" },
};

function fmtDate(value?: string | null) {
  if (!value) return "—";
  return new Date(value).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

export default function BillingPage() {
  const { data: subscription, isLoading } = useSubscription();
  const { data: plans } = usePlans();
  const checkout = useCheckout();
  const [page, setPage] = useState(1);
  const { data: history } = useBillingPayments(page);
  const [plan, setPlan] = useState<BillingPlan | null>(null);

  const selectedPlan = plan ?? subscription?.plan ?? "monthly";
  const badge = subscription ? STATUS_BADGE[subscription.status] : undefined;

  const total = history?.meta?.total ?? 0;
  const limit = history?.meta?.limit ?? 20;
  const pageCount = Math.max(1, Math.ceil(total / limit));

  const pay = () => {
    checkout.mutate(
      { plan: selectedPlan },
      {
        onSuccess: (result) => {
          window.location.href = result.invoice_url;
        },
      },
    );
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Billing</h1>
        <p className="text-muted-foreground">Your subscription, plan, and payment history</p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            Subscription
            {badge && <Badge className={badge.className}>{badge.label}</Badge>}
          </CardTitle>
          <CardDescription>
            {subscription?.status === "active"
              ? `Paid through ${fmtDate(subscription.current_period_end)}`
              : subscription?.status === "pending"
                ? "Complete your first payment to activate the business"
                : "Pay to reactivate your business"}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading && <Skeleton className="h-24 w-full" />}

          {subscription && (
            <>
              <div className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
                <div>
                  <p className="text-xs text-muted-foreground">Current plan</p>
                  <p className="font-medium capitalize">{subscription.plan}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">
                    {subscription.status === "active" ? "Next payment due" : "Was due"}
                  </p>
                  <p className="font-medium">{fmtDate(subscription.current_period_end)}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Renewal price</p>
                  <p className="font-medium tabular-nums">
                    {plans
                      ? formatCentavos(
                          subscription.plan === "yearly" ? plans.yearly_price : plans.monthly_price,
                        )
                      : "—"}
                  </p>
                </div>
              </div>

              <div className="space-y-2 border-t pt-4">
                <p className="text-sm font-medium">
                  {subscription.status === "active" ? "Renew early or switch plan" : "Choose a plan"}
                </p>
                <PlanCards value={selectedPlan} onChange={setPlan} disabled={checkout.isPending} />
                <Button className="w-full" onClick={pay} disabled={checkout.isPending}>
                  {checkout.isPending ? (
                    <Loader2 className="size-4 animate-spin" aria-hidden />
                  ) : (
                    <ExternalLink className="size-4" aria-hidden />
                  )}
                  Pay {plans
                    ? formatCentavos(
                        selectedPlan === "yearly" ? plans.yearly_price : plans.monthly_price,
                      )
                    : ""} via Xendit
                </Button>
                <p className="text-xs text-muted-foreground">
                  Paying before the due date extends your current period — you never lose days.
                </p>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Payment history</CardTitle>
          <CardDescription>{total} payment{total === 1 ? "" : "s"}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {history?.payments.map((p) => (
            <div key={p.id} className="flex flex-wrap items-center gap-3 rounded-lg border p-3 text-sm">
              <div className="min-w-0 flex-1">
                <p className="font-medium tabular-nums">{formatCentavos(p.amount)}</p>
                <p className="text-xs text-muted-foreground">
                  <span className="capitalize">{p.plan}</span> ·{" "}
                  {p.method === "manual" ? "recorded by admin" : p.payment_channel || "Xendit"} ·{" "}
                  {fmtDate(p.paid_at ?? p.created_at)}
                </p>
              </div>
              <Badge
                variant={p.status === "paid" ? "default" : p.status === "pending" ? "secondary" : "outline"}
                className={p.status === "paid" ? "bg-emerald-600" : undefined}
              >
                {p.status}
              </Badge>
              {p.status === "pending" && p.xendit_invoice_url && (
                <Button asChild variant="outline" size="sm">
                  <a href={p.xendit_invoice_url} target="_blank" rel="noreferrer">
                    Open invoice
                  </a>
                </Button>
              )}
            </div>
          ))}

          {history && history.payments.length === 0 && (
            <p className="py-6 text-center text-sm text-muted-foreground">No payments yet.</p>
          )}

          {pageCount > 1 && (
            <div className="flex items-center justify-between pt-2">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                Previous
              </Button>
              <span className="text-sm text-muted-foreground">
                Page {page} of {pageCount}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= pageCount}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
