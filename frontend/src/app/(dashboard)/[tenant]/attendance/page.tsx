"use client";

import { useMemo, useState } from "react";
import { CheckCheck, Loader2 } from "lucide-react";

import {
  useApproveAttendance,
  useAttendanceRecords,
  type AttendanceRecord,
} from "@/hooks/use-attendance";
import { useEmployees } from "@/hooks/use-employees";
import { useAuth } from "@/hooks/use-auth";
import { can } from "@/lib/rbac";
import { ClockPanel } from "@/components/attendance/clock-panel";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

/** Local-timezone YYYY-MM-DD (toISOString would shift the day near midnight). */
function isoDate(d: Date): string {
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${month}-${day}`;
}

function timeLabel(iso: string | null): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleTimeString("en-PH", { hour: "2-digit", minute: "2-digit" });
}

function workedLabel(r: AttendanceRecord): string {
  if (!r.clock_out) return "…";
  const ms = new Date(r.clock_out).getTime() - new Date(r.clock_in).getTime();
  const minutes = Math.max(0, Math.floor(ms / 60_000) - r.break_minutes);
  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}

export default function AttendancePage() {
  const { auth } = useAuth();
  const role = auth?.activeTenant?.role;
  const canRead = can(role, "attendance:read");
  const canApprove = can(role, "attendance:approve");

  const [employeeId, setEmployeeId] = useState("all");
  const [from, setFrom] = useState(() => isoDate(new Date(Date.now() - 6 * 86_400_000)));
  const [to, setTo] = useState(() => isoDate(new Date()));

  const { data: employees } = useEmployees("");
  const filters = useMemo(
    () => ({ employeeId: employeeId === "all" ? "" : employeeId, from, to }),
    [employeeId, from, to],
  );
  const { data: records, isLoading } = useAttendanceRecords(filters, canRead);
  const approve = useApproveAttendance();

  const totals = useMemo(() => {
    const list = records ?? [];
    return {
      shifts: list.length,
      late: list.reduce((sum, r) => sum + r.late_minutes, 0),
      overtime: list.reduce((sum, r) => sum + r.overtime_minutes, 0),
      pending: list.filter((r) => r.status === "pending" && r.clock_out).length,
    };
  }, [records]);

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Attendance</h1>
        <p className="text-muted-foreground">Clock in and out — managers review and approve below</p>
      </header>

      <ClockPanel />

      {canRead && employees !== undefined && (
        <section className="space-y-4" aria-label="Attendance records">
          <div className="flex flex-wrap items-end gap-3">
            <div className="space-y-1.5">
              <Label>Employee</Label>
              <Select value={employeeId} onValueChange={setEmployeeId}>
                <SelectTrigger className="min-w-44"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All employees</SelectItem>
                  {(employees ?? []).map((e) => (
                    <SelectItem key={e.id} value={e.id}>{e.full_name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="a-from">From</Label>
              <Input id="a-from" type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="a-to">To</Label>
              <Input id="a-to" type="date" value={to} onChange={(e) => setTo(e.target.value)} />
            </div>
            <p className="ml-auto text-sm text-muted-foreground">
              {totals.shifts} shifts · {totals.late}m late · {totals.overtime}m OT
              {totals.pending > 0 && ` · ${totals.pending} awaiting approval`}
            </p>
          </div>

          <Card className="py-0">
            <CardContent className="overflow-x-auto p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Employee</TableHead>
                    <TableHead>Date</TableHead>
                    <TableHead>In</TableHead>
                    <TableHead>Out</TableHead>
                    <TableHead className="hidden text-right sm:table-cell">Worked</TableHead>
                    <TableHead className="hidden text-right md:table-cell">Late</TableHead>
                    <TableHead className="hidden text-right md:table-cell">OT</TableHead>
                    <TableHead className="hidden text-right lg:table-cell">Break</TableHead>
                    <TableHead>Status</TableHead>
                    {canApprove && <TableHead className="w-24 text-right">Review</TableHead>}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {isLoading &&
                    Array.from({ length: 4 }, (_, i) => (
                      <TableRow key={i}>
                        <TableCell colSpan={canApprove ? 10 : 9}>
                          <Skeleton className="h-9 w-full" />
                        </TableCell>
                      </TableRow>
                    ))}

                  {(records ?? []).map((r) => (
                    <TableRow key={r.id}>
                      <TableCell className="font-medium">{r.employee_name}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(r.clock_in).toLocaleDateString("en-PH", { month: "short", day: "numeric" })}
                      </TableCell>
                      <TableCell className="tabular-nums">{timeLabel(r.clock_in)}</TableCell>
                      <TableCell className="tabular-nums">{timeLabel(r.clock_out)}</TableCell>
                      <TableCell className="hidden text-right tabular-nums sm:table-cell">{workedLabel(r)}</TableCell>
                      <TableCell className="hidden text-right md:table-cell">
                        {r.late_minutes > 0 ? (
                          <span className="font-medium text-rose-600">{r.late_minutes}m</span>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell className="hidden text-right md:table-cell">
                        {r.overtime_minutes > 0 ? (
                          <span className="font-medium text-emerald-600">{r.overtime_minutes}m</span>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell className="hidden text-right text-muted-foreground lg:table-cell">
                        {r.break_minutes > 0 ? `${r.break_minutes}m` : "—"}
                      </TableCell>
                      <TableCell>
                        {!r.clock_out ? (
                          <Badge className="bg-sky-600">On shift</Badge>
                        ) : r.status === "approved" ? (
                          <Badge className="bg-emerald-600">Approved</Badge>
                        ) : (
                          <Badge variant="secondary">Pending</Badge>
                        )}
                      </TableCell>
                      {canApprove && (
                        <TableCell className="text-right">
                          {r.clock_out && r.status === "pending" && (
                            <Button
                              variant="ghost"
                              size="sm"
                              disabled={approve.isPending}
                              onClick={() => approve.mutate(r.id)}
                            >
                              {approve.isPending ? (
                                <Loader2 className="size-4 animate-spin" aria-hidden />
                              ) : (
                                <CheckCheck className="size-4" aria-hidden />
                              )}
                              Approve
                            </Button>
                          )}
                        </TableCell>
                      )}
                    </TableRow>
                  ))}

                  {records && records.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={canApprove ? 10 : 9} className="py-10 text-center text-muted-foreground">
                        No attendance records in this range.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </section>
      )}
    </div>
  );
}
