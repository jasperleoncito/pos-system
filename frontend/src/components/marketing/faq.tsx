"use client";

import { useState } from "react";
import { ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";

const FAQS = [
  {
    q: "What payment methods can I accept?",
    a: "Cash, GCash, Maya, cards, and bank transfers — including mixed and split payments across a single order. Every sale can print an 80mm thermal receipt.",
  },
  {
    q: "Are receipts VAT-compliant?",
    a: "Yes. Tax is computed as VAT-inclusive (12% by default, editable per business) and shown on the receipt, so your totals match what customers pay.",
  },
  {
    q: "Can I run more than one business?",
    a: "One account manages unlimited businesses. Each is fully isolated with its own menu, staff, inventory, and reports, and re-themed to its own brand colors and logo.",
  },
  {
    q: "How does billing work?",
    a: "Pick a monthly or yearly plan when you sign up and pay securely online to activate. You'll get a reminder before each renewal, and you can switch plans anytime.",
  },
  {
    q: "Is my data separate from other restaurants?",
    a: "Completely. Every record is scoped to your business and protected by role-based access — owner, manager, cashier, kitchen, and employee each see only what they should.",
  },
  {
    q: "What devices do I need?",
    a: "Any modern phone, tablet, or computer with a browser. The interface is touch-first with large tap targets, so a tablet at the counter works great — no special hardware.",
  },
];

export function FAQ() {
  const [open, setOpen] = useState<number | null>(0);

  return (
    <section id="faq" className="mx-auto max-w-3xl scroll-mt-20 px-4 py-16 sm:px-6 lg:py-24">
      <div className="text-center">
        <h2 className="font-display text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
          Questions, answered
        </h2>
        <p className="mt-3 text-muted-foreground">Everything you need to know before you start.</p>
      </div>

      <div className="mt-10 divide-y rounded-xl border bg-card">
        {FAQS.map((item, i) => {
          const isOpen = open === i;
          return (
            <div key={item.q}>
              <button
                type="button"
                onClick={() => setOpen(isOpen ? null : i)}
                aria-expanded={isOpen}
                className="flex w-full items-center justify-between gap-4 px-5 py-4 text-left"
              >
                <span className="font-medium">{item.q}</span>
                <ChevronDown
                  className={cn(
                    "size-5 shrink-0 text-muted-foreground transition-transform",
                    isOpen && "rotate-180",
                  )}
                  aria-hidden
                />
              </button>
              <div
                className={cn(
                  "grid transition-all duration-200 ease-out",
                  isOpen ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0",
                )}
              >
                <div className="overflow-hidden">
                  <p className="px-5 pb-4 text-sm text-muted-foreground">{item.a}</p>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}
