"use client";

import { useEffect, useRef, useState } from "react";
import { Camera, Loader2 } from "lucide-react";
import { toast } from "sonner";

import {
  useSaveEmployee,
  useUploadEmployeePhoto,
  type Employee,
} from "@/hooks/use-employees";
import { pesosToCentavos } from "@/lib/currency";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";

interface EmployeeFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  employee: Employee | null; // null = create
}

const SALARY_TYPES = [
  { value: "hourly", label: "Hourly" },
  { value: "daily", label: "Daily" },
  { value: "monthly", label: "Monthly" },
];

export function EmployeeFormDialog({ open, onOpenChange, employee }: EmployeeFormDialogProps) {
  const save = useSaveEmployee();
  const uploadPhoto = useUploadEmployeePhoto();
  const fileRef = useRef<HTMLInputElement>(null);

  const [fullName, setFullName] = useState("");
  const [position, setPosition] = useState("");
  const [phone, setPhone] = useState("");
  const [email, setEmail] = useState("");
  const [address, setAddress] = useState("");
  const [salaryType, setSalaryType] = useState("daily");
  const [ratePesos, setRatePesos] = useState("");
  const [hireDate, setHireDate] = useState("");
  const [userEmail, setUserEmail] = useState("");
  const [notes, setNotes] = useState("");
  const [isActive, setIsActive] = useState(true);

  useEffect(() => {
    if (!open) return;
    setFullName(employee?.full_name ?? "");
    setPosition(employee?.position ?? "");
    setPhone(employee?.phone ?? "");
    setEmail(employee?.email ?? "");
    setAddress(employee?.address ?? "");
    setSalaryType(employee?.salary_type ?? "daily");
    setRatePesos(employee ? (employee.salary_rate / 100).toString() : "");
    setHireDate(employee?.hire_date ? employee.hire_date.slice(0, 10) : "");
    setUserEmail(employee?.user_email ?? "");
    setNotes(employee?.notes ?? "");
    setIsActive(employee?.is_active ?? true);
  }, [open, employee]);

  const submit = () => {
    if (!fullName.trim()) {
      toast.error("Full name is required");
      return;
    }
    save.mutate(
      {
        id: employee?.id,
        input: {
          full_name: fullName.trim(),
          position: position.trim(),
          phone: phone.trim(),
          email: email.trim(),
          address: address.trim(),
          salary_type: salaryType,
          salary_rate: pesosToCentavos(Number(ratePesos) || 0),
          hire_date: hireDate || undefined,
          notes: notes.trim(),
          is_active: isActive,
          user_email: userEmail.trim() || undefined,
        },
      },
      { onSuccess: () => onOpenChange(false) },
    );
  };

  const onPhotoPicked = (file: File | undefined) => {
    if (!file || !employee) return;
    uploadPhoto.mutate({ id: employee.id, file });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{employee ? "Edit employee" : "New employee"}</DialogTitle>
          <DialogDescription>
            Link a login email so this person can use the self-service clock.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {employee && (
            <div className="flex items-center gap-3">
              <Avatar className="size-16">
                <AvatarImage src={employee.thumb_url || undefined} alt="" />
                <AvatarFallback>{employee.full_name.slice(0, 2).toUpperCase()}</AvatarFallback>
              </Avatar>
              <input
                ref={fileRef}
                type="file"
                accept="image/png,image/jpeg,image/webp"
                className="hidden"
                onChange={(e) => onPhotoPicked(e.target.files?.[0])}
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={uploadPhoto.isPending}
                onClick={() => fileRef.current?.click()}
              >
                {uploadPhoto.isPending ? (
                  <Loader2 className="size-4 animate-spin" aria-hidden />
                ) : (
                  <Camera className="size-4" aria-hidden />
                )}
                Change photo
              </Button>
            </div>
          )}

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="e-name">Full name</Label>
              <Input id="e-name" value={fullName} onChange={(e) => setFullName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="e-position">Position</Label>
              <Input id="e-position" placeholder="Cashier, Cook…" value={position} onChange={(e) => setPosition(e.target.value)} />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="e-phone">Phone</Label>
              <Input id="e-phone" value={phone} onChange={(e) => setPhone(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="e-email">Contact email</Label>
              <Input id="e-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="e-address">Address</Label>
            <Input id="e-address" value={address} onChange={(e) => setAddress(e.target.value)} />
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            <div className="space-y-2">
              <Label>Salary type</Label>
              <Select value={salaryType} onValueChange={setSalaryType}>
                <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {SALARY_TYPES.map((t) => (
                    <SelectItem key={t.value} value={t.value}>{t.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="e-rate">Rate (PHP)</Label>
              <Input id="e-rate" type="number" min="0" step="0.01" value={ratePesos} onChange={(e) => setRatePesos(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="e-hired">Hire date</Label>
              <Input id="e-hired" type="date" value={hireDate} onChange={(e) => setHireDate(e.target.value)} />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="e-user">Login email (optional)</Label>
            <Input
              id="e-user"
              type="email"
              placeholder="Existing member account, e.g. cashier@teresas.ph"
              value={userEmail}
              onChange={(e) => setUserEmail(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Must already be a member of this restaurant. Enables clock in/out for this person.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="e-notes">Notes</Label>
            <Textarea id="e-notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
          </div>

          {employee && (
            <div className="flex items-center justify-between rounded-lg border p-3">
              <div>
                <p className="text-sm font-medium">Active</p>
                <p className="text-xs text-muted-foreground">Inactive employees cannot clock in.</p>
              </div>
              <Switch checked={isActive} onCheckedChange={setIsActive} aria-label="Active" />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={submit} disabled={save.isPending}>
            {save.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            {employee ? "Save" : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
