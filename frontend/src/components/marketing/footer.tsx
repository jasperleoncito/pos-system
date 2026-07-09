import Link from "next/link";
import { ChefHat } from "lucide-react";

const FACEBOOK_URL = "https://www.facebook.com/webdevbot";

/** Facebook glyph — lucide dropped brand icons, so we inline it. */
function FacebookIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} aria-hidden>
      <path d="M24 12.07C24 5.4 18.63 0 12 0S0 5.4 0 12.07C0 18.1 4.39 23.1 10.13 24v-8.44H7.08v-3.49h3.05V9.41c0-3.02 1.79-4.69 4.53-4.69 1.31 0 2.68.24 2.68.24v2.97h-1.51c-1.49 0-1.96.93-1.96 1.89v2.25h3.33l-.53 3.49h-2.8V24C19.61 23.1 24 18.1 24 12.07Z" />
    </svg>
  );
}

export function Footer() {
  return (
    <footer className="border-t bg-muted/30">
      <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6">
        <div className="flex flex-col gap-8 sm:flex-row sm:items-start sm:justify-between">
          <div className="max-w-xs space-y-3">
            <div className="flex items-center gap-2.5">
              <span className="flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground">
                <ChefHat className="size-5" aria-hidden />
              </span>
              <span className="text-base font-semibold tracking-tight">POS System</span>
            </div>
            <p className="text-sm text-muted-foreground">
              The all-in-one restaurant point of sale for Filipino businesses.
            </p>
            <a
              href={FACEBOOK_URL}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-2 rounded-md text-sm font-medium text-primary underline-offset-4 hover:underline"
            >
              <FacebookIcon className="size-4" />
              Follow us on Facebook
            </a>
          </div>

          <div className="grid grid-cols-2 gap-8 text-sm sm:gap-16">
            <div className="space-y-3">
              <p className="font-medium">Product</p>
              <ul className="space-y-2 text-muted-foreground">
                <li><a href="#features" className="hover:text-foreground">Features</a></li>
                <li><a href="#pricing" className="hover:text-foreground">Pricing</a></li>
                <li><a href="#faq" className="hover:text-foreground">FAQ</a></li>
              </ul>
            </div>
            <div className="space-y-3">
              <p className="font-medium">Get started</p>
              <ul className="space-y-2 text-muted-foreground">
                <li><Link href="/register" className="hover:text-foreground">Create account</Link></li>
                <li><Link href="/login" className="hover:text-foreground">Log in</Link></li>
                <li><a href={FACEBOOK_URL} target="_blank" rel="noreferrer" className="hover:text-foreground">Contact</a></li>
              </ul>
            </div>
          </div>
        </div>

        <div className="mt-10 border-t pt-6 text-xs text-muted-foreground">
          © 2026 POS System. Built for Filipino restaurants.
        </div>
      </div>
    </footer>
  );
}
