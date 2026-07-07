"use client";

import { useState } from "react";
import { Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { useDeleteTax, useSaveTax, useTaxes } from "@/hooks/use-catalog";
import type { Tax } from "@/types/catalog";
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

export function TaxesPanel({ canWrite }: { canWrite: boolean }) {
  const { data: taxes, isLoading } = useTaxes();
  const saveTax = useSaveTax();
  const deleteTax = useDeleteTax();

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Tax | null>(null);
  const [deleting, setDeleting] = useState<Tax | null>(null);
  const [name, setName] = useState("");
  const [rate, setRate] = useState("");
  const [isInclusive, setIsInclusive] = useState(true);
  const [isDefault, setIsDefault] = useState(false);

  const openForm = (tax: Tax | null) => {
    setEditing(tax);
    setName(tax?.name ?? "");
    setRate(tax ? String(tax.rate_percent) : "");
    setIsInclusive(tax?.is_inclusive ?? true);
    setIsDefault(tax?.is_default ?? false);
    setFormOpen(true);
  };

  const onSubmit = () => {
    const rateValue = Number(rate);
    if (!name.trim() || Number.isNaN(rateValue) || rateValue < 0 || rateValue > 100) {
      toast.error("Name and a rate between 0 and 100 are required");
      return;
    }
    saveTax.mutate(
      {
        id: editing?.id,
        input: {
          name: name.trim(),
          rate_percent: rateValue,
          is_inclusive: isInclusive,
          is_default: isDefault,
          is_active: editing?.is_active ?? true,
        },
      },
      { onSuccess: () => setFormOpen(false) },
    );
  };

  return (
    <div className="space-y-4">
      {canWrite && (
        <div className="flex justify-end">
          <Button onClick={() => openForm(null)}>
            <Plus className="size-4" aria-hidden />
            New tax
          </Button>
        </div>
      )}

      <div className="space-y-2">
        {isLoading && <Skeleton className="h-16 w-full" />}

        {(taxes ?? []).map((t) => (
          <Card key={t.id} className="py-0">
            <CardContent className="flex items-center gap-3 p-4">
              <div className="min-w-0 flex-1">
                <p className="flex items-center gap-2 text-sm font-medium">
                  {t.name}
                  {t.is_default && <Badge>Default</Badge>}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t.rate_percent}% · {t.is_inclusive ? "included in prices" : "added at checkout"}
                </p>
              </div>
              {canWrite && (
                <div className="flex gap-1">
                  <Button variant="ghost" size="icon" aria-label={`Edit ${t.name}`} onClick={() => openForm(t)}>
                    <Pencil className="size-4" aria-hidden />
                  </Button>
                  <Button variant="ghost" size="icon" aria-label={`Delete ${t.name}`} onClick={() => setDeleting(t)}>
                    <Trash2 className="size-4 text-destructive" aria-hidden />
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        ))}

        {taxes && taxes.length === 0 && (
          <p className="py-10 text-center text-sm text-muted-foreground">No taxes configured.</p>
        )}
      </div>

      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit tax" : "New tax"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-[1fr_8rem] gap-4">
              <div className="space-y-2">
                <Label htmlFor="t-name">Name</Label>
                <Input id="t-name" placeholder="VAT 12% (inclusive)" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="t-rate">Rate %</Label>
                <Input
                  id="t-rate"
                  type="number"
                  min="0"
                  max="100"
                  step="0.01"
                  value={rate}
                  onChange={(e) => setRate(e.target.value)}
                />
              </div>
            </div>
            <label className="flex cursor-pointer items-center justify-between rounded-lg border p-3">
              <span className="text-sm font-medium">Included in menu prices</span>
              <Switch checked={isInclusive} onCheckedChange={setIsInclusive} />
            </label>
            <label className="flex cursor-pointer items-center justify-between rounded-lg border p-3">
              <span className="text-sm font-medium">Default for new products</span>
              <Switch checked={isDefault} onCheckedChange={setIsDefault} />
            </label>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setFormOpen(false)}>Cancel</Button>
            <Button onClick={onSubmit} disabled={saveTax.isPending}>
              {saveTax.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              {editing ? "Save changes" : "Create tax"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {deleting?.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              Products using this tax fall back to no tax.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleting) deleteTax.mutate(deleting.id);
                setDeleting(null);
              }}
            >
              Delete tax
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
