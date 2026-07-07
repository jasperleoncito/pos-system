"use client";

import { useEffect, useMemo, useState } from "react";
import { Minus, Plus } from "lucide-react";
import { toast } from "sonner";

import { formatCentavos } from "@/lib/currency";
import { buildCartLine, type CartLine } from "@/lib/pos-cart";
import { cn } from "@/lib/utils";
import type { Product } from "@/types/catalog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface ProductOptionsDialogProps {
  product: Product | null;
  onClose: () => void;
  onAdd: (line: CartLine) => void;
}

/** Variant + modifier + quantity picker, touch-first. */
export function ProductOptionsDialog({ product, onClose, onAdd }: ProductOptionsDialogProps) {
  const [variantId, setVariantId] = useState<string | undefined>();
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [qty, setQty] = useState(1);
  const [notes, setNotes] = useState("");

  useEffect(() => {
    if (product) {
      setVariantId(product.variants?.[0]?.id);
      setSelected(new Set());
      setQty(1);
      setNotes("");
    }
  }, [product]);

  const preview = useMemo(() => {
    if (!product) return 0;
    return buildCartLine(product, { variantId, modifierIds: [...selected], qty }).unitPrice * qty;
  }, [product, variantId, selected, qty]);

  if (!product) return null;

  const toggleModifier = (modId: string, groupId: string, maxSelect: number) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(modId)) {
        next.delete(modId);
        return next;
      }
      // Enforce max per group by evicting the oldest selection in it.
      const group = (product.modifier_groups ?? []).find((g) => g.id === groupId);
      const groupModIds = (group?.modifiers ?? []).map((m) => m.id).filter(Boolean) as string[];
      const inGroup = [...next].filter((id) => groupModIds.includes(id));
      if (inGroup.length >= maxSelect) {
        next.delete(inGroup[0]);
      }
      next.add(modId);
      return next;
    });
  };

  const onConfirm = () => {
    for (const group of product.modifier_groups ?? []) {
      if (!group.is_required) continue;
      const groupModIds = (group.modifiers ?? []).map((m) => m.id).filter(Boolean) as string[];
      const count = [...selected].filter((id) => groupModIds.includes(id)).length;
      if (count < group.min_select) {
        toast.error(`Pick a ${group.name}`);
        return;
      }
    }
    onAdd(buildCartLine(product, { variantId, modifierIds: [...selected], qty, notes: notes || undefined }));
    onClose();
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{product.name}</DialogTitle>
          <DialogDescription>{formatCentavos(product.base_price)} base</DialogDescription>
        </DialogHeader>

        <div className="space-y-5">
          {(product.variants?.length ?? 0) > 0 && (
            <div className="space-y-2">
              <Label>Variant</Label>
              <div className="grid grid-cols-2 gap-2">
                {product.variants!.map((v) => (
                  <button
                    key={v.id}
                    type="button"
                    onClick={() => setVariantId(v.id)}
                    className={cn(
                      "min-h-11 cursor-pointer rounded-lg border px-3 py-2 text-sm font-medium transition-colors",
                      variantId === v.id
                        ? "border-primary bg-primary text-primary-foreground"
                        : "hover:bg-accent/10",
                    )}
                  >
                    {v.name}
                    {v.price_delta > 0 && (
                      <span className="block text-xs opacity-75">
                        +{formatCentavos(v.price_delta)}
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}

          {(product.modifier_groups ?? []).map((group) => (
            <div key={group.id} className="space-y-2">
              <Label className="flex items-center gap-2">
                {group.name}
                <span className="text-xs font-normal text-muted-foreground">
                  {group.is_required ? "required" : "optional"}
                  {group.max_select > 1 && ` · up to ${group.max_select}`}
                </span>
              </Label>
              <div className="grid grid-cols-2 gap-2">
                {(group.modifiers ?? []).map((mod) => {
                  const isSelected = mod.id ? selected.has(mod.id) : false;
                  return (
                    <button
                      key={mod.id ?? mod.name}
                      type="button"
                      onClick={() => mod.id && toggleModifier(mod.id, group.id, group.max_select)}
                      className={cn(
                        "min-h-11 cursor-pointer rounded-lg border px-3 py-2 text-sm font-medium transition-colors",
                        isSelected
                          ? "border-primary bg-primary text-primary-foreground"
                          : "hover:bg-accent/10",
                      )}
                    >
                      {mod.name}
                      {mod.price_delta > 0 && (
                        <span className="block text-xs opacity-75">
                          +{formatCentavos(mod.price_delta)}
                        </span>
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}

          <div className="flex items-end justify-between gap-4">
            <div className="space-y-2">
              <Label>Quantity</Label>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  aria-label="Decrease quantity"
                  onClick={() => setQty((q) => Math.max(1, q - 1))}
                >
                  <Minus className="size-4" aria-hidden />
                </Button>
                <span className="w-10 text-center text-lg font-semibold tabular-nums">{qty}</span>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  aria-label="Increase quantity"
                  onClick={() => setQty((q) => q + 1)}
                >
                  <Plus className="size-4" aria-hidden />
                </Button>
              </div>
            </div>
            <div className="flex-1 space-y-2">
              <Label htmlFor="line-notes">Notes</Label>
              <Input
                id="line-notes"
                placeholder="e.g. no onions"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
              />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button className="min-h-12 w-full text-base" onClick={onConfirm}>
            Add {qty > 1 ? `${qty} × ` : ""}· {formatCentavos(preview)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
