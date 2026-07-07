"use client";

import { useState } from "react";
import { Loader2, TicketPercent, X } from "lucide-react";

import { useDiscounts, useValidateCoupon } from "@/hooks/use-promos";
import { formatCentavos } from "@/lib/currency";
import { applyPromo, type Discount } from "@/types/promo";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

/** Promo state carried by the POS cart. */
export interface AppliedPromo {
  discount?: Discount;
  couponCode?: string;
  couponAmount?: number; // validated server-side, preview only
}

interface PromoDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  subtotal: number;
  applied: AppliedPromo;
  onApply: (promo: AppliedPromo) => void;
}

export function PromoDialog({ open, onOpenChange, subtotal, applied, onApply }: PromoDialogProps) {
  const { data: discounts } = useDiscounts();
  const validateCoupon = useValidateCoupon();
  const [code, setCode] = useState("");

  const activeDiscounts = (discounts ?? []).filter((d) => d.is_active);

  const applyCoupon = () => {
    if (!code.trim()) return;
    validateCoupon.mutate(
      { code: code.trim(), subtotal },
      {
        onSuccess: (data) => {
          onApply({ ...applied, couponCode: data.coupon.code, couponAmount: data.discount });
          setCode("");
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Discounts & coupons</DialogTitle>
          <DialogDescription>Applied to the whole order</DialogDescription>
        </DialogHeader>

        <div className="space-y-5">
          <div className="space-y-2">
            <Label>Discount</Label>
            {activeDiscounts.length === 0 ? (
              <p className="text-xs text-muted-foreground">No discounts configured.</p>
            ) : (
              <div className="grid grid-cols-2 gap-2">
                {activeDiscounts.map((d) => {
                  const isSelected = applied.discount?.id === d.id;
                  return (
                    <button
                      key={d.id}
                      type="button"
                      onClick={() =>
                        onApply({ ...applied, discount: isSelected ? undefined : d })
                      }
                      className={cn(
                        "min-h-11 cursor-pointer rounded-lg border px-3 py-2 text-left text-sm font-medium transition-colors",
                        isSelected
                          ? "border-primary bg-primary text-primary-foreground"
                          : "hover:bg-accent/10",
                      )}
                    >
                      {d.name}
                      <span className="block text-xs opacity-75">
                        {d.type === "percent"
                          ? `${d.percent_value}% off`
                          : `${formatCentavos(d.amount_value)} off`}
                        {" · −"}
                        {formatCentavos(applyPromo(d.type, d.percent_value, d.amount_value, subtotal))}
                      </span>
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="coupon-code">Coupon code</Label>
            {applied.couponCode ? (
              <div className="flex items-center justify-between rounded-lg border border-primary/40 bg-primary/5 p-3">
                <div>
                  <p className="font-mono text-sm font-semibold">{applied.couponCode}</p>
                  <p className="text-xs text-muted-foreground">
                    −{formatCentavos(applied.couponAmount ?? 0)}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Remove coupon"
                  onClick={() => onApply({ ...applied, couponCode: undefined, couponAmount: undefined })}
                >
                  <X className="size-4" aria-hidden />
                </Button>
              </div>
            ) : (
              <div className="flex gap-2">
                <Input
                  id="coupon-code"
                  placeholder="SAVE10"
                  className="font-mono uppercase"
                  value={code}
                  onChange={(e) => setCode(e.target.value.toUpperCase())}
                  onKeyDown={(e) => e.key === "Enter" && applyCoupon()}
                />
                <Button onClick={applyCoupon} disabled={validateCoupon.isPending || !code.trim()}>
                  {validateCoupon.isPending ? (
                    <Loader2 className="size-4 animate-spin" aria-hidden />
                  ) : (
                    <TicketPercent className="size-4" aria-hidden />
                  )}
                  Apply
                </Button>
              </div>
            )}
          </div>

          <Button className="w-full" variant="outline" onClick={() => onOpenChange(false)}>
            Done
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
