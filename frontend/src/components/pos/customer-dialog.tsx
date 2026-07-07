"use client";

import { useState } from "react";
import { Loader2, Search, Star, UserPlus, UserX } from "lucide-react";
import { toast } from "sonner";

import { useCustomers, useSaveCustomer, type Customer } from "@/hooks/use-customers";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

export const TIER_STYLES: Record<Customer["tier"], string> = {
  regular: "bg-muted text-muted-foreground",
  silver: "bg-slate-300 text-slate-900",
  gold: "bg-amber-400 text-amber-950",
  vip: "bg-violet-600 text-white",
};

export function TierBadge({ tier }: { tier: Customer["tier"] }) {
  return <Badge className={cn("capitalize", TIER_STYLES[tier])}>{tier}</Badge>;
}

interface CustomerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selected: Customer | null;
  onSelect: (customer: Customer | null) => void;
}

/** POS customer picker: search members, quick-create, or detach. */
export function CustomerDialog({ open, onOpenChange, selected, onSelect }: CustomerDialogProps) {
  const [search, setSearch] = useState("");
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const [newPhone, setNewPhone] = useState("");

  const { data: customers, isLoading } = useCustomers(search);
  const save = useSaveCustomer();

  const pick = (customer: Customer | null) => {
    onSelect(customer);
    onOpenChange(false);
    setCreating(false);
  };

  const quickCreate = () => {
    if (!newName.trim()) {
      toast.error("Customer name is required");
      return;
    }
    save.mutate(
      {
        input: {
          full_name: newName.trim(), phone: newPhone.trim(),
          email: "", notes: "", is_active: true,
        },
      },
      {
        onSuccess: (created) => {
          if (created) pick(created);
          setNewName("");
          setNewPhone("");
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Attach customer</DialogTitle>
          <DialogDescription>Points are earned and redeemed on this sale.</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
            <Input
              placeholder="Search name or phone…"
              className="pl-9"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>

          {selected && (
            <Button variant="outline" className="w-full justify-start text-destructive" onClick={() => pick(null)}>
              <UserX className="size-4" aria-hidden />
              Detach {selected.full_name}
            </Button>
          )}

          <div className="max-h-64 space-y-1.5 overflow-y-auto">
            {isLoading && <Skeleton className="h-24 w-full" />}
            {(customers ?? []).map((customer) => (
              <button
                key={customer.id}
                type="button"
                onClick={() => pick(customer)}
                className={cn(
                  "flex w-full cursor-pointer items-center justify-between gap-2 rounded-lg border p-2.5 text-left transition-colors hover:border-primary",
                  selected?.id === customer.id && "border-primary bg-primary/5",
                )}
              >
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{customer.full_name}</p>
                  <p className="text-xs text-muted-foreground">
                    {customer.phone || "no phone"} · <Star className="inline size-3" aria-hidden />{" "}
                    {customer.points_balance} pts
                  </p>
                </div>
                <TierBadge tier={customer.tier} />
              </button>
            ))}
            {customers && customers.length === 0 && (
              <p className="py-6 text-center text-sm text-muted-foreground">No customers found.</p>
            )}
          </div>

          {creating ? (
            <div className="space-y-2 rounded-lg border p-3">
              <Input placeholder="Full name" value={newName} onChange={(e) => setNewName(e.target.value)} />
              <Input placeholder="Phone (optional)" value={newPhone} onChange={(e) => setNewPhone(e.target.value)} />
              <div className="flex gap-2">
                <Button variant="outline" size="sm" className="flex-1" onClick={() => setCreating(false)}>
                  Cancel
                </Button>
                <Button size="sm" className="flex-1" disabled={save.isPending} onClick={quickCreate}>
                  {save.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
                  Create & attach
                </Button>
              </div>
            </div>
          ) : (
            <Button variant="outline" className="w-full" onClick={() => setCreating(true)}>
              <UserPlus className="size-4" aria-hidden />
              New customer
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
