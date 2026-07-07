"use client";

import { useEffect, useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { useCategories, useModifierGroups, useSaveProduct, useTaxes } from "@/hooks/use-catalog";
import { pesosToCentavos } from "@/lib/currency";
import type { Product } from "@/types/catalog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";

interface VariantRow {
  name: string;
  price_delta_pesos: string;
}

interface ProductFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  product: Product | null; // null = create
}

const NO_TAX = "none";

export function ProductFormDialog({ open, onOpenChange, product }: ProductFormDialogProps) {
  const { data: categories } = useCategories();
  const { data: taxes } = useTaxes();
  const { data: modifierGroups } = useModifierGroups();
  const saveProduct = useSaveProduct();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [sku, setSku] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [taxId, setTaxId] = useState(NO_TAX);
  const [pricePesos, setPricePesos] = useState("");
  const [costPesos, setCostPesos] = useState("");
  const [isActive, setIsActive] = useState(true);
  const [variants, setVariants] = useState<VariantRow[]>([]);
  const [selectedGroups, setSelectedGroups] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!open) return;
    setName(product?.name ?? "");
    setDescription(product?.description ?? "");
    setSku(product?.sku ?? "");
    setCategoryId(product?.category_id ?? "");
    setTaxId(product?.tax_id ?? NO_TAX);
    setPricePesos(product ? (product.base_price / 100).toString() : "");
    setCostPesos(product && product.cost_price > 0 ? (product.cost_price / 100).toString() : "");
    setIsActive(product?.is_active ?? true);
    setVariants(
      (product?.variants ?? []).map((v) => ({
        name: v.name,
        price_delta_pesos: v.price_delta ? (v.price_delta / 100).toString() : "",
      })),
    );
    setSelectedGroups(new Set((product?.modifier_groups ?? []).map((g) => g.id)));
  }, [open, product]);

  const toggleGroup = (id: string) => {
    setSelectedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const onSubmit = () => {
    const price = Number(pricePesos);
    if (!name.trim() || !categoryId || Number.isNaN(price) || price < 0) {
      toast.error("Name, category, and a valid price are required");
      return;
    }
    saveProduct.mutate(
      {
        id: product?.id,
        input: {
          category_id: categoryId,
          tax_id: taxId === NO_TAX ? null : taxId,
          name: name.trim(),
          description: description.trim(),
          sku: sku.trim(),
          base_price: pesosToCentavos(price),
          cost_price: pesosToCentavos(Number(costPesos) || 0),
          is_active: isActive,
          track_inventory: product?.track_inventory ?? false,
          sort_order: product?.sort_order ?? 0,
          variants: variants
            .filter((v) => v.name.trim())
            .map((v) => ({
              name: v.name.trim(),
              price_delta: pesosToCentavos(Number(v.price_delta_pesos) || 0),
            })),
          modifier_groups: [...selectedGroups],
        },
      },
      { onSuccess: () => onOpenChange(false) },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{product ? "Edit product" : "New product"}</DialogTitle>
          <DialogDescription>
            Prices are in pesos; stored as exact centavos.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="p-name">Name</Label>
            <Input id="p-name" value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Category</Label>
              <Select value={categoryId} onValueChange={setCategoryId}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Pick a category" />
                </SelectTrigger>
                <SelectContent>
                  {(categories ?? []).map((c) => (
                    <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Tax</Label>
              <Select value={taxId} onValueChange={setTaxId}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NO_TAX}>No tax</SelectItem>
                  {(taxes ?? []).map((t) => (
                    <SelectItem key={t.id} value={t.id}>{t.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            <div className="space-y-2">
              <Label htmlFor="p-price">Price (PHP)</Label>
              <Input
                id="p-price"
                type="number"
                min="0"
                step="0.01"
                inputMode="decimal"
                value={pricePesos}
                onChange={(e) => setPricePesos(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="p-cost">Cost (PHP)</Label>
              <Input
                id="p-cost"
                type="number"
                min="0"
                step="0.01"
                inputMode="decimal"
                value={costPesos}
                onChange={(e) => setCostPesos(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="p-sku">SKU</Label>
              <Input id="p-sku" value={sku} onChange={(e) => setSku(e.target.value)} />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="p-desc">Description</Label>
            <Textarea
              id="p-desc"
              rows={2}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          {/* Variants */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Variants</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setVariants((v) => [...v, { name: "", price_delta_pesos: "" }])}
              >
                <Plus className="size-4" aria-hidden />
                Add variant
              </Button>
            </div>
            {variants.length === 0 && (
              <p className="text-xs text-muted-foreground">
                e.g. flavors or sizes — leave empty for a simple product
              </p>
            )}
            {variants.map((v, i) => (
              <div key={i} className="flex items-center gap-2">
                <Input
                  placeholder="Variant name"
                  value={v.name}
                  onChange={(e) =>
                    setVariants((rows) =>
                      rows.map((row, j) => (j === i ? { ...row, name: e.target.value } : row)),
                    )
                  }
                />
                <Input
                  placeholder="+PHP"
                  type="number"
                  step="0.01"
                  inputMode="decimal"
                  className="w-28"
                  value={v.price_delta_pesos}
                  onChange={(e) =>
                    setVariants((rows) =>
                      rows.map((row, j) =>
                        j === i ? { ...row, price_delta_pesos: e.target.value } : row,
                      ),
                    )
                  }
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label="Remove variant"
                  onClick={() => setVariants((rows) => rows.filter((_, j) => j !== i))}
                >
                  <Trash2 className="size-4" aria-hidden />
                </Button>
              </div>
            ))}
          </div>

          {/* Modifier groups */}
          <div className="space-y-2">
            <Label>Modifier groups</Label>
            {(modifierGroups ?? []).length === 0 ? (
              <p className="text-xs text-muted-foreground">No modifier groups yet.</p>
            ) : (
              <div className="grid gap-2 sm:grid-cols-2">
                {(modifierGroups ?? []).map((g) => (
                  <label
                    key={g.id}
                    className="flex cursor-pointer items-center gap-2 rounded-lg border p-2.5 text-sm transition-colors hover:bg-accent/10"
                  >
                    <Checkbox
                      checked={selectedGroups.has(g.id)}
                      onCheckedChange={() => toggleGroup(g.id)}
                    />
                    <span className="min-w-0 flex-1 truncate">{g.name}</span>
                  </label>
                ))}
              </div>
            )}
          </div>

          <label className="flex cursor-pointer items-center justify-between rounded-lg border p-3">
            <span className="text-sm font-medium">Active on the menu</span>
            <Switch checked={isActive} onCheckedChange={setIsActive} />
          </label>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={onSubmit} disabled={saveProduct.isPending}>
            {saveProduct.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            {product ? "Save changes" : "Create product"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
