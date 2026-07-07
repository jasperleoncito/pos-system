"use client";

import { useEffect, useState } from "react";
import { Loader2, Pencil, Plus, Search, Settings2, Star, Trash2 } from "lucide-react";
import { toast } from "sonner";

import {
  useCustomers,
  useDeleteCustomer,
  useSaveCustomer,
  type Customer,
} from "@/hooks/use-customers";
import { useAuth } from "@/hooks/use-auth";
import { can } from "@/lib/rbac";
import { cn } from "@/lib/utils";
import { TierBadge } from "@/components/pos/customer-dialog";
import { CustomerProfileDialog } from "@/components/customers/customer-profile-dialog";
import { LoyaltySettingsDialog } from "@/components/customers/loyalty-settings-dialog";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";

export default function CustomersPage() {
  const { auth } = useAuth();
  const role = auth?.activeTenant?.role;
  const canWrite = can(role, "customers:write");
  const canConfigure = can(role, "catalog:write"); // loyalty program is manager+

  const [search, setSearch] = useState("");
  const { data: customers, isLoading } = useCustomers(search);
  const save = useSaveCustomer();
  const remove = useDeleteCustomer();

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Customer | null>(null);
  const [viewing, setViewing] = useState<Customer | null>(null);
  const [removing, setRemoving] = useState<Customer | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);

  const [fullName, setFullName] = useState("");
  const [phone, setPhone] = useState("");
  const [email, setEmail] = useState("");
  const [birthday, setBirthday] = useState("");
  const [notes, setNotes] = useState("");

  useEffect(() => {
    if (!formOpen) return;
    setFullName(editing?.full_name ?? "");
    setPhone(editing?.phone ?? "");
    setEmail(editing?.email ?? "");
    setBirthday(editing?.birthday ? editing.birthday.slice(0, 10) : "");
    setNotes(editing?.notes ?? "");
  }, [formOpen, editing]);

  const submit = () => {
    if (!fullName.trim()) {
      toast.error("Full name is required");
      return;
    }
    save.mutate(
      {
        id: editing?.id,
        input: {
          full_name: fullName.trim(),
          phone: phone.trim(),
          email: email.trim(),
          birthday: birthday || undefined,
          notes: notes.trim(),
          is_active: editing?.is_active ?? true,
        },
      },
      { onSuccess: () => setFormOpen(false) },
    );
  };

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Customers</h1>
        <p className="text-muted-foreground">Loyalty members, points, and purchase history</p>
      </header>

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-48 flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input
            placeholder="Search name, phone, or email…"
            className="pl-9"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        {canConfigure && (
          <Button variant="outline" onClick={() => setSettingsOpen(true)}>
            <Settings2 className="size-4" aria-hidden />
            Loyalty program
          </Button>
        )}
        {canWrite && (
          <Button onClick={() => { setEditing(null); setFormOpen(true); }}>
            <Plus className="size-4" aria-hidden />
            New customer
          </Button>
        )}
      </div>

      <Card className="py-0">
        <CardContent className="overflow-x-auto p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Customer</TableHead>
                <TableHead className="hidden sm:table-cell">Contact</TableHead>
                <TableHead className="text-right">Points</TableHead>
                <TableHead>Tier</TableHead>
                <TableHead className="w-28 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading &&
                Array.from({ length: 4 }, (_, i) => (
                  <TableRow key={i}>
                    <TableCell colSpan={5}><Skeleton className="h-10 w-full" /></TableCell>
                  </TableRow>
                ))}

              {(customers ?? []).map((customer) => (
                <TableRow
                  key={customer.id}
                  className={cn("cursor-pointer", !customer.is_active && "opacity-50")}
                  onClick={() => setViewing(customer)}
                >
                  <TableCell className="font-medium">{customer.full_name}</TableCell>
                  <TableCell className="hidden text-sm text-muted-foreground sm:table-cell">
                    {customer.phone || customer.email || "—"}
                  </TableCell>
                  <TableCell className="text-right">
                    <span className="inline-flex items-center gap-1 font-semibold tabular-nums">
                      {customer.points_balance}
                      <Star className="size-3.5 text-amber-500" aria-hidden />
                    </span>
                  </TableCell>
                  <TableCell><TierBadge tier={customer.tier} /></TableCell>
                  <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                    {canWrite && (
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Edit ${customer.full_name}`}
                          onClick={() => { setEditing(customer); setFormOpen(true); }}
                        >
                          <Pencil className="size-4" aria-hidden />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Remove ${customer.full_name}`}
                          onClick={() => setRemoving(customer)}
                        >
                          <Trash2 className="size-4 text-destructive" aria-hidden />
                        </Button>
                      </div>
                    )}
                  </TableCell>
                </TableRow>
              ))}

              {customers && customers.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="py-10 text-center text-muted-foreground">
                    No customers yet.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Create / edit */}
      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit customer" : "New customer"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="c-name">Full name</Label>
              <Input id="c-name" value={fullName} onChange={(e) => setFullName(e.target.value)} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="c-phone">Phone</Label>
                <Input id="c-phone" value={phone} onChange={(e) => setPhone(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="c-bday">Birthday</Label>
                <Input id="c-bday" type="date" value={birthday} onChange={(e) => setBirthday(e.target.value)} />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="c-email">Email</Label>
              <Input id="c-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="c-notes">Notes</Label>
              <Textarea id="c-notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
            </div>
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

      <CustomerProfileDialog customer={viewing} onClose={() => setViewing(null)} />
      <LoyaltySettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />

      <AlertDialog open={removing !== null} onOpenChange={(open) => !open && setRemoving(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {removing?.full_name}?</AlertDialogTitle>
            <AlertDialogDescription>
              The profile is archived; their order history is kept.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (removing) remove.mutate(removing.id);
                setRemoving(null);
              }}
            >
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
