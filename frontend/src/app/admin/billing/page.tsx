"use client";

import { useEffect, useState } from "react";
import { BadgeCheck, Loader2, PhilippinePeso, Plus, Power, ReceiptText, Ticket, Trash2 } from "lucide-react";

import {
  useAdminBillingSettings,
  useAdminBillingStats,
  useAdminCreateVoucher,
  useAdminDeleteVoucher,
  useAdminSetVoucherActive,
  useAdminUpdatePrices,
  useAdminVouchers,
  type CreateVoucherInput,
} from "@/hooks/use-billing";
import { formatCentavos } from "@/lib/currency";
import type { Voucher, VoucherDiscountType, VoucherScope } from "@/types/billing";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

function fmtDate(value?: string | null) {
  if (!value) return "—";
  return new Date(value).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

function PricesCard() {
  const { data: settings } = useAdminBillingSettings();
  const update = useAdminUpdatePrices();
  const [monthly, setMonthly] = useState("");
  const [yearly, setYearly] = useState("");

  useEffect(() => {
    if (!settings) return;
    setMonthly((settings.monthly_price / 100).toString());
    setYearly((settings.yearly_price / 100).toString());
  }, [settings]);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Plan prices</CardTitle>
        <CardDescription>Applied to every new invoice and renewal notice platform-wide. In pesos.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="price-monthly">Monthly (PHP)</Label>
            <Input id="price-monthly" type="number" min="1" step="0.01" value={monthly} onChange={(e) => setMonthly(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="price-yearly">Yearly (PHP)</Label>
            <Input id="price-yearly" type="number" min="1" step="0.01" value={yearly} onChange={(e) => setYearly(e.target.value)} />
          </div>
        </div>
        <Button
          disabled={update.isPending || !monthly || !yearly}
          onClick={() => update.mutate({ monthly_price: Math.round(Number(monthly) * 100), yearly_price: Math.round(Number(yearly) * 100) })}
        >
          {update.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
          Save prices
        </Button>
      </CardContent>
    </Card>
  );
}

function CreateVoucherDialog() {
  const create = useAdminCreateVoucher();
  const [open, setOpen] = useState(false);
  const [code, setCode] = useState("");
  const [type, setType] = useState<VoucherDiscountType>("percentage");
  const [value, setValue] = useState("");
  const [scope, setScope] = useState<VoucherScope>("all");
  const [maxUses, setMaxUses] = useState("");
  const [expires, setExpires] = useState("");

  const reset = () => {
    setCode("");
    setType("percentage");
    setValue("");
    setScope("all");
    setMaxUses("");
    setExpires("");
  };

  const submit = () => {
    const input: CreateVoucherInput = {
      code: code.trim(),
      discount_type: type,
      // fixed = pesos → centavos; percentage = whole percent
      discount_value: type === "fixed" ? Math.round(Number(value) * 100) : Math.round(Number(value)),
      applies_to: scope,
      max_uses: maxUses ? Number(maxUses) : null,
      expires_at: expires ? new Date(`${expires}T23:59:59`).toISOString() : null,
    };
    create.mutate(input, {
      onSuccess: () => {
        setOpen(false);
        reset();
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Button onClick={() => setOpen(true)}>
        <Plus className="size-4" aria-hidden />
        New voucher
      </Button>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create a voucher</DialogTitle>
          <DialogDescription>Owners enter the code when paying their subscription.</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="v-code">Code</Label>
            <Input id="v-code" placeholder="LAUNCH50" value={code} onChange={(e) => setCode(e.target.value.toUpperCase())} />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Discount type</Label>
              <Select value={type} onValueChange={(v) => setType(v as VoucherDiscountType)}>
                <SelectTrigger aria-label="Discount type"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="percentage">Percentage (%)</SelectItem>
                  <SelectItem value="fixed">Fixed (₱)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="v-value">{type === "fixed" ? "Amount (PHP)" : "Percent off"}</Label>
              <Input id="v-value" type="number" min="1" max={type === "percentage" ? "100" : undefined} step={type === "fixed" ? "0.01" : "1"} value={value} onChange={(e) => setValue(e.target.value)} />
            </div>
          </div>
          <div className="space-y-2">
            <Label>Applies to</Label>
            <Select value={scope} onValueChange={(v) => setScope(v as VoucherScope)}>
              <SelectTrigger aria-label="Applies to"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All plans</SelectItem>
                <SelectItem value="monthly">Monthly only</SelectItem>
                <SelectItem value="yearly">Yearly only</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="v-max">Max uses (optional)</Label>
              <Input id="v-max" type="number" min="1" placeholder="unlimited" value={maxUses} onChange={(e) => setMaxUses(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="v-exp">Expires (optional)</Label>
              <Input id="v-exp" type="date" value={expires} onChange={(e) => setExpires(e.target.value)} />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
          <Button onClick={submit} disabled={create.isPending || !code.trim() || !value}>
            {create.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Create voucher
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function VoucherRow({ v }: { v: Voucher }) {
  const setActive = useAdminSetVoucherActive();
  const del = useAdminDeleteVoucher();
  const discount = v.discount_type === "fixed" ? `${formatCentavos(v.discount_value)} off` : `${v.discount_value}% off`;
  const uses = v.max_uses ? `${v.used_count}/${v.max_uses} used` : `${v.used_count} used`;

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-lg border p-3">
      <div className="min-w-0 flex-1">
        <p className="flex flex-wrap items-center gap-2 text-sm font-medium">
          <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-semibold">{v.code}</code>
          <Badge variant="outline">{discount}</Badge>
          <Badge variant="outline" className="capitalize">{v.applies_to === "all" ? "all plans" : `${v.applies_to} only`}</Badge>
          {!v.active && <Badge variant="destructive">inactive</Badge>}
        </p>
        <p className="text-xs text-muted-foreground">
          {uses}
          {v.expires_at ? ` · expires ${fmtDate(v.expires_at)}` : " · no expiry"}
        </p>
      </div>
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-1.5">
          <Switch checked={v.active} disabled={setActive.isPending} onCheckedChange={(active) => setActive.mutate({ id: v.id, active })} aria-label="Active" />
          <span className="text-xs text-muted-foreground">{v.active ? "on" : "off"}</span>
        </div>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="ghost" size="icon" aria-label="Delete voucher" disabled={del.isPending}>
              <Trash2 className="size-4 text-destructive" aria-hidden />
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete voucher {v.code}?</AlertDialogTitle>
              <AlertDialogDescription>It can no longer be redeemed. Existing paid subscriptions are unaffected.</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction onClick={() => del.mutate(v.id)}>Delete</AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </div>
  );
}

function VouchersTab() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAdminVouchers(page);
  const total = data?.meta?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / (data?.meta?.limit ?? 20)));

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0">
        <div>
          <CardTitle className="text-base">Vouchers</CardTitle>
          <CardDescription>Subscription discount codes · {total} total</CardDescription>
        </div>
        <CreateVoucherDialog />
      </CardHeader>
      <CardContent className="space-y-3">
        {isLoading && Array.from({ length: 3 }, (_, i) => <Skeleton key={i} className="h-16 w-full" />)}
        {data?.vouchers.map((v) => <VoucherRow key={v.id} v={v} />)}
        {data && data.vouchers.length === 0 && (
          <p className="py-8 text-center text-sm text-muted-foreground">
            No vouchers yet. Create one to give owners a discount on monthly or yearly plans.
          </p>
        )}
        {pageCount > 1 && (
          <div className="flex items-center justify-between pt-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>Previous</Button>
            <span className="text-sm text-muted-foreground">Page {page} of {pageCount}</span>
            <Button variant="outline" size="sm" disabled={page >= pageCount} onClick={() => setPage((p) => p + 1)}>Next</Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default function AdminBillingPage() {
  const { data: stats } = useAdminBillingStats();

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Billing</h1>
        <p className="text-muted-foreground">Plan pricing and discount vouchers</p>
      </header>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {[
          { label: "Active", value: stats ? String(stats.subs_active) : "—", icon: BadgeCheck },
          { label: "Awaiting payment", value: stats ? String(stats.subs_pending) : "—", icon: ReceiptText },
          { label: "Inactive", value: stats ? String(stats.subs_inactive) : "—", icon: Power },
          { label: "Collected (30d)", value: stats ? formatCentavos(stats.collected_30d) : "—", icon: PhilippinePeso },
        ].map((s) => (
          <Card key={s.label} className="py-4">
            <CardContent className="px-4">
              <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <s.icon className="size-3.5" aria-hidden />
                {s.label}
              </p>
              <p className="text-xl font-bold tabular-nums tracking-tight">{s.value}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Tabs defaultValue="vouchers">
        <TabsList>
          <TabsTrigger value="vouchers">
            <Ticket className="mr-1.5 size-4" aria-hidden />
            Vouchers
          </TabsTrigger>
          <TabsTrigger value="prices">Prices</TabsTrigger>
        </TabsList>
        <TabsContent value="vouchers" className="mt-4">
          <VouchersTab />
        </TabsContent>
        <TabsContent value="prices" className="mt-4">
          <PricesCard />
        </TabsContent>
      </Tabs>
    </div>
  );
}
