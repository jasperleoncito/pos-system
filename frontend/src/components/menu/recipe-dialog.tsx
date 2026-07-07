"use client";

import { useEffect, useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { useInventoryItems, useRecipe, useSaveRecipe } from "@/hooks/use-inventory";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";

interface RecipeRow {
  inventory_item_id: string;
  qty: string;
}

/** Recipe (BOM) editor: what one sale of the product consumes. */
export function RecipeDialog({ product, onClose }: { product: Product | null; onClose: () => void }) {
  const { data: recipe, isLoading } = useRecipe(product?.id ?? null);
  const { data: items } = useInventoryItems();
  const saveRecipe = useSaveRecipe();
  const [rows, setRows] = useState<RecipeRow[]>([]);

  useEffect(() => {
    if (recipe) {
      setRows(recipe.map((r) => ({ inventory_item_id: r.inventory_item_id, qty: String(r.qty) })));
    }
  }, [recipe]);

  if (!product) return null;

  const submit = () => {
    const valid = rows.filter((r) => r.inventory_item_id && Number(r.qty) > 0);
    if (valid.length !== rows.length) {
      toast.error("Every row needs an item and a positive quantity");
      return;
    }
    saveRecipe.mutate(
      {
        productId: product.id,
        items: valid.map((r) => ({ inventory_item_id: r.inventory_item_id, qty: Number(r.qty) })),
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Recipe — {product.name}</DialogTitle>
          <DialogDescription>Ingredients consumed per sale (deducted automatically)</DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <Skeleton className="h-32 w-full" />
        ) : (
          <div className="space-y-2">
            {rows.map((row, i) => {
              const selected = (items ?? []).find((it) => it.id === row.inventory_item_id);
              return (
                <div key={i} className="flex items-center gap-2">
                  <Select
                    value={row.inventory_item_id}
                    onValueChange={(v) =>
                      setRows((prev) => prev.map((r, j) => (j === i ? { ...r, inventory_item_id: v } : r)))
                    }
                  >
                    <SelectTrigger className="flex-1"><SelectValue placeholder="Ingredient" /></SelectTrigger>
                    <SelectContent>
                      {(items ?? []).map((it) => (
                        <SelectItem key={it.id} value={it.id}>{it.name} ({it.unit_abbr})</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Input
                    type="number"
                    min="0"
                    step="0.001"
                    placeholder={selected ? selected.unit_abbr : "qty"}
                    className="w-24"
                    aria-label="Quantity"
                    value={row.qty}
                    onChange={(e) =>
                      setRows((prev) => prev.map((r, j) => (j === i ? { ...r, qty: e.target.value } : r)))
                    }
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label="Remove ingredient"
                    onClick={() => setRows((prev) => prev.filter((_, j) => j !== i))}
                  >
                    <Trash2 className="size-4" aria-hidden />
                  </Button>
                </div>
              );
            })}

            <Button
              variant="outline"
              size="sm"
              onClick={() => setRows((prev) => [...prev, { inventory_item_id: "", qty: "" }])}
            >
              <Plus className="size-4" aria-hidden />
              Add ingredient
            </Button>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={submit} disabled={saveRecipe.isPending}>
            {saveRecipe.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Save recipe
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
