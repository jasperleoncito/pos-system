"use client";

import { useState, type ReactNode } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Loader2, Lock, LogOut, TriangleAlert, X } from "lucide-react";

import { useAuth, useLogout } from "@/hooks/use-auth";
import { useCheckout, useSubscription } from "@/hooks/use-billing";
import { PlanCards } from "@/components/billing/plan-cards";
import { formatCentavos } from "@/lib/currency";
import { usePlans } from "@/hooks/use-billing";
import type { BillingPlan, Subscription } from "@/types/billing";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";

const DUE_SOON_MS = 3 * 24 * 60 * 60 * 1000;

/**
 * Wraps tenant pages with the billing state machine:
 * - active + due within 3 days → dismissible pay-now banner (owner)
 * - pending/inactive + owner  → forced plan+pay modal
 * - pending/inactive + staff  → full blocked screen
 */
export function SubscriptionGate({ children }: { children: ReactNode }) {
  const { auth } = useAuth();
  const { data: subscription } = useSubscription();

  const isOwner = auth?.activeTenant?.role === "owner";
  const isSuper = auth?.user.is_super_admin ?? false;

  // No data yet (or super admin browsing) — let the API's 402s rule.
  if (!subscription || isSuper) return <>{children}</>;

  if (subscription.status !== "active") {
    return isOwner ? (
      <>
        {children}
        <PlanPayModal subscription={subscription} />
      </>
    ) : (
      <BlockedScreen />
    );
  }

  const msLeft = new Date(subscription.current_period_end).getTime() - Date.now();
  const showBanner = isOwner && msLeft <= DUE_SOON_MS;

  return (
    <>
      {showBanner && <DueBanner subscription={subscription} />}
      {children}
    </>
  );
}

function PlanPayModal({ subscription }: { subscription: Subscription }) {
  const checkout = useCheckout();
  const logout = useLogout();
  const [plan, setPlan] = useState<BillingPlan>(subscription.plan);
  const isPending = subscription.status === "pending";

  const pay = () => {
    checkout.mutate(plan, {
      onSuccess: (result) => {
        window.location.href = result.invoice_url;
      },
    });
  };

  return (
    <AlertDialog open>
      <AlertDialogContent className="sm:max-w-lg">
        <AlertDialogHeader>
          <AlertDialogTitle>
            {isPending ? "Finish setting up your business" : "Your business is inactive"}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {isPending
              ? "Pick a plan and complete payment to activate your business. Everything you've set up is waiting for you."
              : "The subscription payment wasn't received by the due date. Choose a plan and pay to reactivate — all your data is safe."}
          </AlertDialogDescription>
        </AlertDialogHeader>

        <PlanCards value={plan} onChange={setPlan} disabled={checkout.isPending} />

        <Button className="w-full" size="lg" onClick={pay} disabled={checkout.isPending}>
          {checkout.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
          {isPending ? "Pay & activate" : "Pay & reactivate"}
        </Button>
        <p className="text-center text-xs text-muted-foreground">
          You&apos;ll be redirected to a secure Xendit payment page.
        </p>
        <Button
          variant="ghost"
          size="sm"
          className="mx-auto text-muted-foreground"
          onClick={() => logout()}
          disabled={checkout.isPending}
        >
          <LogOut className="size-4" aria-hidden />
          Log out
        </Button>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function BlockedScreen() {
  const logout = useLogout();
  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <div className="max-w-sm space-y-4 text-center">
        <div className="mx-auto flex size-14 items-center justify-center rounded-full bg-muted">
          <Lock className="size-7 text-muted-foreground" aria-hidden />
        </div>
        <h2 className="text-xl font-bold tracking-tight">This business is inactive</h2>
        <p className="text-sm text-muted-foreground">
          The subscription needs to be renewed before anyone can use the app. Please contact the
          business owner — this screen unlocks automatically once payment is made.
        </p>
        <Button variant="outline" size="sm" className="mx-auto" onClick={() => logout()}>
          <LogOut className="size-4" aria-hidden />
          Log out
        </Button>
      </div>
    </div>
  );
}

function DueBanner({ subscription }: { subscription: Subscription }) {
  const checkout = useCheckout();
  const { data: plans } = usePlans();
  const params = useParams<{ tenant: string }>();
  const [dismissed, setDismissed] = useState(false);
  if (dismissed) return null;

  const dueDate = new Date(subscription.current_period_end).toLocaleDateString(undefined, {
    month: "long",
    day: "numeric",
  });
  const amount =
    plans && (subscription.plan === "yearly" ? plans.yearly_price : plans.monthly_price);

  return (
    <div className="mb-4 flex flex-wrap items-center gap-3 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3">
      <TriangleAlert className="size-5 shrink-0 text-amber-600" aria-hidden />
      <p className="min-w-0 flex-1 text-sm">
        <span className="font-medium">Payment due {dueDate}.</span>{" "}
        {amount ? `Pay ${formatCentavos(amount)} to keep your business active` : "Pay to keep your business active"}
        {" — or "}
        <Link href={`/${params.tenant}/settings/billing`} className="underline underline-offset-4">
          switch plans
        </Link>
        .
      </p>
      <Button
        size="sm"
        disabled={checkout.isPending}
        onClick={() =>
          checkout.mutate(subscription.plan, {
            onSuccess: (result) => {
              window.location.href = result.invoice_url;
            },
          })
        }
      >
        {checkout.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
        Pay now
      </Button>
      <Button variant="ghost" size="icon" aria-label="Dismiss" onClick={() => setDismissed(true)}>
        <X className="size-4" aria-hidden />
      </Button>
    </div>
  );
}
