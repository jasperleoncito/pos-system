"use client";

import { Suspense, useEffect, useRef } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { CheckCircle2, Loader2, XCircle } from "lucide-react";

import { api, getApiErrorMessage } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

function VerifyEmailContent() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const fired = useRef(false);

  const verify = useMutation({
    mutationFn: async () => {
      await api.post("/auth/verify-email", { token });
    },
  });

  useEffect(() => {
    if (token && !fired.current) {
      fired.current = true;
      verify.mutate();
    }
  }, [token, verify]);

  if (!token || verify.isError) {
    return (
      <div className="space-y-6 text-center">
        <div className="mx-auto flex size-14 items-center justify-center rounded-full bg-destructive/10 text-destructive">
          <XCircle className="size-7" aria-hidden />
        </div>
        <div className="space-y-2">
          <h2 className="text-2xl font-bold tracking-tight">Verification failed</h2>
          <p className="text-sm text-muted-foreground">
            {token ? getApiErrorMessage(verify.error) : "This link is missing its token."}
          </p>
        </div>
        <Button asChild variant="outline" className="w-full">
          <Link href="/login">Back to sign in</Link>
        </Button>
      </div>
    );
  }

  if (verify.isSuccess) {
    return (
      <div className="space-y-6 text-center">
        <div className="mx-auto flex size-14 items-center justify-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400">
          <CheckCircle2 className="size-7" aria-hidden />
        </div>
        <div className="space-y-2">
          <h2 className="text-2xl font-bold tracking-tight">Email verified</h2>
          <p className="text-sm text-muted-foreground">Your email address is confirmed.</p>
        </div>
        <Button asChild className="w-full">
          <Link href="/login">Continue to sign in</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center gap-4 py-12">
      <Loader2 className="size-8 animate-spin text-muted-foreground" aria-hidden />
      <p className="text-sm text-muted-foreground">Verifying your email…</p>
    </div>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense fallback={<Skeleton className="h-64 w-full" />}>
      <VerifyEmailContent />
    </Suspense>
  );
}
