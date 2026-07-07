"use client";

import { useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { useCreateExpense, useDeleteExpense, useExpenses } from "@/hooks/use-analytics";
import { formatCentavos, pesosToCentavos } from "@/lib/currency";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const CATEGORIES = ["rent", "utilities", "supplies", "salaries", "other"];

interface ExpensesCardProps {
  from: string;
  to: string;
}

/** Expense list for the selected range with quick add/delete. */
export function ExpensesCard({ from, to }: ExpensesCardProps) {
  const { data: expenses } = useExpenses(from, to);
  const create = useCreateExpense();
  const remove = useDeleteExpense();

  const [open, setOpen] = useState(false);
  const [category, setCategory] = useState("supplies");
  const [description, setDescription] = useState("");
  const [amountPesos, setAmountPesos] = useState("");
  const [date, setDate] = useState("");

  const total = (expenses ?? []).reduce((sum, e) => sum + e.amount, 0);

  const submit = () => {
    const amount = pesosToCentavos(Number(amountPesos) || 0);
    if (!description.trim() || amount <= 0) {
      toast.error("Description and a positive amount are required");
      return;
    }
    create.mutate(
      {
        category,
        description: description.trim(),
        amount,
        expense_date: date || undefined,
      },
      {
        onSuccess: () => {
          setOpen(false);
          setDescription("");
          setAmountPesos("");
          setDate("");
        },
      },
    );
  };

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between pb-2">
        <CardTitle className="text-base">
          Expenses <span className="font-normal text-muted-foreground">· {formatCentavos(total)}</span>
        </CardTitle>
        <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
          <Plus className="size-4" aria-hidden />
          Add
        </Button>
      </CardHeader>
      <CardContent className="space-y-2">
        {(expenses ?? []).map((expense) => (
          <div key={expense.id} className="flex items-center justify-between gap-2 rounded-lg border p-2.5 text-sm">
            <div className="min-w-0">
              <p className="truncate font-medium">{expense.description}</p>
              <p className="text-xs text-muted-foreground">
                {new Date(expense.expense_date).toLocaleDateString("en-PH", { month: "short", day: "numeric" })}{" "}
                <Badge variant="secondary" className="ml-1 capitalize">{expense.category}</Badge>
              </p>
            </div>
            <div className="flex shrink-0 items-center gap-1">
              <span className="font-semibold tabular-nums">{formatCentavos(expense.amount)}</span>
              <Button
                variant="ghost"
                size="icon"
                className="size-7"
                aria-label={`Delete ${expense.description}`}
                disabled={remove.isPending}
                onClick={() => remove.mutate(expense.id)}
              >
                <Trash2 className="size-3.5 text-destructive" aria-hidden />
              </Button>
            </div>
          </div>
        ))}
        {expenses && expenses.length === 0 && (
          <p className="py-6 text-center text-sm text-muted-foreground">No expenses in range.</p>
        )}
      </CardContent>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Record expense</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Category</Label>
                <Select value={category} onValueChange={setCategory}>
                  <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {CATEGORIES.map((c) => (
                      <SelectItem key={c} value={c} className="capitalize">{c}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="x-amount">Amount (PHP)</Label>
                <Input
                  id="x-amount"
                  type="number"
                  min="0"
                  step="0.01"
                  value={amountPesos}
                  onChange={(e) => setAmountPesos(e.target.value)}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="x-desc">Description</Label>
              <Input id="x-desc" value={description} onChange={(e) => setDescription(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="x-date">Date (default today)</Label>
              <Input id="x-date" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
            <Button onClick={submit} disabled={create.isPending}>
              {create.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
              Record
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
