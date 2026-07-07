"use client";

import Link from "next/link";
import { Bell, CalendarClock, ChartLine, Package } from "lucide-react";

import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
  useNotificationFeed,
  type AppNotification,
} from "@/hooks/use-notifications";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const TYPE_ICONS: Record<AppNotification["type"], typeof Bell> = {
  low_stock: Package,
  attendance: CalendarClock,
  daily_summary: ChartLine,
  system: Bell,
};

function timeAgo(iso: string): string {
  const minutes = Math.floor((Date.now() - new Date(iso).getTime()) / 60_000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

/** Topbar bell: unread badge + recent-notification dropdown. */
export function NotificationsBell({ tenantSlug }: { tenantSlug: string }) {
  const { data: feed } = useNotificationFeed(8);
  const markRead = useMarkNotificationRead();
  const markAll = useMarkAllNotificationsRead();

  const unread = feed?.unread ?? 0;
  const items = feed?.items ?? [];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="relative" aria-label={`Notifications — ${unread} unread`}>
          <Bell className="size-5" aria-hidden />
          {unread > 0 && (
            <span className="absolute right-1 top-1 flex size-4 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-white">
              {unread > 9 ? "9+" : unread}
            </span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80 p-0">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <p className="text-sm font-semibold">Notifications</p>
          {unread > 0 && (
            <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={() => markAll.mutate()}>
              Mark all read
            </Button>
          )}
        </div>
        <div className="max-h-80 overflow-y-auto">
          {items.length === 0 && (
            <p className="py-8 text-center text-sm text-muted-foreground">You’re all caught up.</p>
          )}
          {items.map((n) => {
            const Icon = TYPE_ICONS[n.type] ?? Bell;
            return (
              <Link
                key={n.id}
                href={`/${tenantSlug}${n.link || ""}`}
                onClick={() => !n.read_at && markRead.mutate(n.id)}
                className={cn(
                  "flex gap-3 border-b px-3 py-2.5 text-sm transition-colors last:border-0 hover:bg-accent/10",
                  !n.read_at && "bg-primary/5",
                )}
              >
                <Icon className={cn("mt-0.5 size-4 shrink-0", !n.read_at ? "text-primary" : "text-muted-foreground")} aria-hidden />
                <div className="min-w-0">
                  <p className={cn("leading-snug", !n.read_at && "font-medium")}>{n.title}</p>
                  {n.body && <p className="truncate text-xs text-muted-foreground">{n.body}</p>}
                  <p className="text-xs text-muted-foreground">{timeAgo(n.created_at)}</p>
                </div>
              </Link>
            );
          })}
        </div>
        <div className="border-t p-1.5">
          <Button asChild variant="ghost" size="sm" className="w-full">
            <Link href={`/${tenantSlug}/notifications`}>View all</Link>
          </Button>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
