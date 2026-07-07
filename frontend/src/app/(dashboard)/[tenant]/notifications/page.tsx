"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Bell, CalendarClock, ChartLine, CheckCheck, Loader2, Package } from "lucide-react";

import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
  useNotificationFeed,
  useNotificationPrefs,
  useSaveNotificationPrefs,
  type AppNotification,
} from "@/hooks/use-notifications";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";

const TYPE_ICONS: Record<AppNotification["type"], typeof Bell> = {
  low_stock: Package,
  attendance: CalendarClock,
  daily_summary: ChartLine,
  system: Bell,
};

const PREF_FIELDS = [
  { key: "email_low_stock" as const, label: "Low stock alerts", hint: "When an item drops below its reorder level" },
  { key: "email_attendance" as const, label: "Attendance alerts", hint: "When staff clock in late" },
  { key: "email_daily_summary" as const, label: "Daily summary", hint: "End-of-day sales recap" },
];

export default function NotificationsPage() {
  const params = useParams<{ tenant: string }>();
  const { data: feed, isLoading } = useNotificationFeed(100);
  const markRead = useMarkNotificationRead();
  const markAll = useMarkAllNotificationsRead();

  const { data: prefs } = useNotificationPrefs();
  const savePrefs = useSaveNotificationPrefs();
  const [draft, setDraft] = useState({
    email_low_stock: true,
    email_attendance: true,
    email_daily_summary: true,
  });

  useEffect(() => {
    if (prefs) setDraft(prefs);
  }, [prefs]);

  const items = feed?.items ?? [];

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">Notifications</h1>
          <p className="text-muted-foreground">
            {feed?.unread ? `${feed.unread} unread` : "You’re all caught up"}
          </p>
        </div>
        {Boolean(feed?.unread) && (
          <Button variant="outline" disabled={markAll.isPending} onClick={() => markAll.mutate()}>
            {markAll.isPending ? <Loader2 className="size-4 animate-spin" aria-hidden /> : <CheckCheck className="size-4" aria-hidden />}
            Mark all read
          </Button>
        )}
      </header>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="space-y-2 lg:col-span-2">
          {isLoading && Array.from({ length: 4 }, (_, i) => <Skeleton key={i} className="h-16 w-full rounded-xl" />)}
          {items.map((n) => {
            const Icon = TYPE_ICONS[n.type] ?? Bell;
            return (
              <Link
                key={n.id}
                href={`/${params.tenant}${n.link || ""}`}
                onClick={() => !n.read_at && markRead.mutate(n.id)}
                className={cn(
                  "flex gap-3 rounded-xl border p-3.5 transition-colors hover:border-primary",
                  !n.read_at && "border-primary/40 bg-primary/5",
                )}
              >
                <Icon className={cn("mt-0.5 size-5 shrink-0", !n.read_at ? "text-primary" : "text-muted-foreground")} aria-hidden />
                <div className="min-w-0">
                  <p className={cn("text-sm leading-snug", !n.read_at && "font-semibold")}>{n.title}</p>
                  {n.body && <p className="text-sm text-muted-foreground">{n.body}</p>}
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    {new Date(n.created_at).toLocaleString("en-PH")}
                  </p>
                </div>
              </Link>
            );
          })}
          {!isLoading && items.length === 0 && (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                <Bell className="mx-auto mb-2 size-8" aria-hidden />
                Nothing here yet — alerts and summaries will show up as they happen.
              </CardContent>
            </Card>
          )}
        </div>

        <Card className="h-fit">
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Email preferences</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {PREF_FIELDS.map((field) => (
              <div key={field.key} className="flex items-center justify-between gap-3">
                <div>
                  <Label className="text-sm">{field.label}</Label>
                  <p className="text-xs text-muted-foreground">{field.hint}</p>
                </div>
                <Switch
                  checked={draft[field.key]}
                  onCheckedChange={(v) => setDraft((prev) => ({ ...prev, [field.key]: v }))}
                  aria-label={field.label}
                />
              </div>
            ))}
            <Button
              className="w-full"
              disabled={savePrefs.isPending}
              onClick={() => savePrefs.mutate(draft)}
            >
              {savePrefs.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              Save preferences
            </Button>
            <p className="text-xs text-muted-foreground">
              In-app notifications are always on; these control emails only.
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
