"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";

import { registerSchema, type RegisterInput } from "@/schemas/auth";
import { useRegister } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

function toSlug(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function RegisterPage() {
  const router = useRouter();
  const registerMutation = useRegister();
  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<RegisterInput>({ resolver: zodResolver(registerSchema) });

  const onSubmit = handleSubmit((values) => {
    const { confirm_password, ...input } = values;
    void confirm_password;
    registerMutation.mutate(input, {
      onSuccess: (result) => {
        const slug = result.active_tenant?.tenant_slug;
        router.replace(slug ? `/${slug}/dashboard` : "/login");
      },
    });
  });

  return (
    <div className="space-y-8">
      <header className="space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">Create your account</h2>
        <p className="text-sm text-muted-foreground">
          Set up your first business — you can add more later
        </p>
      </header>

      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        <div className="space-y-2">
          <Label htmlFor="full_name">Your name</Label>
          <Input id="full_name" autoComplete="name" placeholder="Juan dela Cruz" {...register("full_name")} />
          {errors.full_name && <p className="text-sm text-destructive">{errors.full_name.message}</p>}
        </div>

        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input id="email" type="email" autoComplete="email" placeholder="you@business.com" {...register("email")} />
          {errors.email && <p className="text-sm text-destructive">{errors.email.message}</p>}
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="password">Password</Label>
            <Input id="password" type="password" autoComplete="new-password" {...register("password")} />
            {errors.password && <p className="text-sm text-destructive">{errors.password.message}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="confirm_password">Confirm</Label>
            <Input id="confirm_password" type="password" autoComplete="new-password" {...register("confirm_password")} />
            {errors.confirm_password && (
              <p className="text-sm text-destructive">{errors.confirm_password.message}</p>
            )}
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="business_name">Business name</Label>
          <Input
            id="business_name"
            placeholder="Teresa's Eatery"
            {...register("business_name")}
            onChange={(e) => {
              register("business_name").onChange(e);
              setValue("business_slug", toSlug(e.target.value));
            }}
          />
          {errors.business_name && (
            <p className="text-sm text-destructive">{errors.business_name.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="business_slug">Business URL</Label>
          <Input id="business_slug" placeholder="teresas-eatery" {...register("business_slug")} />
          {errors.business_slug && (
            <p className="text-sm text-destructive">{errors.business_slug.message}</p>
          )}
        </div>

        <Button type="submit" className="w-full" disabled={registerMutation.isPending}>
          {registerMutation.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
          Create account
        </Button>
      </form>

      <p className="text-center text-sm text-muted-foreground">
        Already have an account?{" "}
        <Link href="/login" className="text-primary underline-offset-4 hover:underline">
          Sign in
        </Link>
      </p>
    </div>
  );
}
