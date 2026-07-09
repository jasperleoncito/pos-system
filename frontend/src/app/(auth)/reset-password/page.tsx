"use client";

import { Suspense } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

import { api, getApiErrorMessage } from "@/lib/api";
import { resetPasswordSchema, type ResetPasswordInput } from "@/schemas/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";

function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";
  // Invite emails link here with welcome=1 — same token flow, warmer copy.
  const isWelcome = searchParams.get("welcome") === "1";

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ResetPasswordInput>({ resolver: zodResolver(resetPasswordSchema) });

  const reset = useMutation({
    mutationFn: async (input: ResetPasswordInput) => {
      await api.post("/auth/reset-password", {
        token,
        new_password: input.new_password,
      });
    },
    onSuccess: () => {
      toast.success(
        isWelcome ? "Password set — sign in to get started" : "Password updated — please sign in again",
      );
      router.replace("/login");
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });

  if (!token) {
    return (
      <div className="space-y-4 text-center">
        <h2 className="text-2xl font-bold tracking-tight">Invalid link</h2>
        <p className="text-sm text-muted-foreground">
          This reset link is missing its token. Request a new one.
        </p>
        <Button asChild className="w-full">
          <Link href="/forgot-password">Request new link</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <header className="space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">
          {isWelcome ? "Welcome! Set your password" : "Choose a new password"}
        </h2>
        <p className="text-sm text-muted-foreground">
          {isWelcome
            ? "Pick a password for your new account, then sign in"
            : "You'll be signed out of all devices afterwards"}
        </p>
      </header>

      <form onSubmit={handleSubmit((input) => reset.mutate(input))} className="space-y-5" noValidate>
        <div className="space-y-2">
          <Label htmlFor="new_password">New password</Label>
          <Input id="new_password" type="password" autoComplete="new-password" {...register("new_password")} />
          {errors.new_password && (
            <p className="text-sm text-destructive">{errors.new_password.message}</p>
          )}
        </div>
        <div className="space-y-2">
          <Label htmlFor="confirm_password">Confirm password</Label>
          <Input id="confirm_password" type="password" autoComplete="new-password" {...register("confirm_password")} />
          {errors.confirm_password && (
            <p className="text-sm text-destructive">{errors.confirm_password.message}</p>
          )}
        </div>
        <Button type="submit" className="w-full" disabled={reset.isPending}>
          {reset.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
          {isWelcome ? "Set password" : "Update password"}
        </Button>
      </form>
    </div>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={<Skeleton className="h-64 w-full" />}>
      <ResetPasswordForm />
    </Suspense>
  );
}
