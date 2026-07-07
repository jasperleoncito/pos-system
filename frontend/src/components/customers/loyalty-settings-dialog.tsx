"use client";

import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";

import {
  useLoyaltySettings,
  useSaveLoyaltySettings,
} from "@/hooks/use-customers";
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
import { Switch } from "@/components/ui/switch";

interface LoyaltySettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Loyalty program configuration: earn rate, redemption value, tiers. */
export function LoyaltySettingsDialog({ open, onOpenChange }: LoyaltySettingsDialogProps) {
  const { data: settings } = useLoyaltySettings();
  const save = useSaveLoyaltySettings();

  const [enabled, setEnabled] = useState(true);
  const [earnPesos, setEarnPesos] = useState("50");
  const [redeemPesos, setRedeemPesos] = useState("1");
  const [silverAt, setSilverAt] = useState("500");
  const [goldAt, setGoldAt] = useState("1500");
  const [vipAt, setVipAt] = useState("4000");
  const [silverX, setSilverX] = useState("1.25");
  const [goldX, setGoldX] = useState("1.5");
  const [vipX, setVipX] = useState("2");

  useEffect(() => {
    if (!open || !settings) return;
    setEnabled(settings.is_enabled);
    setEarnPesos((settings.earn_rate / 100).toString());
    setRedeemPesos((settings.redeem_value / 100).toString());
    setSilverAt(String(settings.silver_threshold));
    setGoldAt(String(settings.gold_threshold));
    setVipAt(String(settings.vip_threshold));
    setSilverX(String(settings.silver_multiplier));
    setGoldX(String(settings.gold_multiplier));
    setVipX(String(settings.vip_multiplier));
  }, [open, settings]);

  const submit = () => {
    save.mutate(
      {
        is_enabled: enabled,
        earn_rate: Math.round((Number(earnPesos) || 0) * 100),
        redeem_value: Math.round((Number(redeemPesos) || 0) * 100),
        silver_threshold: Number(silverAt) || 0,
        gold_threshold: Number(goldAt) || 0,
        vip_threshold: Number(vipAt) || 0,
        silver_multiplier: Number(silverX) || 1,
        gold_multiplier: Number(goldX) || 1,
        vip_multiplier: Number(vipX) || 1,
      },
      { onSuccess: () => onOpenChange(false) },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Loyalty program</DialogTitle>
          <DialogDescription>
            Customers earn 1 point per ₱{earnPesos || "—"} spent; each point is worth ₱{redeemPesos || "—"} at checkout.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex items-center justify-between rounded-lg border p-3">
            <div>
              <p className="text-sm font-medium">Program enabled</p>
              <p className="text-xs text-muted-foreground">Earning and redemption at the POS.</p>
            </div>
            <Switch checked={enabled} onCheckedChange={setEnabled} aria-label="Program enabled" />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="l-earn">Spend per point (₱)</Label>
              <Input id="l-earn" type="number" min="1" step="1" value={earnPesos} onChange={(e) => setEarnPesos(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="l-redeem">Point value (₱)</Label>
              <Input id="l-redeem" type="number" min="0.01" step="0.01" value={redeemPesos} onChange={(e) => setRedeemPesos(e.target.value)} />
            </div>
          </div>

          <div className="space-y-2">
            <Label>Tiers — lifetime points threshold × earn multiplier</Label>
            {[
              { name: "Silver", at: silverAt, setAt: setSilverAt, x: silverX, setX: setSilverX },
              { name: "Gold", at: goldAt, setAt: setGoldAt, x: goldX, setX: setGoldX },
              { name: "VIP", at: vipAt, setAt: setVipAt, x: vipX, setX: setVipX },
            ].map((tier) => (
              <div key={tier.name} className="grid grid-cols-[4rem_1fr_1fr] items-center gap-2">
                <span className="text-sm font-medium">{tier.name}</span>
                <Input
                  type="number"
                  min="0"
                  aria-label={`${tier.name} threshold`}
                  value={tier.at}
                  onChange={(e) => tier.setAt(e.target.value)}
                />
                <Input
                  type="number"
                  min="1"
                  step="0.05"
                  aria-label={`${tier.name} multiplier`}
                  value={tier.x}
                  onChange={(e) => tier.setX(e.target.value)}
                />
              </div>
            ))}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={submit} disabled={save.isPending}>
            {save.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Save settings
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
