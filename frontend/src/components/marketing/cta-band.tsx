"use client";

import Link from "next/link";
import { motion } from "motion/react";
import { ArrowRight } from "lucide-react";

import { Button } from "@/components/ui/button";

export function CtaBand() {
  return (
    <section className="mx-auto max-w-6xl px-4 py-16 sm:px-6 lg:py-20">
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-80px" }}
        transition={{ duration: 0.4, ease: "easeOut" }}
        className="bg-warm-hero warm-grain relative overflow-hidden rounded-3xl px-6 py-14 text-center sm:px-12"
      >
        <div className="relative mx-auto max-w-xl space-y-5">
          <h2 className="font-display text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            Ready to run a smarter kitchen?
          </h2>
          <p className="text-primary-foreground/85 text-pretty">
            Set up your business in minutes and take your first order today.
          </p>
          <div className="flex flex-wrap justify-center gap-3 pt-1">
            <Button asChild size="lg" variant="secondary" className="group">
              <Link href="/register">
                Get started
                <ArrowRight className="size-4 transition-transform group-hover:translate-x-0.5" aria-hidden />
              </Link>
            </Button>
            <Button
              asChild
              size="lg"
              variant="outline"
              className="border-white/40 bg-transparent text-primary-foreground hover:bg-white/10 hover:text-primary-foreground"
            >
              <Link href="/login">I already have an account</Link>
            </Button>
          </div>
        </div>
      </motion.div>
    </section>
  );
}
