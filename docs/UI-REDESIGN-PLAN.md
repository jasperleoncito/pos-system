# C6 — Login redesign + marketing landing page ✅ DONE

Redesign the login (owner disliked the current look) and build a real public landing page (root `/` currently only redirects). See `docs/PLAN.md` for the phase log; this file tracks the C6 build.

**Status: shipped.** tsc + lint clean; browser-verified. Note during build: lucide-react in this repo dropped brand icons, so the Facebook glyph is an inlined SVG in `footer.tsx`.

## Confirmed decisions

- **Visual style:** appetizing warm — red→amber gradients, rounded cards, friendly/food-forward, theme-aware (light + dark).
- **Product name:** stays "POS System".
- **Login:** restyled split-screen (also upgrades register/forgot/reset — shared shell).
- **Landing sections:** hero + app mockup, feature showcase, pricing (live ₱800/₱8,000), FAQ, footer.
- **Footer links owner's Facebook:** https://www.facebook.com/webdevbot

## Reuse (don't rebuild)

- `motion` ^12.42.2 (`motion/react`) — short easeOut, ~8px y+fade; `whileInView` reveals; respect `prefers-reduced-motion`.
- `usePlans()` (`hooks/use-billing.ts` → public `GET /billing/plans`) + `formatCentavos` (`lib/currency.ts`) for live pricing.
- shadcn `ui/*` primitives (button asChild/lg, card, badge, separator). FAQ = tiny local accordion (no new dep).
- Tokens in `globals.css` (`--primary` red, `--accent` amber, `.dark` set); `useAuth()` for the "go to dashboard" target; `ChefHat` lockup as wordmark.

## Checklist

### Foundation
- [x] Add Fraunces display font via `next/font/google` in `app/layout.tsx` → `--font-display`; wire `font-display` utility in `globals.css` `@theme inline`. Headlines only (body stays Geist).
- [x] Add `.bg-warm-hero` (red→amber, theme-aware) + any small gradient helper to `globals.css`.

### Login / auth redesign
- [x] Rebuild `(auth)/layout.tsx` brand panel: warm gradient, refined ChefHat lockup, `font-display` headline, 3 value-props w/ icons, soft glow (drop flat blobs), subtle motion; widen right column to `max-w-md` with a `bg-card` form surface.
- [x] Polish `(auth)/login/page.tsx`: `font-display` heading, input leading icons (Mail/Lock), show/hide password toggle, arrow+Loader2 button; keep RHF+zod + redirect logic.
- [x] Add logged-in guard in auth layout (bounce authed users to dashboard / admin).
- [x] Verify register (plan cards)/forgot/reset still lay out well in the new shell.

### Landing page
- [x] `app/page.tsx` → render `<LandingPage/>` (drop forced redirect; nav adapts instead).
- [x] `components/marketing/landing-nav.tsx` — sticky blur header, anchors, theme toggle, auth-aware CTAs (Dashboard vs Log in/Get started).
- [x] `hero.tsx` — `.bg-warm-hero`, display headline + subcopy (PH/₱/GCash/Maya/VAT), CTAs, motion entrance.
- [x] `app-mockup.tsx` — stylized POS terminal built from divs (product tiles + cart + ₱ total) matching app vocabulary.
- [x] `features.tsx` — bento grid of 8 shipped features (POS, payments+receipts, KDS, inventory+recipes, analytics, employees+attendance, loyalty, multi-tenant), lucide icons, whileInView.
- [x] `pricing.tsx` — two cards from `usePlans()`, highlight yearly "2 months free", CTA → /register.
- [x] `faq.tsx` — local accordion: payments, VAT receipts, multiple businesses, billing, data isolation, devices.
- [x] `footer.tsx` — links + Facebook (https://www.facebook.com/webdevbot), © 2026 POS System.
- [x] `landing-page.tsx` composes all; sections wrap `max-w-6xl mx-auto px-…`.

### Finalize
- [x] `npx tsc --noEmit` + `npm run lint` clean.
- [x] Rebuild frontend; browser check 375/768/1440 in light + dark (landing incl. live pricing that reflects an /admin/billing price edit; FAQ toggles; FB link; login show/hide + real login + logged-in bounce; register/forgot/reset intact; no h-overflow; reduced-motion honored).
- [x] Update `docs/PLAN.md` (C6 entry) + this checklist; conventional commit.

## Verification recipes

- Live pricing: edit prices in `/admin/billing`, reload `/` → pricing section reflects new ₱ (proves `usePlans` wiring).
- Auth guard: log in, visit `/login` → bounced to dashboard.
- Public access: landing loads without a token (public `/billing/plans`).
