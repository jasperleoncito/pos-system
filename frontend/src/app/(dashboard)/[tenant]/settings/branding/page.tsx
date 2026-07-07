"use client";

import { useEffect, useRef } from "react";
import { useForm } from "react-hook-form";
import { ImagePlus, Loader2 } from "lucide-react";
import { toast } from "sonner";

import { useTenantSettings, useUpdateTenantSettings, useUploadLogo, type UpdateSettingsInput } from "@/hooks/use-tenant";
import { useAuth } from "@/hooks/use-auth";
import { can } from "@/lib/rbac";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

const COLOR_FIELDS = [
  { name: "primary_color", label: "Primary" },
  { name: "secondary_color", label: "Secondary" },
  { name: "accent_color", label: "Accent" },
] as const;

export default function BrandingPage() {
  const { auth } = useAuth();
  const { data: settings, isLoading } = useTenantSettings();
  const updateSettings = useUpdateTenantSettings();
  const uploadLogo = useUploadLogo();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const canWrite = can(auth?.activeTenant?.role, "tenant_settings:write");

  const { register, handleSubmit, reset, watch } = useForm<UpdateSettingsInput>();

  useEffect(() => {
    if (settings) {
      reset({
        primary_color: settings.primary_color,
        secondary_color: settings.secondary_color,
        accent_color: settings.accent_color,
        receipt_header: settings.receipt_header,
        receipt_footer: settings.receipt_footer,
        contact_number: settings.contact_number,
        facebook: settings.facebook,
        website: settings.website,
        address: settings.address,
        tax_label: settings.tax_label,
        tax_id: settings.tax_id,
      });
    }
  }, [settings, reset]);

  const onLogoPicked = (file: File | undefined) => {
    if (!file) return;
    if (file.size > MAX_UPLOAD_BYTES) {
      toast.error("Logo must be 10MB or smaller");
      return;
    }
    uploadLogo.mutate(file);
  };

  if (isLoading || !settings) {
    return (
      <div className="mx-auto max-w-2xl space-y-4">
        <Skeleton className="h-10 w-56" />
        <Skeleton className="h-64 w-full" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }

  const colors = watch();

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Branding</h1>
        <p className="text-muted-foreground">
          Logo, colors, and receipt details for {auth?.activeTenant?.tenant_name}
        </p>
      </header>

      {/* ---- Logo ---- */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Logo</CardTitle>
          <CardDescription>
            PNG, JPG, or WEBP up to 10MB — automatically optimized to WebP with
            favicons generated
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap items-center gap-6">
          <div className="flex size-24 items-center justify-center overflow-hidden rounded-xl border bg-muted">
            {settings.logo_thumb_url ? (
              // eslint-disable-next-line @next/next/no-img-element -- MinIO-served, unoptimizable by Next
              <img
                src={settings.logo_thumb_url}
                alt="Business logo"
                width={96}
                height={96}
                className="size-full object-contain"
              />
            ) : (
              <ImagePlus className="size-8 text-muted-foreground" aria-hidden />
            )}
          </div>
          {canWrite && (
            <div className="space-y-2">
              <input
                ref={fileInputRef}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                className="hidden"
                onChange={(e) => onLogoPicked(e.target.files?.[0])}
              />
              <Button
                variant="outline"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploadLogo.isPending}
              >
                {uploadLogo.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
                {uploadLogo.isPending ? "Optimizing…" : "Upload logo"}
              </Button>
              <p className="text-xs text-muted-foreground">
                Large uploads are resized, compressed, and stripped of metadata.
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* ---- Colors + business info ---- */}
      <form
        onSubmit={handleSubmit((input) => updateSettings.mutate(input))}
        className="space-y-6"
      >
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Brand colors</CardTitle>
            <CardDescription>The dashboard and receipts recolor instantly on save</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-3">
            {COLOR_FIELDS.map(({ name, label }) => (
              <div key={name} className="space-y-2">
                <Label htmlFor={name}>{label}</Label>
                <div className="flex items-center gap-2">
                  <input
                    type="color"
                    aria-label={`${label} color picker`}
                    className="size-10 shrink-0 cursor-pointer rounded-md border bg-transparent p-1"
                    value={colors[name] || "#000000"}
                    onChange={(e) =>
                      reset({ ...colors, [name]: e.target.value }, { keepDirty: true })
                    }
                    disabled={!canWrite}
                  />
                  <Input id={name} {...register(name)} disabled={!canWrite} />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Receipt & business details</CardTitle>
            <CardDescription>Shown on printed receipts and customer-facing pages</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="receipt_header">Receipt header</Label>
                <Input id="receipt_header" {...register("receipt_header")} disabled={!canWrite} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="receipt_footer">Receipt footer</Label>
                <Input id="receipt_footer" {...register("receipt_footer")} disabled={!canWrite} />
              </div>
            </div>

            <Separator />

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="contact_number">Contact number</Label>
                <Input id="contact_number" {...register("contact_number")} disabled={!canWrite} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="facebook">Facebook</Label>
                <Input id="facebook" {...register("facebook")} disabled={!canWrite} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="website">Website</Label>
                <Input id="website" {...register("website")} disabled={!canWrite} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="address">Business address</Label>
                <Input id="address" {...register("address")} disabled={!canWrite} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tax_label">Tax label</Label>
                <Input id="tax_label" placeholder="VAT Reg TIN" {...register("tax_label")} disabled={!canWrite} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tax_id">Tax ID</Label>
                <Input id="tax_id" {...register("tax_id")} disabled={!canWrite} />
              </div>
            </div>
          </CardContent>
        </Card>

        {canWrite && (
          <div className="flex justify-end">
            <Button type="submit" disabled={updateSettings.isPending}>
              {updateSettings.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              Save branding
            </Button>
          </div>
        )}
      </form>
    </div>
  );
}
