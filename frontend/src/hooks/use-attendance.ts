"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import type { Employee, ScheduleDay } from "@/hooks/use-employees";

export interface AttendanceRecord {
  id: string;
  employee_id: string;
  employee_name?: string;
  clock_in: string;
  clock_out: string | null;
  scheduled_start: string | null;
  scheduled_end: string | null;
  break_start: string | null;
  break_minutes: number;
  late_minutes: number;
  early_out_minutes: number;
  overtime_minutes: number;
  status: "pending" | "approved";
  notes: string;
  created_at: string;
}

export interface ClockStatus {
  employee: Employee;
  today_schedule: ScheduleDay | null;
  open: AttendanceRecord | null;
  server_time: string;
}

/** Self-service clock state; 404-ish validation errors mean "no profile linked". */
export function useClockStatus(enabled: boolean) {
  return useQuery({
    queryKey: ["attendance", "me"],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<ClockStatus>>("/attendance/me");
      return res.data.data;
    },
    enabled,
    retry: false,
    refetchInterval: 60_000,
  });
}

function useClockMutation(path: string, successMessage: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await api.post<ApiEnvelope<AttendanceRecord>>(path);
      return res.data.data;
    },
    onSuccess: () => {
      toast.success(successMessage);
      queryClient.invalidateQueries({ queryKey: ["attendance"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useClockIn() {
  return useClockMutation("/attendance/clock-in", "Clocked in — have a great shift!");
}

export function useClockOut() {
  return useClockMutation("/attendance/clock-out", "Clocked out — see you next time!");
}

export function useStartBreak() {
  return useClockMutation("/attendance/break/start", "Break started");
}

export function useEndBreak() {
  return useClockMutation("/attendance/break/end", "Break ended");
}

export interface AttendanceFilters {
  employeeId?: string;
  from?: string; // YYYY-MM-DD
  to?: string;
}

export function useAttendanceRecords(filters: AttendanceFilters, enabled: boolean) {
  return useQuery({
    queryKey: ["attendance", "records", filters],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<AttendanceRecord[] | null>>("/attendance", {
        params: {
          employee_id: filters.employeeId || undefined,
          from: filters.from || undefined,
          to: filters.to || undefined,
        },
      });
      return res.data.data ?? [];
    },
    enabled,
  });
}

export function useApproveAttendance() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await api.post<ApiEnvelope<AttendanceRecord>>(`/attendance/${id}/approve`);
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Attendance approved");
      queryClient.invalidateQueries({ queryKey: ["attendance"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
