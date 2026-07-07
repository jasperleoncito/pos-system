"use client";

import { useState } from "react";
import { Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { useCategories, useDeleteCategory, useSaveCategory } from "@/hooks/use-catalog";
import type { Category } from "@/types/catalog";
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
import { Textarea } from "@/components/ui/textarea";

export function CategoriesPanel({ canWrite }: { canWrite: boolean }) {
  const { data: categories, isLoading } = useCategories();
  const saveCategory = useSaveCategory();
  const deleteCategory = useDeleteCategory();

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Category | null>(null);
  const [deleting, setDeleting] = useState<Category | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [sortOrder, setSortOrder] = useState("0");
  const [isActive, setIsActive] = useState(true);

  const openForm = (category: Category | null) => {
    setEditing(category);
    setName(category?.name ?? "");
    setDescription(category?.description ?? "");
    setSortOrder(String(category?.sort_order ?? (categories?.length ?? 0)));
    setIsActive(category?.is_active ?? true);
    setFormOpen(true);
  };

  const onSubmit = () => {
    if (!name.trim()) {
      toast.error("Category name is required");
      return;
    }
    saveCategory.mutate(
      {
        id: editing?.id,
        input: {
          name: name.trim(),
          description: description.trim(),
          sort_order: Number(sortOrder) || 0,
          is_active: isActive,
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
            New category
          </Button>
        </div>
      )}

      <div className="space-y-2">
        {isLoading &&
          Array.from({ length: 4 }, (_, i) => <Skeleton key={i} className="h-16 w-full" />)}

        {(categories ?? []).map((c) => (
          <Card key={c.id} className="py-0">
            <CardContent className="flex items-center gap-3 p-4">
              <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-sm font-semibold text-muted-foreground">
                {c.sort_order + 1}
              </span>
              <div className="min-w-0 flex-1">
                <p className="flex items-center gap-2 text-sm font-medium">
                  {c.name}
                  {!c.is_active && <Badge variant="secondary">Hidden</Badge>}
                </p>
                {c.description && (
                  <p className="truncate text-xs text-muted-foreground">{c.description}</p>
                )}
              </div>
              {canWrite && (
                <div className="flex gap-1">
                  <Button variant="ghost" size="icon" aria-label={`Edit ${c.name}`} onClick={() => openForm(c)}>
                    <Pencil className="size-4" aria-hidden />
                  </Button>
                  <Button variant="ghost" size="icon" aria-label={`Delete ${c.name}`} onClick={() => setDeleting(c)}>
                    <Trash2 className="size-4 text-destructive" aria-hidden />
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        ))}

        {categories && categories.length === 0 && (
          <p className="py-10 text-center text-sm text-muted-foreground">No categories yet.</p>
        )}
      </div>

      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit category" : "New category"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="c-name">Name</Label>
              <Input id="c-name" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="c-desc">Description</Label>
              <Textarea
                id="c-desc"
                rows={2}
                placeholder="e.g. Served with Garlic Rice, Egg & Atchara"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
            <div className="grid grid-cols-2 items-end gap-4">
              <div className="space-y-2">
                <Label htmlFor="c-sort">Sort order</Label>
                <Input
                  id="c-sort"
                  type="number"
                  min="0"
                  value={sortOrder}
                  onChange={(e) => setSortOrder(e.target.value)}
                />
              </div>
              <label className="flex cursor-pointer items-center justify-between rounded-lg border p-3">
                <span className="text-sm font-medium">Active</span>
                <Switch checked={isActive} onCheckedChange={setIsActive} />
              </label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setFormOpen(false)}>Cancel</Button>
            <Button onClick={onSubmit} disabled={saveCategory.isPending}>
              {saveCategory.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              {editing ? "Save changes" : "Create category"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {deleting?.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              Categories that still contain products cannot be deleted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleting) deleteCategory.mutate(deleting.id);
                setDeleting(null);
              }}
            >
              Delete category
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
