"use client";

import { useEffect, useState } from "react";
import { AlarmClockCheck, Coffee, Loader2, LogIn, LogOut } from "lucide-react";

import {
  useClockIn,
  useClockOut,
  useClockStatus,
  useEndBreak,
  useStartBreak,
} from "@/hooks/use-attendance";
import { getApiErrorMessage } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

const DAY_NAMES = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

function useTicker() {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);
  return now;
}

function elapsedLabel(fromIso: string, now: Date): string {
  const ms = Math.max(0, now.getTime() - new Date(fromIso).getTime());
  const totalMinutes = Math.floor(ms / 60_000);
  const h = Math.floor(totalMinutes / 60);
  const m = totalMinutes % 60;
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

/** Big-button self-service clock: in / break / out with live elapsed time. */
export function ClockPanel() {
  const now = useTicker();
  const { data: status, isLoading, error } = useClockStatus(true);
  const clockIn = useClockIn();
  const clockOut = useClockOut();
  const startBreak = useStartBreak();
  const endBreak = useEndBreak();

  if (isLoading) {
    return <Skeleton className="h-64 w-full rounded-xl" />;
  }

  if (error || !status) {
    return (
      <Card>
        <CardContent className="py-10 text-center text-muted-foreground">
          {error ? getApiErrorMessage(error) : "Clock unavailable."}
        </CardContent>
      </Card>
    );
  }

  const open = status.open;
  const onBreak = Boolean(open?.break_start);
  const schedule = status.today_schedule;
  const busy = clockIn.isPending || clockOut.isPending || startBreak.isPending || endBreak.isPending;

  return (
    <Card>
      <CardContent className="space-y-6 py-6 text-center">
        <div>
          <p className="text-4xl font-bold tabular-nums tracking-tight sm:text-5xl">
            {now.toLocaleTimeString("en-PH", { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {now.toLocaleDateString("en-PH", { weekday: "long", year: "numeric", month: "long", day: "numeric" })}
          </p>
        </div>

        <div className="flex flex-wrap items-center justify-center gap-2 text-sm">
          {schedule ? (
            <Badge variant="outline" className="px-3 py-1">
              {DAY_NAMES[schedule.day_of_week]} shift: {schedule.start_time}–{schedule.end_time}
              {schedule.grace_minutes > 0 && ` · ${schedule.grace_minutes}m grace`}
            </Badge>
          ) : (
            <Badge variant="secondary" className="px-3 py-1">No shift scheduled today</Badge>
          )}
          {open && (
            <Badge className={cn("px-3 py-1", onBreak ? "bg-amber-500" : "bg-emerald-600")}>
              {onBreak ? "On break" : "On shift"} · in at{" "}
              {new Date(open.clock_in).toLocaleTimeString("en-PH", { hour: "2-digit", minute: "2-digit" })}
              {" · "}{elapsedLabel(open.clock_in, now)}
            </Badge>
          )}
          {open && open.late_minutes > 0 && (
            <Badge variant="destructive" className="px-3 py-1">{open.late_minutes}m late</Badge>
          )}
        </div>

        {!open ? (
          <Button
            size="lg"
            className="h-20 w-full max-w-sm text-lg font-semibold"
            disabled={busy}
            onClick={() => clockIn.mutate()}
          >
            {clockIn.isPending ? <Loader2 className="size-6 animate-spin" aria-hidden /> : <LogIn className="size-6" aria-hidden />}
            Clock in
          </Button>
        ) : (
          <div className="mx-auto grid w-full max-w-md gap-3 sm:grid-cols-2">
            {onBreak ? (
              <Button
                size="lg"
                variant="outline"
                className="h-16 text-base font-semibold"
                disabled={busy}
                onClick={() => endBreak.mutate()}
              >
                {endBreak.isPending ? <Loader2 className="size-5 animate-spin" aria-hidden /> : <AlarmClockCheck className="size-5" aria-hidden />}
                End break
              </Button>
            ) : (
              <Button
                size="lg"
                variant="outline"
                className="h-16 text-base font-semibold"
                disabled={busy}
                onClick={() => startBreak.mutate()}
              >
                {startBreak.isPending ? <Loader2 className="size-5 animate-spin" aria-hidden /> : <Coffee className="size-5" aria-hidden />}
                Start break
              </Button>
            )}
            <Button
              size="lg"
              variant="destructive"
              className="h-16 text-base font-semibold"
              disabled={busy || onBreak}
              onClick={() => clockOut.mutate()}
            >
              {clockOut.isPending ? <Loader2 className="size-5 animate-spin" aria-hidden /> : <LogOut className="size-5" aria-hidden />}
              Clock out
            </Button>
          </div>
        )}

        {open && open.break_minutes > 0 && (
          <p className="text-xs text-muted-foreground">Breaks so far: {open.break_minutes} minutes</p>
        )}
        {onBreak && (
          <p className="text-xs text-muted-foreground">End your break before clocking out.</p>
        )}
      </CardContent>
    </Card>
  );
}
