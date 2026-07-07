"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { ChefHat } from "lucide-react";

import { useAuth } from "@/hooks/use-auth";

/** Root route: sends the user to their dashboard or the login page. */
export default function Home() {
  const router = useRouter();
  const { auth, isReady } = useAuth();

  useEffect(() => {
    if (!isReady) return;
    if (auth?.user.is_super_admin) {
      router.replace("/admin/tenants");
      return;
    }
    const slug = auth?.activeTenant?.tenant_slug;
    router.replace(auth && slug ? `/${slug}/dashboard` : "/login");
  }, [isReady, auth, router]);

  return (
    <main className="flex min-h-dvh items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <div className="flex size-14 animate-pulse items-center justify-center rounded-2xl bg-primary text-primary-foreground">
          <ChefHat className="size-7" aria-hidden />
        </div>
        <p className="text-sm text-muted-foreground">Loading…</p>
      </div>
    </main>
  );
}
