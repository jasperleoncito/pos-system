"use client";

import { useState } from "react";
import { ArrowDownUp, History, Loader2, Pencil, Plus, Search } from "lucide-react";
import { toast } from "sonner";

import {
  useApplyMovement,
  useInventoryItems,
  useMovements,
  useSaveInventoryItem,
  useUnits,
  type InventoryItem,
} from "@/hooks/use-inventory";
import { useAuth } from "@/hooks/use-auth";
import { can } from "@/lib/rbac";
import { cn } from "@/lib/utils";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function StockBadge({ item }: { item: InventoryItem }) {
  if (item.current_stock <= 0) {
    return <Badge variant="destructive">Out of stock</Badge>;
  }
  if (item.current_stock <= item.reorder_level) {
    return <Badge className="bg-amber-500">Low</Badge>;
  }
  return <Badge className="bg-emerald-600">OK</Badge>;
}

const MOVE_TYPES = [
  { value: "stock_in", label: "Stock in" },
  { value: "stock_out", label: "Stock out" },
  { value: "adjustment", label: "Adjustment" },
  { value: "waste", label: "Waste" },
];

export default function InventoryPage() {
  const { auth } = useAuth();
  const canWrite = can(auth?.activeTenant?.role, "inventory:write");

  const [search, setSearch] = useState("");
  const { data: items, isLoading } = useInventoryItems(search);
  const { data: units } = useUnits();
  const saveItem = useSaveInventoryItem();
  const applyMove = useApplyMovement();

  // Item form state
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<InventoryItem | null>(null);
  const [name, setName] = useState("");
  const [type, setType] = useState("ingredient");
  const [unitId, setUnitId] = useState("");
  const [stock, setStock] = useState("");
  const [reorder, setReorder] = useState("");
  const [costPesos, setCostPesos] = useState("");

  // Movement dialog state
  const [movingItem, setMovingItem] = useState<InventoryItem | null>(null);
  const [moveType, setMoveType] = useState("stock_in");
  const [moveQty, setMoveQty] = useState("");
  const [moveNotes, setMoveNotes] = useState("");

  // History dialog
  const [historyItem, setHistoryItem] = useState<InventoryItem | null>(null);
  const { data: movements, isLoading: historyLoading } = useMovements(historyItem?.id ?? null);

  const openForm = (item: InventoryItem | null) => {
    setEditing(item);
    setName(item?.name ?? "");
    setType(item?.type ?? "ingredient");
    setUnitId(item?.unit_id ?? units?.[0]?.id ?? "");
    setStock(item ? String(item.current_stock) : "0");
    setReorder(item ? String(item.reorder_level) : "0");
    setCostPesos(item ? (item.cost_per_unit / 100).toString() : "");
    setFormOpen(true);
  };

  const submitItem = () => {
    if (!name.trim() || !unitId) {
      toast.error("Name and unit are required");
      return;
    }
    saveItem.mutate(
      {
        id: editing?.id,
        input: {
          name: name.trim(),
          type,
          unit_id: unitId,
          current_stock: Number(stock) || 0,
          reorder_level: Number(reorder) || 0,
          cost_per_unit: Math.round((Number(costPesos) || 0) * 100),
          is_active: editing?.is_active ?? true,
        },
      },
      { onSuccess: () => setFormOpen(false) },
    );
  };

  const submitMove = () => {
    const qty = Number(moveQty);
    if (!movingItem || Number.isNaN(qty) || qty === 0) {
      toast.error("Enter a non-zero quantity");
      return;
    }
    if (moveType === "adjustment" && !moveNotes.trim()) {
      toast.error("Adjustments require a reason");
      return;
    }
    applyMove.mutate(
      { item_id: movingItem.id, movement_type: moveType, qty, notes: moveNotes.trim() || undefined },
      {
        onSuccess: () => {
          setMovingItem(null);
          setMoveQty("");
          setMoveNotes("");
        },
      },
    );
  };

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Inventory</h1>
        <p className="text-muted-foreground">Ingredients, stock levels, and movement history</p>
      </header>

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-48 flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input placeholder="Search items…" className="pl-9" value={search} onChange={(e) => setSearch(e.target.value)} />
        </div>
        {canWrite && (
          <Button onClick={() => openForm(null)}>
            <Plus className="size-4" aria-hidden />
            New item
          </Button>
        )}
      </div>

      <Card className="py-0">
        <CardContent className="overflow-x-auto p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Item</TableHead>
                <TableHead className="hidden sm:table-cell">Type</TableHead>
                <TableHead className="text-right">Stock</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-32 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading &&
                Array.from({ length: 5 }, (_, i) => (
                  <TableRow key={i}>
                    <TableCell colSpan={5}><Skeleton className="h-9 w-full" /></TableCell>
                  </TableRow>
                ))}

              {(items ?? []).map((item) => (
                <TableRow key={item.id} className={cn(!item.is_active && "opacity-50")}>
                  <TableCell className="font-medium">{item.name}</TableCell>
                  <TableCell className="hidden capitalize text-muted-foreground sm:table-cell">
                    {item.type.replace("_", " ")}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {item.current_stock} {item.unit_abbr}
                  </TableCell>
                  <TableCell><StockBadge item={item} /></TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button variant="ghost" size="icon" aria-label={`History for ${item.name}`} onClick={() => setHistoryItem(item)}>
                        <History className="size-4" aria-hidden />
                      </Button>
                      {canWrite && (
                        <>
                          <Button variant="ghost" size="icon" aria-label={`Move stock for ${item.name}`} onClick={() => setMovingItem(item)}>
                            <ArrowDownUp className="size-4" aria-hidden />
                          </Button>
                          <Button variant="ghost" size="icon" aria-label={`Edit ${item.name}`} onClick={() => openForm(item)}>
                            <Pencil className="size-4" aria-hidden />
                          </Button>
                        </>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}

              {items && items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="py-10 text-center text-muted-foreground">
                    No inventory items.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Item form */}
      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit item" : "New inventory item"}</DialogTitle>
            {editing && (
              <DialogDescription>Stock levels change via movements, not here.</DialogDescription>
            )}
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="i-name">Name</Label>
              <Input id="i-name" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Type</Label>
                <Select value={type} onValueChange={setType}>
                  <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="ingredient">Ingredient</SelectItem>
                    <SelectItem value="finished_good">Finished good</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Unit</Label>
                <Select value={unitId} onValueChange={setUnitId}>
                  <SelectTrigger className="w-full"><SelectValue placeholder="Unit" /></SelectTrigger>
                  <SelectContent>
                    {(units ?? []).map((u) => (
                      <SelectItem key={u.id} value={u.id}>{u.name} ({u.abbreviation})</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="grid grid-cols-3 gap-4">
              {!editing && (
                <div className="space-y-2">
                  <Label htmlFor="i-stock">Opening stock</Label>
                  <Input id="i-stock" type="number" min="0" step="0.001" value={stock} onChange={(e) => setStock(e.target.value)} />
                </div>
              )}
              <div className="space-y-2">
                <Label htmlFor="i-reorder">Reorder level</Label>
                <Input id="i-reorder" type="number" min="0" step="0.001" value={reorder} onChange={(e) => setReorder(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="i-cost">Cost/unit (PHP)</Label>
                <Input id="i-cost" type="number" min="0" step="0.01" value={costPesos} onChange={(e) => setCostPesos(e.target.value)} />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setFormOpen(false)}>Cancel</Button>
            <Button onClick={submitItem} disabled={saveItem.isPending}>
              {saveItem.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              {editing ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Movement dialog */}
      <Dialog open={movingItem !== null} onOpenChange={(open) => !open && setMovingItem(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Move stock — {movingItem?.name}</DialogTitle>
            <DialogDescription>
              Current: {movingItem?.current_stock} {movingItem?.unit_abbr}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-2">
              {MOVE_TYPES.map((t) => (
                <button
                  key={t.value}
                  type="button"
                  onClick={() => setMoveType(t.value)}
                  className={cn(
                    "min-h-10 cursor-pointer rounded-lg border text-sm font-medium transition-colors",
                    moveType === t.value ? "border-primary bg-primary text-primary-foreground" : "hover:bg-accent/10",
                  )}
                >
                  {t.label}
                </button>
              ))}
            </div>
            <div className="space-y-2">
              <Label htmlFor="m-qty">
                Quantity ({movingItem?.unit_abbr})
                {moveType === "adjustment" && " — use negative to subtract"}
              </Label>
              <Input id="m-qty" type="number" step="0.001" value={moveQty} onChange={(e) => setMoveQty(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="m-notes">Notes{moveType === "adjustment" && " (required)"}</Label>
              <Input id="m-notes" value={moveNotes} onChange={(e) => setMoveNotes(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setMovingItem(null)}>Cancel</Button>
            <Button onClick={submitMove} disabled={applyMove.isPending}>
              {applyMove.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              Record movement
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* History dialog */}
      <Dialog open={historyItem !== null} onOpenChange={(open) => !open && setHistoryItem(null)}>
        <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-md">
          <DialogHeader>
            <DialogTitle>History — {historyItem?.name}</DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            {historyLoading && <Skeleton className="h-32 w-full" />}
            {(movements ?? []).map((m) => (
              <div key={m.id} className="rounded-lg border p-2.5 text-sm">
                <div className="flex items-center justify-between">
                  <span className="font-medium capitalize">{m.movement_type.replace("_", " ")}</span>
                  <span className={cn("font-semibold tabular-nums", m.qty_delta < 0 ? "text-rose-600" : "text-emerald-600")}>
                    {m.qty_delta > 0 ? "+" : ""}{m.qty_delta}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground">
                  {m.qty_before} → {m.qty_after} · {new Date(m.created_at).toLocaleString("en-PH")}
                </p>
                {m.notes && <p className="text-xs text-muted-foreground">{m.notes}</p>}
              </div>
            ))}
            {movements && movements.length === 0 && (
              <p className="py-6 text-center text-sm text-muted-foreground">No movements yet.</p>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
