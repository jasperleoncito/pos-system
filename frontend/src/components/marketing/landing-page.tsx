import { LandingNav } from "@/components/marketing/landing-nav";
import { Hero } from "@/components/marketing/hero";
import { Features } from "@/components/marketing/features";
import { Pricing } from "@/components/marketing/pricing";
import { FAQ } from "@/components/marketing/faq";
import { CtaBand } from "@/components/marketing/cta-band";
import { Footer } from "@/components/marketing/footer";

/** Public marketing landing page shown at the site root. */
export function LandingPage() {
  return (
    <div className="flex min-h-dvh flex-col">
      <LandingNav />
      <main className="flex-1">
        <Hero />
        <Features />
        <Pricing />
        <FAQ />
        <CtaBand />
      </main>
      <Footer />
    </div>
  );
}
