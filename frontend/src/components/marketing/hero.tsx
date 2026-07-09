"use client";

import Link from "next/link";
import { motion } from "motion/react";
import { ArrowRight, Check } from "lucide-react";

import { Button } from "@/components/ui/button";
import { AppMockup } from "@/components/marketing/app-mockup";

const TRUST = ["No hardware needed", "GCash & Maya ready", "VAT-inclusive receipts"];

export function Hero() {
  return (
    <section className="relative overflow-hidden">
      <div className="bg-warm-soft pointer-events-none absolute inset-0" aria-hidden />
      <div className="relative mx-auto grid max-w-6xl gap-10 px-4 pt-14 pb-16 sm:px-6 lg:grid-cols-[1.05fr_1fr] lg:items-center lg:gap-8 lg:pt-20 lg:pb-24">
        <motion.div
          className="space-y-6"
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4, ease: "easeOut" }}
        >
          <span className="inline-flex items-center gap-2 rounded-full border bg-background/60 px-3 py-1 text-xs font-medium text-muted-foreground backdrop-blur">
            <span className="size-1.5 rounded-full bg-primary" />
            Restaurant POS for Filipino businesses
          </span>

          <h1 className="font-display text-4xl font-semibold leading-[1.05] tracking-tight text-balance sm:text-5xl lg:text-6xl">
            Run your eatery, the modern way.
          </h1>

          <p className="max-w-xl text-lg text-muted-foreground text-pretty">
            Point of sale, live kitchen display, inventory, employees, loyalty, and
            analytics — one account for every business you run, branded your way.
          </p>

          <div className="flex flex-wrap items-center gap-3">
            <Button asChild size="lg" className="group">
              <Link href="/register">
                Get started
                <ArrowRight className="size-4 transition-transform group-hover:translate-x-0.5" aria-hidden />
              </Link>
            </Button>
            <Button asChild size="lg" variant="outline">
              <a href="#features">See features</a>
            </Button>
          </div>

          <ul className="flex flex-wrap gap-x-5 gap-y-2 pt-2">
            {TRUST.map((t) => (
              <li key={t} className="flex items-center gap-1.5 text-sm text-muted-foreground">
                <Check className="size-4 text-primary" aria-hidden />
                {t}
              </li>
            ))}
          </ul>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 16, scale: 0.98 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: 0.5, ease: "easeOut", delay: 0.1 }}
        >
          <AppMockup />
        </motion.div>
      </div>
    </section>
  );
}
