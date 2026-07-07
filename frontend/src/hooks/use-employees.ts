"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";

export interface Employee {
  id: string;
  user_id: string | null;
  user_email?: string;
  full_name: string;
  position: string;
  phone: string;
  email: string;
  address: string;
  salary_type: "hourly" | "daily" | "monthly";
  salary_rate: number; // centavos
  hire_date: string | null;
  photo_url: string;
  thumb_url: string;
  notes: string;
  is_active: boolean;
  created_at: string;
}

export interface ScheduleDay {
  day_of_week: number; // 0 = Sunday
  start_time: string; // "HH:MM"
  end_time: string;
  grace_minutes: number;
}

export interface EmployeeInput {
  full_name: string;
  position: string;
  phone: string;
  email: string;
  address: string;
  salary_type: string;
  salary_rate: number;
  hire_date?: string;
  notes: string;
  is_active: boolean;
  user_email?: string;
}

export function useEmployees(search = "") {
  return useQuery({
    queryKey: ["employees", "list", search],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<Employee[] | null>>("/employees", {
        params: search ? { search } : undefined,
      });
      return res.data.data ?? [];
    },
  });
}

export function useSaveEmployee() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id?: string; input: EmployeeInput }) => {
      if (id) {
        const res = await api.put<ApiEnvelope<Employee>>(`/employees/${id}`, input);
        return res.data.data;
      }
      const res = await api.post<ApiEnvelope<Employee>>("/employees", input);
      return res.data.data;
    },
    onSuccess: (_, { id }) => {
      toast.success(id ? "Employee updated" : "Employee created");
      queryClient.invalidateQueries({ queryKey: ["employees"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useDeleteEmployee() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => api.delete(`/employees/${id}`),
    onSuccess: () => {
      toast.success("Employee removed");
      queryClient.invalidateQueries({ queryKey: ["employees"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useUploadEmployeePhoto() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, file }: { id: string; file: File }) => {
      const form = new FormData();
      form.append("image", file);
      const res = await api.post<ApiEnvelope<Employee>>(`/employees/${id}/photo`, form, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Photo updated");
      queryClient.invalidateQueries({ queryKey: ["employees"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useEmployeeSchedule(employeeId: string | null) {
  return useQuery({
    queryKey: ["employees", "schedule", employeeId],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<ScheduleDay[] | null>>(`/employees/${employeeId}/schedule`);
      return res.data.data ?? [];
    },
    enabled: Boolean(employeeId),
  });
}

export function useSaveSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ employeeId, days }: { employeeId: string; days: ScheduleDay[] }) => {
      const res = await api.put<ApiEnvelope<ScheduleDay[]>>(`/employees/${employeeId}/schedule`, { days });
      return res.data.data;
    },
    onSuccess: () => {
      toast.success("Schedule saved");
      queryClient.invalidateQueries({ queryKey: ["employees", "schedule"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
