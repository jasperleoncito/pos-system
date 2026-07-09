"use client";

import { useState } from "react";
import { Crown, Loader2, MailPlus, Trash2, UserPlus } from "lucide-react";
import { toast } from "sonner";

import { useAuth } from "@/hooks/use-auth";
import {
  useInviteMember,
  useRemoveMember,
  useResendInvite,
  useTeam,
  useUpdateMemberRole,
} from "@/hooks/use-team";
import type { TeamMember } from "@/types/tenant";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
import { Skeleton } from "@/components/ui/skeleton";

const ASSIGNABLE_ROLES = [
  { value: "manager", label: "Manager" },
  { value: "cashier", label: "Cashier" },
  { value: "kitchen", label: "Kitchen" },
  { value: "employee", label: "Employee" },
];

function InviteDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const invite = useInviteMember();
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("cashier");

  const submit = () => {
    if (fullName.trim().length < 2) {
      toast.error("Full name is required");
      return;
    }
    if (!email.trim()) {
      toast.error("Email is required");
      return;
    }
    invite.mutate(
      { full_name: fullName.trim(), email: email.trim(), role },
      {
        onSuccess: () => {
          onOpenChange(false);
          setFullName("");
          setEmail("");
          setRole("cashier");
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Invite a team member</DialogTitle>
          <DialogDescription>
            New emails get an invitation to set their password. Existing accounts are attached
            with the chosen role right away.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="t-name">Full name</Label>
            <Input id="t-name" value={fullName} onChange={(e) => setFullName(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="t-email">Email</Label>
            <Input
              id="t-email"
              type="email"
              placeholder="staff@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>Role</Label>
            <Select value={role} onValueChange={setRole}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {ASSIGNABLE_ROLES.map((r) => (
                  <SelectItem key={r.value} value={r.value}>
                    {r.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={submit} disabled={invite.isPending}>
            {invite.isPending && <Loader2 className="size-4 animate-spin" aria-hidden />}
            Send invite
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function MemberRow({ member, selfUserId }: { member: TeamMember; selfUserId: string }) {
  const updateRole = useUpdateMemberRole();
  const removeMember = useRemoveMember();
  const resendInvite = useResendInvite();

  const isSelf = member.user_id === selfUserId;
  const roleLocked = member.is_owner || isSelf;

  return (
    <div className="flex flex-wrap items-center gap-3 rounded-lg border p-3">
      <div className="min-w-0 flex-1">
        <p className="flex items-center gap-2 text-sm font-medium">
          <span className="truncate">{member.full_name}</span>
          {member.is_owner && (
            <Badge className="shrink-0 gap-1 bg-amber-600 text-white">
              <Crown className="size-3" aria-hidden /> Owner
            </Badge>
          )}
          {isSelf && !member.is_owner && (
            <Badge variant="secondary" className="shrink-0">
              You
            </Badge>
          )}
          {!member.email_verified_at && !member.is_owner && (
            <Badge variant="outline" className="shrink-0 text-muted-foreground">
              Invited
            </Badge>
          )}
        </p>
        <p className="truncate text-xs text-muted-foreground">
          {member.email} · joined {new Date(member.joined_at).toLocaleDateString()}
        </p>
      </div>

      {roleLocked ? (
        <Badge variant="secondary" className="capitalize">
          {member.role}
        </Badge>
      ) : (
        <Select
          value={member.role}
          onValueChange={(role) => updateRole.mutate({ userId: member.user_id, role })}
          disabled={updateRole.isPending}
        >
          <SelectTrigger className="h-9 w-[130px]" aria-label={`Role for ${member.full_name}`}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {ASSIGNABLE_ROLES.map((r) => (
              <SelectItem key={r.value} value={r.value}>
                {r.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}

      {!member.is_owner && !isSelf && (
        <div className="flex items-center gap-1">
          {!member.email_verified_at && (
            <Button
              variant="ghost"
              size="icon"
              title="Resend invite email"
              aria-label={`Resend invite to ${member.full_name}`}
              disabled={resendInvite.isPending}
              onClick={() => resendInvite.mutate(member.user_id)}
            >
              {resendInvite.isPending ? (
                <Loader2 className="size-4 animate-spin" aria-hidden />
              ) : (
                <MailPlus className="size-4" aria-hidden />
              )}
            </Button>
          )}

          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="text-destructive hover:text-destructive"
                aria-label={`Remove ${member.full_name}`}
              >
                <Trash2 className="size-4" aria-hidden />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Remove {member.full_name}?</AlertDialogTitle>
                <AlertDialogDescription>
                  They will lose access to this business. Their account is kept — you can invite
                  them again later.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  className="bg-destructive text-white hover:bg-destructive/90"
                  onClick={() => removeMember.mutate(member.user_id)}
                >
                  Remove
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      )}
    </div>
  );
}

export default function TeamPage() {
  const { auth } = useAuth();
  const { data: members, isLoading } = useTeam();
  const [inviteOpen, setInviteOpen] = useState(false);

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight">Team</h1>
          <p className="text-muted-foreground">Staff accounts and their roles in this business</p>
        </div>
        <Button onClick={() => setInviteOpen(true)}>
          <UserPlus className="size-4" aria-hidden />
          Invite member
        </Button>
      </header>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Members</CardTitle>
          <CardDescription>
            {members ? `${members.length} member${members.length === 1 ? "" : "s"}` : "…"}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading &&
            Array.from({ length: 3 }, (_, i) => <Skeleton key={i} className="h-16 w-full" />)}

          {members?.map((m) => (
            <MemberRow key={m.user_id} member={m} selfUserId={auth?.user.id ?? ""} />
          ))}

          {members && members.length === 0 && (
            <p className="py-6 text-center text-sm text-muted-foreground">No members yet.</p>
          )}
        </CardContent>
      </Card>

      <InviteDialog open={inviteOpen} onOpenChange={setInviteOpen} />
    </div>
  );
}
