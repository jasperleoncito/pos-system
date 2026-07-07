"use client";

import { useState } from "react";
import { Loader2, PackageCheck, Plus, Send, Trash2, XCircle } from "lucide-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import { formatCentavos, pesosToCentavos } from "@/lib/currency";
import { useInventoryItems } from "@/hooks/use-inventory";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface Supplier {
  id: string; name: string; contact_person: string; phone: string; email: string; is_active: boolean;
}
interface POItem {
  id: string; item_id: string; item_name: string; unit_abbr: string;
  qty_ordered: number; qty_received: number; unit_cost: number;
}
interface PO {
  id: string; po_number: number; supplier_name: string; status: string;
  total: number; created_at: string; items?: POItem[];
}

const PO_STATUS_COLORS: Record<string, string> = {
  draft: "bg-neutral-500", ordered: "bg-sky-600",
  partially_received: "bg-amber-500", received: "bg-emerald-600", cancelled: "bg-rose-600",
};

function SuppliersTab() {
  const queryClient = useQueryClient();
  const { data: suppliers, isLoading } = useQuery({
    queryKey: ["procure", "suppliers"],
    queryFn: async () => (await api.get<ApiEnvelope<Supplier[] | null>>("/suppliers")).data.data ?? [],
  });
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Supplier | null>(null);
  const [name, setName] = useState("");
  const [contact, setContact] = useState("");
  const [phone, setPhone] = useState("");

  const save = useMutation({
    mutationFn: async () => {
      const input = { name: name.trim(), contact_person: contact.trim(), phone: phone.trim() };
      if (editing) return (await api.put(`/suppliers/${editing.id}`, input)).data;
      return (await api.post("/suppliers", input)).data;
    },
    onSuccess: () => {
      toast.success("Supplier saved");
      queryClient.invalidateQueries({ queryKey: ["procure"] });
      setFormOpen(false);
    },
    onError: (e) => toast.error(getApiErrorMessage(e)),
  });

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <Button onClick={() => { setEditing(null); setName(""); setContact(""); setPhone(""); setFormOpen(true); }}>
          <Plus className="size-4" aria-hidden /> New supplier
        </Button>
      </div>
      {isLoading && <Skeleton className="h-16 w-full" />}
      {(suppliers ?? []).map((s) => (
        <Card key={s.id} className="py-0">
          <CardContent className="flex items-center gap-3 p-4">
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">{s.name}</p>
              <p className="text-xs text-muted-foreground">{[s.contact_person, s.phone].filter(Boolean).join(" · ")}</p>
            </div>
            <Button variant="outline" size="sm" onClick={() => {
              setEditing(s); setName(s.name); setContact(s.contact_person); setPhone(s.phone); setFormOpen(true);
            }}>Edit</Button>
          </CardContent>
        </Card>
      ))}
      {suppliers?.length === 0 && <p className="py-8 text-center text-sm text-muted-foreground">No suppliers yet.</p>}

      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader><DialogTitle>{editing ? "Edit supplier" : "New supplier"}</DialogTitle></DialogHeader>
          <div className="space-y-3">
            <div className="space-y-2"><Label htmlFor="s-name">Name</Label>
              <Input id="s-name" value={name} onChange={(e) => setName(e.target.value)} /></div>
            <div className="space-y-2"><Label htmlFor="s-contact">Contact person</Label>
              <Input id="s-contact" value={contact} onChange={(e) => setContact(e.target.value)} /></div>
            <div className="space-y-2"><Label htmlFor="s-phone">Phone</Label>
              <Input id="s-phone" value={phone} onChange={(e) => setPhone(e.target.value)} /></div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setFormOpen(false)}>Cancel</Button>
            <Button onClick={() => name.trim() ? save.mutate() : toast.error("Name is required")} disabled={save.isPending}>
              {save.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />} Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

interface PORow { item_id: string; qty: string; costPesos: string }

function PurchaseOrdersTab() {
  const queryClient = useQueryClient();
  const { data: pos, isLoading } = useQuery({
    queryKey: ["procure", "pos"],
    queryFn: async () => (await api.get<ApiEnvelope<PO[] | null>>("/purchase-orders")).data.data ?? [],
  });
  const { data: suppliers } = useQuery({
    queryKey: ["procure", "suppliers"],
    queryFn: async () => (await api.get<ApiEnvelope<Supplier[] | null>>("/suppliers")).data.data ?? [],
  });
  const { data: items } = useInventoryItems();

  const [formOpen, setFormOpen] = useState(false);
  const [supplierId, setSupplierId] = useState("");
  const [rows, setRows] = useState<PORow[]>([{ item_id: "", qty: "", costPesos: "" }]);
  const [receiving, setReceiving] = useState<PO | null>(null);
  const [receiveQty, setReceiveQty] = useState<Record<string, string>>({});

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["procure"] });

  const createPO = useMutation({
    mutationFn: async () => {
      const lines = rows.filter((r) => r.item_id && Number(r.qty) > 0);
      if (!supplierId || lines.length === 0) throw new Error("Supplier and at least one line are required");
      return (await api.post("/purchase-orders", {
        supplier_id: supplierId,
        items: lines.map((r) => ({ item_id: r.item_id, qty: Number(r.qty), unit_cost: pesosToCentavos(Number(r.costPesos) || 0) })),
      })).data;
    },
    onSuccess: () => { toast.success("PO created"); invalidate(); setFormOpen(false); },
    onError: (e) => toast.error(getApiErrorMessage(e)),
  });

  const action = useMutation({
    mutationFn: async ({ id, verb, body }: { id: string; verb: string; body?: unknown }) =>
      (await api.post(`/purchase-orders/${id}/${verb}`, body ?? {})).data.data as PO,
    onSuccess: (po, { verb }) => {
      toast.success(`PO ${verb === "receive" ? "received" : verb + "ed"}`);
      invalidate();
      queryClient.invalidateQueries({ queryKey: ["inventory"] });
      if (verb === "receive") setReceiving(po.status === "received" ? null : po);
    },
    onError: (e) => toast.error(getApiErrorMessage(e)),
  });

  const openReceive = async (po: PO) => {
    const full = (await api.get<ApiEnvelope<PO>>(`/purchase-orders/${po.id}`)).data.data;
    setReceiving(full);
    const defaults: Record<string, string> = {};
    for (const it of full.items ?? []) defaults[it.id] = String(it.qty_ordered - it.qty_received);
    setReceiveQty(defaults);
  };

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <Button onClick={() => { setSupplierId(""); setRows([{ item_id: "", qty: "", costPesos: "" }]); setFormOpen(true); }}>
          <Plus className="size-4" aria-hidden /> New purchase order
        </Button>
      </div>
      {isLoading && <Skeleton className="h-16 w-full" />}
      {(pos ?? []).map((po) => (
        <Card key={po.id} className="py-0">
          <CardContent className="flex flex-wrap items-center gap-3 p-4">
            <div className="min-w-0 flex-1">
              <p className="flex items-center gap-2 text-sm font-medium">
                PO #{po.po_number}
                <Badge className={PO_STATUS_COLORS[po.status]}>{po.status.replace("_", " ")}</Badge>
              </p>
              <p className="text-xs text-muted-foreground">
                {po.supplier_name} · {formatCentavos(po.total)} · {new Date(po.created_at).toLocaleDateString()}
              </p>
            </div>
            {po.status === "draft" && (
              <>
                <Button variant="outline" size="sm" onClick={() => action.mutate({ id: po.id, verb: "order" })}>
                  <Send className="size-4" aria-hidden /> Mark ordered
                </Button>
                <Button variant="ghost" size="sm" onClick={() => action.mutate({ id: po.id, verb: "cancel" })}>
                  <XCircle className="size-4 text-destructive" aria-hidden />
                </Button>
              </>
            )}
            {(po.status === "ordered" || po.status === "partially_received") && (
              <Button size="sm" onClick={() => openReceive(po)}>
                <PackageCheck className="size-4" aria-hidden /> Receive
              </Button>
            )}
          </CardContent>
        </Card>
      ))}
      {pos?.length === 0 && <p className="py-8 text-center text-sm text-muted-foreground">No purchase orders yet.</p>}

      {/* create PO */}
      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-md">
          <DialogHeader><DialogTitle>New purchase order</DialogTitle></DialogHeader>
          <div className="space-y-3">
            <Select value={supplierId} onValueChange={setSupplierId}>
              <SelectTrigger className="w-full"><SelectValue placeholder="Supplier" /></SelectTrigger>
              <SelectContent>
                {(suppliers ?? []).map((s) => <SelectItem key={s.id} value={s.id}>{s.name}</SelectItem>)}
              </SelectContent>
            </Select>
            {rows.map((row, i) => (
              <div key={i} className="flex items-center gap-2">
                <Select value={row.item_id} onValueChange={(v) => setRows((p) => p.map((r, j) => j === i ? { ...r, item_id: v } : r))}>
                  <SelectTrigger className="flex-1"><SelectValue placeholder="Item" /></SelectTrigger>
                  <SelectContent>
                    {(items ?? []).map((it) => <SelectItem key={it.id} value={it.id}>{it.name} ({it.unit_abbr})</SelectItem>)}
                  </SelectContent>
                </Select>
                <Input type="number" placeholder="Qty" className="w-20" aria-label="Quantity" value={row.qty}
                  onChange={(e) => setRows((p) => p.map((r, j) => j === i ? { ...r, qty: e.target.value } : r))} />
                <Input type="number" placeholder="₱/unit" className="w-24" aria-label="Unit cost" value={row.costPesos}
                  onChange={(e) => setRows((p) => p.map((r, j) => j === i ? { ...r, costPesos: e.target.value } : r))} />
                <Button variant="ghost" size="icon" aria-label="Remove line"
                  onClick={() => setRows((p) => p.filter((_, j) => j !== i))}>
                  <Trash2 className="size-4" aria-hidden />
                </Button>
              </div>
            ))}
            <Button variant="outline" size="sm" onClick={() => setRows((p) => [...p, { item_id: "", qty: "", costPesos: "" }])}>
              <Plus className="size-4" aria-hidden /> Add line
            </Button>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setFormOpen(false)}>Cancel</Button>
            <Button onClick={() => createPO.mutate()} disabled={createPO.isPending}>
              {createPO.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />} Create draft
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* receive PO */}
      <Dialog open={receiving !== null} onOpenChange={(o) => !o && setReceiving(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader><DialogTitle>Receive PO #{receiving?.po_number}</DialogTitle></DialogHeader>
          <div className="space-y-2">
            {(receiving?.items ?? []).map((it) => {
              const remaining = it.qty_ordered - it.qty_received;
              return (
                <div key={it.id} className="flex items-center gap-2">
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium">{it.item_name}</p>
                    <p className="text-xs text-muted-foreground">
                      {it.qty_received}/{it.qty_ordered} {it.unit_abbr} received · {formatCentavos(it.unit_cost)}/{it.unit_abbr}
                    </p>
                  </div>
                  <Input type="number" min="0" max={remaining} step="0.001" className="w-24"
                    aria-label={`Receive ${it.item_name}`}
                    value={receiveQty[it.id] ?? ""}
                    onChange={(e) => setReceiveQty((p) => ({ ...p, [it.id]: e.target.value }))} />
                </div>
              );
            })}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setReceiving(null)}>Cancel</Button>
            <Button
              disabled={action.isPending}
              onClick={() => receiving && action.mutate({
                id: receiving.id, verb: "receive",
                body: { lines: Object.entries(receiveQty).map(([po_item_id, qty]) => ({ po_item_id, qty: Number(qty) || 0 })) },
              })}
            >
              {action.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />} Receive stock
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default function ProcurementPage() {
  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Procurement</h1>
        <p className="text-muted-foreground">Suppliers and purchase orders</p>
      </header>
      <Tabs defaultValue="pos" className="space-y-4">
        <TabsList className="h-11">
          <TabsTrigger value="pos" className="min-h-9 px-4">Purchase orders</TabsTrigger>
          <TabsTrigger value="suppliers" className="min-h-9 px-4">Suppliers</TabsTrigger>
        </TabsList>
        <TabsContent value="pos"><PurchaseOrdersTab /></TabsContent>
        <TabsContent value="suppliers"><SuppliersTab /></TabsContent>
      </Tabs>
    </div>
  );
}
