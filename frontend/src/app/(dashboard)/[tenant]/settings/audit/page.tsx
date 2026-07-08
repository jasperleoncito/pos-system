"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { api, type ApiEnvelope } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface AuditLog {
  id: string;
  user_name?: string;
  action: string;
  entity_type: string;
  entity_id: string;
  after?: Record<string, unknown>;
  created_at: string;
}

interface AuditMeta {
  total: number;
  page: number;
  limit: number;
}

const PAGE_SIZE = 50;

function useAuditLogs(page: number) {
  return useQuery({
    queryKey: ["audit-logs", page],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<AuditLog[] | null> & { meta?: AuditMeta }>("/audit-logs", {
        params: { page, limit: PAGE_SIZE },
      });
      return { items: res.data.data ?? [], meta: res.data.meta };
    },
  });
}

/** Compact "key: value" summary of the change payload. */
function changeSummary(after?: Record<string, unknown>): string {
  if (!after) return "";
  return Object.entries(after)
    .slice(0, 3)
    .map(([k, v]) => `${k}: ${String(v)}`)
    .join(" · ");
}

export default function AuditLogPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = useAuditLogs(page);

  const totalPages = data?.meta ? Math.max(1, Math.ceil(data.meta.total / PAGE_SIZE)) : 1;

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Audit log</h1>
        <p className="text-muted-foreground">Every change in this business — who, what, and when</p>
      </header>

      <Card className="py-0">
        <CardContent className="overflow-x-auto p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>When</TableHead>
                <TableHead>Who</TableHead>
                <TableHead>Action</TableHead>
                <TableHead className="hidden md:table-cell">Details</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading &&
                Array.from({ length: 8 }, (_, i) => (
                  <TableRow key={i}>
                    <TableCell colSpan={4}><Skeleton className="h-8 w-full" /></TableCell>
                  </TableRow>
                ))}
              {(data?.items ?? []).map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                    {new Date(log.created_at).toLocaleString("en-PH", {
                      month: "short", day: "numeric", hour: "2-digit", minute: "2-digit",
                    })}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-sm font-medium">
                    {log.user_name || "system"}
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary" className="font-mono text-xs">{log.action}</Badge>
                  </TableCell>
                  <TableCell className="hidden max-w-96 truncate text-sm text-muted-foreground md:table-cell">
                    {changeSummary(log.after)}
                  </TableCell>
                </TableRow>
              ))}
              {data && data.items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="py-10 text-center text-muted-foreground">
                    No audit entries yet.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          Page {page} of {totalPages}
          {data?.meta && ` · ${data.meta.total} entries`}
        </p>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            Previous
          </Button>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}
