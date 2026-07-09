"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { api, getApiErrorMessage, type ApiEnvelope } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import type { TeamMember } from "@/types/tenant";

export interface InviteMemberInput {
  full_name: string;
  email: string;
  role: string;
}

interface InviteResult {
  member: TeamMember;
  user_created: boolean;
}

export function useTeam() {
  const { auth } = useAuth();
  return useQuery({
    queryKey: ["team", auth?.activeTenant?.tenant_id],
    queryFn: async () => {
      const res = await api.get<ApiEnvelope<TeamMember[]>>("/team");
      return res.data.data ?? [];
    },
    enabled: Boolean(auth?.activeTenant),
  });
}

export function useInviteMember() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: InviteMemberInput) => {
      const res = await api.post<ApiEnvelope<InviteResult>>("/team", input);
      return res.data;
    },
    onSuccess: (envelope) => {
      toast.success(envelope.message || "Member added");
      queryClient.invalidateQueries({ queryKey: ["team"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useUpdateMemberRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ userId, role }: { userId: string; role: string }) => {
      await api.patch(`/team/${userId}/role`, { role });
    },
    onSuccess: () => {
      toast.success("Role updated");
      queryClient.invalidateQueries({ queryKey: ["team"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useRemoveMember() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (userId: string) => {
      await api.delete(`/team/${userId}`);
    },
    onSuccess: () => {
      toast.success("Member removed");
      queryClient.invalidateQueries({ queryKey: ["team"] });
    },
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}

export function useResendInvite() {
  return useMutation({
    mutationFn: async (userId: string) => {
      await api.post(`/team/${userId}/resend-invite`);
    },
    onSuccess: () => toast.success("Invite email sent"),
    onError: (error) => toast.error(getApiErrorMessage(error)),
  });
}
