import type { ReactNode } from "react";
import { ChefHat } from "lucide-react";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="grid min-h-dvh lg:grid-cols-[1.1fr_1fr]">
      {/* Brand panel — hidden on small screens */}
      <aside className="relative hidden overflow-hidden bg-primary text-primary-foreground lg:flex lg:flex-col lg:justify-between lg:p-12">
        <div className="flex items-center gap-3">
          <div className="flex size-11 items-center justify-center rounded-xl bg-primary-foreground/10 backdrop-blur">
            <ChefHat className="size-6" aria-hidden />
          </div>
          <span className="text-lg font-semibold tracking-tight">POS System</span>
        </div>

        <div className="space-y-4">
          <h1 className="max-w-md text-4xl font-bold leading-tight tracking-tight">
            Run every business you own from one place.
          </h1>
          <p className="max-w-md text-primary-foreground/75">
            Point of sale, inventory, kitchen display, attendance, and analytics —
            isolated per business, branded your way.
          </p>
        </div>

        <p className="text-sm text-primary-foreground/60">
          Multi-tenant restaurant management platform
        </p>

        {/* Decorative layers */}
        <div
          aria-hidden
          className="pointer-events-none absolute -right-24 -top-24 size-96 rounded-full bg-primary-foreground/10 blur-3xl"
        />
        <div
          aria-hidden
          className="pointer-events-none absolute -bottom-32 -left-16 size-96 rounded-full bg-black/10 blur-3xl"
        />
      </aside>

      <main className="flex items-center justify-center p-6 sm:p-10">
        <div className="w-full max-w-sm">{children}</div>
      </main>
    </div>
  );
}
