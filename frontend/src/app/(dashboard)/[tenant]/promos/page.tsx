"use client";

import { useState } from "react";
import { Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import {
  useCoupons,
  useDeleteCoupon,
  useDeleteDiscount,
  useDiscounts,
  useSaveCoupon,
  useSaveDiscount,
} from "@/hooks/use-promos";
import { formatCentavos, pesosToCentavos } from "@/lib/currency";
import type { Coupon, Discount, PromoType } from "@/types/promo";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

function promoValueLabel(type: PromoType, percent: number, amount: number): string {
  return type === "percent" ? `${percent}% off` : `${formatCentavos(amount)} off`;
}

// ---- discounts panel ----

function DiscountsPanel() {
  const { data: discounts, isLoading } = useDiscounts();
  const save = useSaveDiscount();
  const remove = useDeleteDiscount();

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Discount | null>(null);
  const [deleting, setDeleting] = useState<Discount | null>(null);
  const [name, setName] = useState("");
  const [type, setType] = useState<PromoType>("percent");
  const [value, setValue] = useState("");
  const [isActive, setIsActive] = useState(true);

  const openForm = (d: Discount | null) => {
    setEditing(d);
    setName(d?.name ?? "");
    setType(d?.type ?? "percent");
    setValue(d ? (d.type === "percent" ? String(d.percent_value) : String(d.amount_value / 100)) : "");
    setIsActive(d?.is_active ?? true);
    setFormOpen(true);
  };

  const submit = () => {
    const v = Number(value);
    if (!name.trim() || Number.isNaN(v) || v <= 0) {
      toast.error("Name and a positive value are required");
      return;
    }
    save.mutate(
      {
        id: editing?.id,
        input: {
          name: name.trim(),
          type,
          percent_value: type === "percent" ? v : 0,
          amount_value: type === "fixed" ? pesosToCentavos(v) : 0,
          requires_approval: false,
          is_active: isActive,
        },
      },
      { onSuccess: () => setFormOpen(false) },
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => openForm(null)}>
          <Plus className="size-4" aria-hidden />
          New discount
        </Button>
      </div>

      <div className="space-y-2">
        {isLoading && <Skeleton className="h-16 w-full" />}
        {(discounts ?? []).map((d) => (
          <Card key={d.id} className="py-0">
            <CardContent className="flex items-center gap-3 p-4">
              <div className="min-w-0 flex-1">
                <p className="flex items-center gap-2 text-sm font-medium">
                  {d.name}
                  {!d.is_active && <Badge variant="secondary">Inactive</Badge>}
                </p>
                <p className="text-xs text-muted-foreground">
                  {promoValueLabel(d.type, d.percent_value, d.amount_value)}
                </p>
              </div>
              <Button variant="ghost" size="icon" aria-label={`Edit ${d.name}`} onClick={() => openForm(d)}>
                <Pencil className="size-4" aria-hidden />
              </Button>
              <Button variant="ghost" size="icon" aria-label={`Delete ${d.name}`} onClick={() => setDeleting(d)}>
                <Trash2 className="size-4 text-destructive" aria-hidden />
              </Button>
            </CardContent>
          </Card>
        ))}
        {discounts && discounts.length === 0 && (
          <p className="py-10 text-center text-sm text-muted-foreground">
            No discounts yet — e.g. &quot;Senior Citizen 20%&quot;.
          </p>
        )}
      </div>

      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit discount" : "New discount"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="d-name">Name</Label>
              <Input id="d-name" placeholder="Senior Citizen" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <PromoTypeValueFields type={type} onTypeChange={setType} value={value} onValueChange={setValue} />
            <label className="flex cursor-pointer items-center justify-between rounded-lg border p-3">
              <span className="text-sm font-medium">Active</span>
              <Switch checked={isActive} onCheckedChange={setIsActive} />
            </label>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setFormOpen(false)}>Cancel</Button>
            <Button onClick={submit} disabled={save.isPending}>
              {save.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              {editing ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {deleting?.name}?</AlertDialogTitle>
            <AlertDialogDescription>Past orders keep their applied discounts.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleting) remove.mutate(deleting.id);
                setDeleting(null);
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

// ---- coupons panel ----

function CouponsPanel() {
  const { data: coupons, isLoading } = useCoupons();
  const save = useSaveCoupon();
  const remove = useDeleteCoupon();

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Coupon | null>(null);
  const [deleting, setDeleting] = useState<Coupon | null>(null);
  const [code, setCode] = useState("");
  const [type, setType] = useState<PromoType>("percent");
  const [value, setValue] = useState("");
  const [minOrder, setMinOrder] = useState("");
  const [maxUses, setMaxUses] = useState("");
  const [isActive, setIsActive] = useState(true);

  const openForm = (c: Coupon | null) => {
    setEditing(c);
    setCode(c?.code ?? "");
    setType(c?.discount_type ?? "percent");
    setValue(c ? (c.discount_type === "percent" ? String(c.percent_value) : String(c.amount_value / 100)) : "");
    setMinOrder(c && c.min_order_amount > 0 ? String(c.min_order_amount / 100) : "");
    setMaxUses(c && c.max_uses > 0 ? String(c.max_uses) : "");
    setIsActive(c?.is_active ?? true);
    setFormOpen(true);
  };

  const submit = () => {
    const v = Number(value);
    if (!code.trim() || Number.isNaN(v) || v <= 0) {
      toast.error("Code and a positive value are required");
      return;
    }
    save.mutate(
      {
        id: editing?.id,
        input: {
          code: code.trim().toUpperCase(),
          discount_type: type,
          percent_value: type === "percent" ? v : 0,
          amount_value: type === "fixed" ? pesosToCentavos(v) : 0,
          min_order_amount: pesosToCentavos(Number(minOrder) || 0),
          max_uses: Number(maxUses) || 0,
          valid_from: null,
          valid_to: null,
          is_active: isActive,
        },
      },
      { onSuccess: () => setFormOpen(false) },
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => openForm(null)}>
          <Plus className="size-4" aria-hidden />
          New coupon
        </Button>
      </div>

      <div className="space-y-2">
        {isLoading && <Skeleton className="h-16 w-full" />}
        {(coupons ?? []).map((c) => (
          <Card key={c.id} className="py-0">
            <CardContent className="flex items-center gap-3 p-4">
              <div className="min-w-0 flex-1">
                <p className="flex items-center gap-2 text-sm font-medium">
                  <span className="font-mono">{c.code}</span>
                  {!c.is_active && <Badge variant="secondary">Inactive</Badge>}
                </p>
                <p className="text-xs text-muted-foreground">
                  {promoValueLabel(c.discount_type, c.percent_value, c.amount_value)}
                  {c.min_order_amount > 0 && ` · min ${formatCentavos(c.min_order_amount)}`}
                  {" · "}
                  {c.max_uses > 0 ? `${c.uses_count}/${c.max_uses} used` : `${c.uses_count} used`}
                </p>
              </div>
              <Button variant="ghost" size="icon" aria-label={`Edit ${c.code}`} onClick={() => openForm(c)}>
                <Pencil className="size-4" aria-hidden />
              </Button>
              <Button variant="ghost" size="icon" aria-label={`Delete ${c.code}`} onClick={() => setDeleting(c)}>
                <Trash2 className="size-4 text-destructive" aria-hidden />
              </Button>
            </CardContent>
          </Card>
        ))}
        {coupons && coupons.length === 0 && (
          <p className="py-10 text-center text-sm text-muted-foreground">No coupons yet.</p>
        )}
      </div>

      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit coupon" : "New coupon"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="c-code">Code</Label>
              <Input
                id="c-code"
                placeholder="SAVE10"
                className="font-mono uppercase"
                value={code}
                onChange={(e) => setCode(e.target.value.toUpperCase())}
              />
            </div>
            <PromoTypeValueFields type={type} onTypeChange={setType} value={value} onValueChange={setValue} />
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="c-min">Min order (PHP)</Label>
                <Input id="c-min" type="number" min="0" placeholder="0" value={minOrder} onChange={(e) => setMinOrder(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="c-max">Max uses</Label>
                <Input id="c-max" type="number" min="0" placeholder="unlimited" value={maxUses} onChange={(e) => setMaxUses(e.target.value)} />
              </div>
            </div>
            <label className="flex cursor-pointer items-center justify-between rounded-lg border p-3">
              <span className="text-sm font-medium">Active</span>
              <Switch checked={isActive} onCheckedChange={setIsActive} />
            </label>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setFormOpen(false)}>Cancel</Button>
            <Button onClick={submit} disabled={save.isPending}>
              {save.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              {editing ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete coupon {deleting?.code}?</AlertDialogTitle>
            <AlertDialogDescription>It can no longer be redeemed.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleting) remove.mutate(deleting.id);
                setDeleting(null);
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

// ---- shared type/value fields ----

function PromoTypeValueFields({
  type,
  onTypeChange,
  value,
  onValueChange,
}: {
  type: PromoType;
  onTypeChange: (t: PromoType) => void;
  value: string;
  onValueChange: (v: string) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div className="space-y-2">
        <Label>Type</Label>
        <div className="grid grid-cols-2 gap-1">
          <Button
            type="button"
            variant={type === "percent" ? "default" : "outline"}
            size="sm"
            onClick={() => onTypeChange("percent")}
          >
            %
          </Button>
          <Button
            type="button"
            variant={type === "fixed" ? "default" : "outline"}
            size="sm"
            onClick={() => onTypeChange("fixed")}
          >
            ₱
          </Button>
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="promo-value">{type === "percent" ? "Percent off" : "Amount off (PHP)"}</Label>
        <Input
          id="promo-value"
          type="number"
          min="0"
          step="0.01"
          value={value}
          onChange={(e) => onValueChange(e.target.value)}
        />
      </div>
    </div>
  );
}

export default function PromosPage() {
  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Promos</h1>
        <p className="text-muted-foreground">Discounts and coupon codes for the POS</p>
      </header>

      <Tabs defaultValue="discounts" className="space-y-4">
        <TabsList className="h-11">
          <TabsTrigger value="discounts" className="min-h-9 px-4">Discounts</TabsTrigger>
          <TabsTrigger value="coupons" className="min-h-9 px-4">Coupons</TabsTrigger>
        </TabsList>
        <TabsContent value="discounts">
          <DiscountsPanel />
        </TabsContent>
        <TabsContent value="coupons">
          <CouponsPanel />
        </TabsContent>
      </Tabs>
    </div>
  );
}
