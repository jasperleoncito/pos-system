"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { CheckCircle2, Loader2, TriangleAlert } from "lucide-react";
import { toast } from "sonner";

import { useAuth } from "@/hooks/use-auth";
import { useSubscription } from "@/hooks/use-billing";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

const POLL_MS = 3_000;
const GIVE_UP_MS = 60_000;

/**
 * Xendit redirects here after checkout. The webhook activates the
 * subscription asynchronously, so we poll until it flips (or time out).
 */
function BillingReturn() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const failed = searchParams.get("status") === "failed";
  const { auth, isReady } = useAuth();
  const [timedOut, setTimedOut] = useState(false);

  const { data: subscription } = useSubscription(failed || timedOut ? undefined : POLL_MS);
  const slug = auth?.activeTenant?.tenant_slug;
  const isActive = subscription?.status === "active";

  useEffect(() => {
    if (!isActive || !slug) return;
    toast.success("Payment received — your business is active!");
    router.replace(`/${slug}/dashboard`);
  }, [isActive, slug, router]);

  useEffect(() => {
    if (failed) return;
    const timer = setTimeout(() => setTimedOut(true), GIVE_UP_MS);
    return () => clearTimeout(timer);
  }, [failed]);

  if (!isReady) return <Skeleton className="h-48 w-full max-w-md" />;

  if (!auth) {
    return (
      <div className="space-y-4 text-center">
        <h1 className="text-2xl font-bold tracking-tight">Almost there</h1>
        <p className="text-sm text-muted-foreground">
          Sign in to check your payment status.
        </p>
        <Button asChild>
          <Link href="/login">Sign in</Link>
        </Button>
      </div>
    );
  }

  if (failed) {
    return (
      <div className="space-y-4 text-center">
        <TriangleAlert className="mx-auto size-10 text-destructive" aria-hidden />
        <h1 className="text-2xl font-bold tracking-tight">Payment not completed</h1>
        <p className="text-sm text-muted-foreground">
          No worries — you can pay any time from your dashboard.
        </p>
        <Button asChild>
          <Link href={slug ? `/${slug}/dashboard` : "/login"}>Go to dashboard</Link>
        </Button>
      </div>
    );
  }

  if (isActive) {
    return (
      <div className="space-y-4 text-center">
        <CheckCircle2 className="mx-auto size-10 text-emerald-600" aria-hidden />
        <h1 className="text-2xl font-bold tracking-tight">Payment received!</h1>
        <p className="text-sm text-muted-foreground">Taking you to your dashboard…</p>
      </div>
    );
  }

  if (timedOut) {
    return (
      <div className="space-y-4 text-center">
        <TriangleAlert className="mx-auto size-10 text-amber-500" aria-hidden />
        <h1 className="text-2xl font-bold tracking-tight">Payment not confirmed yet</h1>
        <p className="text-sm text-muted-foreground">
          Payments can take a moment to settle. Check your dashboard — if it&apos;s still locked,
          you can retry payment from there.
        </p>
        <div className="flex justify-center gap-2">
          <Button variant="outline" onClick={() => setTimedOut(false)}>
            Keep checking
          </Button>
          <Button asChild>
            <Link href={slug ? `/${slug}/dashboard` : "/login"}>Go to dashboard</Link>
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4 text-center">
      <Loader2 className="mx-auto size-10 animate-spin text-primary" aria-hidden />
      <h1 className="text-2xl font-bold tracking-tight">Confirming your payment…</h1>
      <p className="text-sm text-muted-foreground">
        This usually takes a few seconds. Don&apos;t close this page.
      </p>
    </div>
  );
}

export default function BillingReturnPage() {
  return (
    <main className="flex min-h-dvh items-center justify-center p-6">
      <div className="w-full max-w-md">
        <Suspense fallback={<Skeleton className="h-48 w-full" />}>
          <BillingReturn />
        </Suspense>
      </div>
    </main>
  );
}
