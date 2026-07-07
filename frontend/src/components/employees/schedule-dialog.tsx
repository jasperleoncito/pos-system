"use client";

import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

import {
  useEmployeeSchedule,
  useSaveSchedule,
  type Employee,
  type ScheduleDay,
} from "@/hooks/use-employees";
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
import { Switch } from "@/components/ui/switch";

const DAY_NAMES = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

interface DayRow {
  enabled: boolean;
  start: string;
  end: string;
  grace: string;
}

const DEFAULT_ROW: DayRow = { enabled: false, start: "09:00", end: "17:00", grace: "10" };

interface ScheduleDialogProps {
  employee: Employee | null;
  onOpenChange: (open: boolean) => void;
}

/** Weekly schedule editor: one row per day with shift window + grace. */
export function ScheduleDialog({ employee, onOpenChange }: ScheduleDialogProps) {
  const { data: schedule, isLoading } = useEmployeeSchedule(employee?.id ?? null);
  const saveSchedule = useSaveSchedule();
  const [rows, setRows] = useState<DayRow[]>(Array.from({ length: 7 }, () => ({ ...DEFAULT_ROW })));

  useEffect(() => {
    if (!schedule) return;
    setRows(
      Array.from({ length: 7 }, (_, dow) => {
        const day = schedule.find((d) => d.day_of_week === dow);
        return day
          ? { enabled: true, start: day.start_time, end: day.end_time, grace: String(day.grace_minutes) }
          : { ...DEFAULT_ROW };
      }),
    );
  }, [schedule]);

  const setRow = (dow: number, patch: Partial<DayRow>) =>
    setRows((prev) => prev.map((row, i) => (i === dow ? { ...row, ...patch } : row)));

  const submit = () => {
    if (!employee) return;
    const days: ScheduleDay[] = [];
    for (let dow = 0; dow < 7; dow++) {
      const row = rows[dow];
      if (!row.enabled) continue;
      if (row.start >= row.end) {
        toast.error(`${DAY_NAMES[dow]}: start must be before end`);
        return;
      }
      days.push({
        day_of_week: dow,
        start_time: row.start,
        end_time: row.end,
        grace_minutes: Number(row.grace) || 0,
      });
    }
    saveSchedule.mutate(
      { employeeId: employee.id, days },
      { onSuccess: () => onOpenChange(false) },
    );
  };

  return (
    <Dialog open={employee !== null} onOpenChange={(open) => !open && onOpenChange(false)}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Schedule — {employee?.full_name}</DialogTitle>
          <DialogDescription>
            Clock-ins after the shift start plus grace minutes count as late.
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <Skeleton className="h-64 w-full" />
        ) : (
          <div className="space-y-2">
            <div className="hidden grid-cols-[7.5rem_1fr_1fr_5rem] gap-2 px-1 text-xs font-medium text-muted-foreground sm:grid">
              <span>Day</span>
              <span>Start</span>
              <span>End</span>
              <span>Grace (min)</span>
            </div>
            {rows.map((row, dow) => (
              <div
                key={DAY_NAMES[dow]}
                className="grid grid-cols-2 items-center gap-2 rounded-lg border p-2 sm:grid-cols-[7.5rem_1fr_1fr_5rem] sm:border-0 sm:p-1"
              >
                <div className="flex items-center gap-2">
                  <Switch
                    checked={row.enabled}
                    onCheckedChange={(v) => setRow(dow, { enabled: v })}
                    aria-label={`Works on ${DAY_NAMES[dow]}`}
                  />
                  <Label className="text-sm">{DAY_NAMES[dow].slice(0, 3)}</Label>
                </div>
                <Input
                  type="time"
                  value={row.start}
                  disabled={!row.enabled}
                  onChange={(e) => setRow(dow, { start: e.target.value })}
                  aria-label={`${DAY_NAMES[dow]} start`}
                />
                <Input
                  type="time"
                  value={row.end}
                  disabled={!row.enabled}
                  onChange={(e) => setRow(dow, { end: e.target.value })}
                  aria-label={`${DAY_NAMES[dow]} end`}
                />
                <Input
                  type="number"
                  min="0"
                  max="240"
                  value={row.grace}
                  disabled={!row.enabled}
                  onChange={(e) => setRow(dow, { grace: e.target.value })}
                  aria-label={`${DAY_NAMES[dow]} grace minutes`}
                />
              </div>
            ))}
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={submit} disabled={saveSchedule.isPending}>
            {saveSchedule.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Save schedule
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
