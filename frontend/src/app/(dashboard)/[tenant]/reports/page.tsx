"use client";

import { useState } from "react";
import { Download, FileSpreadsheet, FileText, Loader2 } from "lucide-react";

import {
  REPORT_TYPES,
  downloadReport,
  useReport,
  type ReportColumn,
  type ReportType,
} from "@/hooks/use-reports";
import { formatCentavos } from "@/lib/currency";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

/** Local YYYY-MM-DD. */
function isoDate(d: Date): string {
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${month}-${day}`;
}

function formatCell(value: unknown, kind: ReportColumn["kind"]): string {
  if (value === null || value === undefined || value === "") return "";
  if (kind === "money") return formatCentavos(Number(value));
  if (kind === "number") {
    const n = Number(value);
    return Number.isInteger(n) ? String(n) : n.toFixed(2);
  }
  return String(value);
}

export default function ReportsPage() {
  const [type, setType] = useState<ReportType>("sales");
  const [from, setFrom] = useState(() => isoDate(new Date(Date.now() - 29 * 86_400_000)));
  const [to, setTo] = useState(() => isoDate(new Date()));
  const [downloading, setDownloading] = useState<string | null>(null);

  const meta = REPORT_TYPES.find((r) => r.value === type)!;
  const { data: doc, isLoading } = useReport(type, from, to);

  const download = async (format: "csv" | "xlsx" | "pdf") => {
    setDownloading(format);
    await downloadReport(type, format, from, to);
    setDownloading(null);
  };

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Reports</h1>
        <p className="text-muted-foreground">Preview any report, then export it as CSV, Excel, or PDF</p>
      </header>

      {/* Report type chips */}
      <div className="flex flex-wrap gap-1.5">
        {REPORT_TYPES.map((report) => (
          <button
            key={report.value}
            type="button"
            onClick={() => setType(report.value)}
            className={cn(
              "min-h-10 cursor-pointer rounded-full border px-4 text-sm font-medium transition-colors",
              type === report.value
                ? "border-primary bg-primary text-primary-foreground"
                : "hover:bg-accent/10",
            )}
          >
            {report.label}
          </button>
        ))}
      </div>

      {/* Filters + downloads */}
      <div className="flex flex-wrap items-end gap-3">
        {meta.rangeful && (
          <>
            <div className="space-y-1.5">
              <Label htmlFor="r-from">From</Label>
              <Input id="r-from" type="date" value={from} max={to} onChange={(e) => setFrom(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="r-to">To</Label>
              <Input id="r-to" type="date" value={to} min={from} onChange={(e) => setTo(e.target.value)} />
            </div>
          </>
        )}
        <div className="ml-auto flex gap-2">
          <Button variant="outline" disabled={downloading !== null} onClick={() => download("csv")}>
            {downloading === "csv" ? <Loader2 className="size-4 animate-spin" aria-hidden /> : <FileText className="size-4" aria-hidden />}
            CSV
          </Button>
          <Button variant="outline" disabled={downloading !== null} onClick={() => download("xlsx")}>
            {downloading === "xlsx" ? <Loader2 className="size-4 animate-spin" aria-hidden /> : <FileSpreadsheet className="size-4" aria-hidden />}
            Excel
          </Button>
          <Button disabled={downloading !== null} onClick={() => download("pdf")}>
            {downloading === "pdf" ? <Loader2 className="size-4 animate-spin" aria-hidden /> : <Download className="size-4" aria-hidden />}
            PDF
          </Button>
        </div>
      </div>

      {/* Preview */}
      <Card className="py-0">
        <CardContent className="overflow-x-auto p-0">
          {isLoading || !doc ? (
            <div className="space-y-2 p-4">
              {Array.from({ length: 6 }, (_, i) => <Skeleton key={i} className="h-9 w-full" />)}
            </div>
          ) : (
            <>
              <div className="border-b px-4 py-3">
                <p className="font-semibold">{doc.title}</p>
                <p className="text-xs text-muted-foreground">{doc.subtitle}</p>
              </div>
              <Table>
                <TableHeader>
                  <TableRow>
                    {doc.columns.map((col) => (
                      <TableHead
                        key={col.key}
                        className={cn(col.kind !== "text" && "text-right")}
                      >
                        {col.label}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(doc.rows ?? []).map((row, i) => (
                    <TableRow key={i}>
                      {doc.columns.map((col) => (
                        <TableCell
                          key={col.key}
                          className={cn("whitespace-nowrap", col.kind !== "text" && "text-right tabular-nums")}
                        >
                          {formatCell(row[col.key], col.kind)}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                  {doc.totals && (
                    <TableRow className="bg-muted/50 font-semibold">
                      {doc.columns.map((col) => (
                        <TableCell
                          key={col.key}
                          className={cn("whitespace-nowrap", col.kind !== "text" && "text-right tabular-nums")}
                        >
                          {col.key in doc.totals!
                            ? formatCell(doc.totals![col.key], col.kind === "text" ? "text" : col.kind)
                            : ""}
                        </TableCell>
                      ))}
                    </TableRow>
                  )}
                  {doc.rows && doc.rows.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={doc.columns.length} className="py-10 text-center text-muted-foreground">
                        No data for this report.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
