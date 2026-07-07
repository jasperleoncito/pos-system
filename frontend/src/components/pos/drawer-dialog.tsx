"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

import { formatCentavos, pesosToCentavos } from "@/lib/currency";
import { useCloseDrawer, useCurrentDrawer, useOpenDrawer } from "@/hooks/use-orders";
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
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";

interface DrawerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Open/close the cash drawer with live expected-cash and variance. */
export function DrawerDialog({ open, onOpenChange }: DrawerDialogProps) {
  const { data, isLoading } = useCurrentDrawer();
  const openDrawer = useOpenDrawer();
  const closeDrawer = useCloseDrawer();
  const [amountPesos, setAmountPesos] = useState("");

  const session = data?.session;

  const submit = () => {
    const amount = Number(amountPesos);
    if (Number.isNaN(amount) || amount < 0) {
      toast.error("Enter a valid amount");
      return;
    }
    if (session) {
      closeDrawer.mutate(pesosToCentavos(amount), {
        onSuccess: (closed) => {
          const variance = closed.variance ?? 0;
          if (variance === 0) {
            toast.success("Drawer closed — balanced");
          } else {
            toast.warning(
              `Drawer closed — ${variance > 0 ? "over" : "short"} by ${formatCentavos(Math.abs(variance))}`,
            );
          }
          setAmountPesos("");
          onOpenChange(false);
        },
      });
    } else {
      openDrawer.mutate(pesosToCentavos(amount), {
        onSuccess: () => {
          setAmountPesos("");
          onOpenChange(false);
        },
      });
    }
  };

  const isBusy = openDrawer.isPending || closeDrawer.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{session ? "Close cash drawer" : "Open cash drawer"}</DialogTitle>
          <DialogDescription>
            {session
              ? "Count the cash in the drawer to close the session"
              : "Enter the starting float to begin taking cash"}
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <Skeleton className="h-24 w-full" />
        ) : (
          <div className="space-y-4">
            {session && (
              <div className="space-y-1 rounded-lg bg-muted p-3 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Opening float</span>
                  <span className="tabular-nums">{formatCentavos(session.opening_float)}</span>
                </div>
                <div className="flex justify-between font-medium">
                  <span>Expected cash</span>
                  <span className="tabular-nums">{formatCentavos(session.expected_cash)}</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Open since {new Date(session.opened_at).toLocaleTimeString()}
                </p>
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="drawer-amount">
                {session ? "Counted cash (PHP)" : "Opening float (PHP)"}
              </Label>
              <Input
                id="drawer-amount"
                type="number"
                min="0"
                step="0.01"
                inputMode="decimal"
                value={amountPesos}
                onChange={(e) => setAmountPesos(e.target.value)}
              />
            </div>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={submit} disabled={isBusy || isLoading}>
            {isBusy && <Loader2 className="size-4 animate-spin" aria-hidden />}
            {session ? "Close drawer" : "Open drawer"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
