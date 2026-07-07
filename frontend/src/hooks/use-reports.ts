"use client";

import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";

export interface ReportColumn {
  key: string;
  label: string;
  kind: "text" | "money" | "number";
}

export interface ReportDocument {
  title: string;
  subtitle: string;
  columns: ReportColumn[];
  rows: Record<string, unknown>[] | null;
  totals?: Record<string, unknown>;
}

export type ReportType =
  | "sales"
  | "inventory"
  | "employees"
  | "attendance"
  | "profit"
  | "tax"
  | "receipts";

export const REPORT_TYPES: { value: ReportType; label: string; rangeful: boolean }[] = [
  { value: "sales", label: "Sales", rangeful: true },
  { value: "profit", label: "Profit", rangeful: true },
  { value: "tax", label: "Tax", rangeful: true },
  { value: "receipts", label: "Receipts", rangeful: true },
  { value: "attendance", label: "Attendance", rangeful: true },
  { value: "inventory", label: "Inventory", rangeful: false },
  { value: "employees", label: "Employees", rangeful: false },
];

export function useReport(type: ReportType, from: string, to: string) {
  return useQuery({
    queryKey: ["reports", type, from, to],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<ReportDocument>>(`/reports/${type}`, {
        params: { from, to },
      });
      return res.data.data;
    },
  });
}

/** Streams a report file and triggers a browser download. */
export async function downloadReport(type: ReportType, format: "csv" | "xlsx" | "pdf", from: string, to: string) {
  try {
    const res = await api.get<Blob>(`/reports/${type}`, {
      params: { from, to, format },
      responseType: "blob",
    });
    const url = URL.createObjectURL(res.data);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${type}-report.${format}`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  } catch (error) {
    toast.error(getApiErrorMessage(error));
  }
}
