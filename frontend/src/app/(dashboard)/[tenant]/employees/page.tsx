"use client";

import { useState } from "react";
import { CalendarClock, Pencil, Plus, Search, Trash2 } from "lucide-react";

import { useDeleteEmployee, useEmployees, type Employee } from "@/hooks/use-employees";
import { useAuth } from "@/hooks/use-auth";
import { can } from "@/lib/rbac";
import { cn } from "@/lib/utils";
import { formatCentavos } from "@/lib/currency";
import { EmployeeFormDialog } from "@/components/employees/employee-form-dialog";
import { ScheduleDialog } from "@/components/employees/schedule-dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const SALARY_LABEL: Record<Employee["salary_type"], string> = {
  hourly: "/ hour",
  daily: "/ day",
  monthly: "/ month",
};

export default function EmployeesPage() {
  const { auth } = useAuth();
  const canWrite = can(auth?.activeTenant?.role, "employees:write");

  const [search, setSearch] = useState("");
  const { data: employees, isLoading } = useEmployees(search);
  const deleteEmployee = useDeleteEmployee();

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Employee | null>(null);
  const [scheduling, setScheduling] = useState<Employee | null>(null);
  const [removing, setRemoving] = useState<Employee | null>(null);

  const openForm = (employee: Employee | null) => {
    setEditing(employee);
    setFormOpen(true);
  };

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Employees</h1>
        <p className="text-muted-foreground">Staff directory, salaries, and weekly schedules</p>
      </header>

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-48 flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input
            placeholder="Search by name or position…"
            className="pl-9"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        {canWrite && (
          <Button onClick={() => openForm(null)}>
            <Plus className="size-4" aria-hidden />
            New employee
          </Button>
        )}
      </div>

      <Card className="py-0">
        <CardContent className="overflow-x-auto p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Employee</TableHead>
                <TableHead className="hidden md:table-cell">Contact</TableHead>
                <TableHead className="hidden sm:table-cell">Salary</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-32 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading &&
                Array.from({ length: 4 }, (_, i) => (
                  <TableRow key={i}>
                    <TableCell colSpan={5}><Skeleton className="h-11 w-full" /></TableCell>
                  </TableRow>
                ))}

              {(employees ?? []).map((employee) => (
                <TableRow key={employee.id} className={cn(!employee.is_active && "opacity-50")}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <Avatar className="size-9">
                        <AvatarImage src={employee.thumb_url || undefined} alt="" />
                        <AvatarFallback>{employee.full_name.slice(0, 2).toUpperCase()}</AvatarFallback>
                      </Avatar>
                      <div>
                        <p className="font-medium">{employee.full_name}</p>
                        <p className="text-xs text-muted-foreground">{employee.position || "—"}</p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell className="hidden text-sm text-muted-foreground md:table-cell">
                    <p>{employee.phone || "—"}</p>
                    {employee.user_email && (
                      <p className="text-xs">login: {employee.user_email}</p>
                    )}
                  </TableCell>
                  <TableCell className="hidden text-sm sm:table-cell">
                    <span className="font-medium tabular-nums">{formatCentavos(employee.salary_rate)}</span>
                    <span className="text-xs text-muted-foreground"> {SALARY_LABEL[employee.salary_type]}</span>
                  </TableCell>
                  <TableCell>
                    {employee.is_active ? (
                      <Badge className="bg-emerald-600">Active</Badge>
                    ) : (
                      <Badge variant="secondary">Inactive</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      {canWrite && (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label={`Schedule for ${employee.full_name}`}
                            onClick={() => setScheduling(employee)}
                          >
                            <CalendarClock className="size-4" aria-hidden />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label={`Edit ${employee.full_name}`}
                            onClick={() => openForm(employee)}
                          >
                            <Pencil className="size-4" aria-hidden />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label={`Remove ${employee.full_name}`}
                            onClick={() => setRemoving(employee)}
                          >
                            <Trash2 className="size-4 text-destructive" aria-hidden />
                          </Button>
                        </>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}

              {employees && employees.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="py-10 text-center text-muted-foreground">
                    No employees yet.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <EmployeeFormDialog open={formOpen} onOpenChange={setFormOpen} employee={editing} />
      <ScheduleDialog employee={scheduling} onOpenChange={(open) => !open && setScheduling(null)} />

      <AlertDialog open={removing !== null} onOpenChange={(open) => !open && setRemoving(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {removing?.full_name}?</AlertDialogTitle>
            <AlertDialogDescription>
              The profile is archived and attendance history is kept. This does not delete their login account.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (removing) deleteEmployee.mutate(removing.id);
                setRemoving(null);
              }}
            >
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
