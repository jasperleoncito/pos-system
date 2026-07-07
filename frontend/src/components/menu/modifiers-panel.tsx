"use client";

import { useState } from "react";
import { Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import {
  useDeleteModifierGroup,
  useModifierGroups,
  useSaveModifierGroup,
} from "@/hooks/use-catalog";
import { formatCentavos, pesosToCentavos } from "@/lib/currency";
import type { ModifierGroup } from "@/types/catalog";
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
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";

interface OptionRow {
  name: string;
  price_delta_pesos: string;
}

export function ModifiersPanel({ canWrite }: { canWrite: boolean }) {
  const { data: groups, isLoading } = useModifierGroups();
  const saveGroup = useSaveModifierGroup();
  const deleteGroup = useDeleteModifierGroup();

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<ModifierGroup | null>(null);
  const [deleting, setDeleting] = useState<ModifierGroup | null>(null);
  const [name, setName] = useState("");
  const [isRequired, setIsRequired] = useState(false);
  const [maxSelect, setMaxSelect] = useState("1");
  const [options, setOptions] = useState<OptionRow[]>([]);

  const openForm = (group: ModifierGroup | null) => {
    setEditing(group);
    setName(group?.name ?? "");
    setIsRequired(group?.is_required ?? false);
    setMaxSelect(String(group?.max_select ?? 1));
    setOptions(
      (group?.modifiers ?? []).map((m) => ({
        name: m.name,
        price_delta_pesos: m.price_delta ? (m.price_delta / 100).toString() : "",
      })),
    );
    setFormOpen(true);
  };

  const onSubmit = () => {
    const validOptions = options.filter((o) => o.name.trim());
    if (!name.trim() || validOptions.length === 0) {
      toast.error("Name and at least one option are required");
      return;
    }
    const max = Math.max(1, Number(maxSelect) || 1);
    saveGroup.mutate(
      {
        id: editing?.id,
        input: {
          name: name.trim(),
          min_select: isRequired ? 1 : 0,
          max_select: max,
          is_required: isRequired,
          sort_order: editing?.sort_order ?? (groups?.length ?? 0),
          modifiers: validOptions.map((o) => ({
            name: o.name.trim(),
            price_delta: pesosToCentavos(Number(o.price_delta_pesos) || 0),
          })),
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
            New modifier group
          </Button>
        </div>
      )}

      <div className="grid gap-3 lg:grid-cols-2">
        {isLoading &&
          Array.from({ length: 2 }, (_, i) => <Skeleton key={i} className="h-32 w-full" />)}

        {(groups ?? []).map((g) => (
          <Card key={g.id} className="py-0">
            <CardContent className="space-y-3 p-4">
              <div className="flex items-center gap-2">
                <p className="flex-1 text-sm font-semibold">{g.name}</p>
                {g.is_required ? (
                  <Badge>Required</Badge>
                ) : (
                  <Badge variant="secondary">Optional</Badge>
                )}
                {canWrite && (
                  <div className="flex gap-1">
                    <Button variant="ghost" size="icon" aria-label={`Edit ${g.name}`} onClick={() => openForm(g)}>
                      <Pencil className="size-4" aria-hidden />
                    </Button>
                    <Button variant="ghost" size="icon" aria-label={`Delete ${g.name}`} onClick={() => setDeleting(g)}>
                      <Trash2 className="size-4 text-destructive" aria-hidden />
                    </Button>
                  </div>
                )}
              </div>
              <div className="flex flex-wrap gap-1.5">
                {(g.modifiers ?? []).map((m) => (
                  <Badge key={m.id ?? m.name} variant="outline" className="font-normal">
                    {m.name}
                    {m.price_delta > 0 && (
                      <span className="text-muted-foreground"> +{formatCentavos(m.price_delta)}</span>
                    )}
                  </Badge>
                ))}
              </div>
            </CardContent>
          </Card>
        ))}

        {groups && groups.length === 0 && (
          <p className="col-span-full py-10 text-center text-sm text-muted-foreground">
            No modifier groups yet — e.g. &quot;Choice of Side Dish&quot;.
          </p>
        )}
      </div>

      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit modifier group" : "New modifier group"}</DialogTitle>
            <DialogDescription>
              Options a cashier picks when adding the product — sides, drinks, dips.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="g-name">Group name</Label>
              <Input
                id="g-name"
                placeholder="Choice of Side Dish"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="grid grid-cols-2 items-end gap-4">
              <div className="space-y-2">
                <Label htmlFor="g-max">Max selections</Label>
                <Input
                  id="g-max"
                  type="number"
                  min="1"
                  value={maxSelect}
                  onChange={(e) => setMaxSelect(e.target.value)}
                />
              </div>
              <label className="flex cursor-pointer items-center justify-between rounded-lg border p-3">
                <span className="text-sm font-medium">Required</span>
                <Switch checked={isRequired} onCheckedChange={setIsRequired} />
              </label>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>Options</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setOptions((o) => [...o, { name: "", price_delta_pesos: "" }])}
                >
                  <Plus className="size-4" aria-hidden />
                  Add option
                </Button>
              </div>
              {options.map((o, i) => (
                <div key={i} className="flex items-center gap-2">
                  <Input
                    placeholder="Option name"
                    value={o.name}
                    onChange={(e) =>
                      setOptions((rows) =>
                        rows.map((row, j) => (j === i ? { ...row, name: e.target.value } : row)),
                      )
                    }
                  />
                  <Input
                    placeholder="+PHP"
                    type="number"
                    step="0.01"
                    inputMode="decimal"
                    className="w-24"
                    value={o.price_delta_pesos}
                    onChange={(e) =>
                      setOptions((rows) =>
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
                    aria-label="Remove option"
                    onClick={() => setOptions((rows) => rows.filter((_, j) => j !== i))}
                  >
                    <Trash2 className="size-4" aria-hidden />
                  </Button>
                </div>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setFormOpen(false)}>Cancel</Button>
            <Button onClick={onSubmit} disabled={saveGroup.isPending}>
              {saveGroup.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              {editing ? "Save changes" : "Create group"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {deleting?.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              The group is removed from all products that use it.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleting) deleteGroup.mutate(deleting.id);
                setDeleting(null);
              }}
            >
              Delete group
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
