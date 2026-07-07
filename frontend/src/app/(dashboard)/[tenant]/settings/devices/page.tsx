"use client";

import { Laptop, Loader2, LogOut, Smartphone } from "lucide-react";

import { useLogoutAll, useRevokeSession, useSessions } from "@/hooks/use-auth";
import type { DeviceSession } from "@/types/auth";
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
import { Skeleton } from "@/components/ui/skeleton";

function deviceLabel(session: DeviceSession): string {
  if (session.device_name) return session.device_name;
  const ua = session.user_agent;
  if (/mobile|android|iphone/i.test(ua)) return "Mobile device";
  if (/ipad|tablet/i.test(ua)) return "Tablet";
  return "Computer";
}

function isMobile(session: DeviceSession): boolean {
  return /mobile|android|iphone|ipad|tablet/i.test(session.user_agent + session.device_name);
}

export default function DevicesPage() {
  const { data, isLoading } = useSessions();
  const revoke = useRevokeSession();
  const logoutAll = useLogoutAll();

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Devices</h1>
        <p className="text-muted-foreground">
          Sessions currently signed in to your account
        </p>
      </header>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div className="space-y-1">
            <CardTitle className="text-base">Active sessions</CardTitle>
            <CardDescription>Revoke anything you don&apos;t recognize</CardDescription>
          </div>

          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="outline" size="sm" disabled={logoutAll.isPending}>
                {logoutAll.isPending ? (
                  <Loader2 className="size-4 animate-spin" aria-hidden />
                ) : (
                  <LogOut className="size-4" aria-hidden />
                )}
                Sign out everywhere
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Sign out of all devices?</AlertDialogTitle>
                <AlertDialogDescription>
                  Every session including this one will be revoked, and you&apos;ll
                  need to sign in again.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction onClick={() => logoutAll.mutate()}>
                  Sign out everywhere
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </CardHeader>

        <CardContent className="space-y-3">
          {isLoading &&
            Array.from({ length: 2 }, (_, i) => <Skeleton key={i} className="h-16 w-full" />)}

          {data?.sessions.map((session) => {
            const isCurrent = session.id === data.current_session_id;
            const Icon = isMobile(session) ? Smartphone : Laptop;
            return (
              <div
                key={session.id}
                className="flex items-center gap-3 rounded-lg border p-3"
              >
                <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                  <Icon className="size-5 text-muted-foreground" aria-hidden />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="flex items-center gap-2 text-sm font-medium">
                    {deviceLabel(session)}
                    {isCurrent && <Badge className="bg-emerald-600">This device</Badge>}
                  </p>
                  <p className="truncate text-xs text-muted-foreground">
                    {session.ip} · last active{" "}
                    {new Date(session.last_used_at).toLocaleString()}
                  </p>
                </div>

                {!isCurrent && (
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="ghost" size="sm" disabled={revoke.isPending}>
                        Revoke
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Revoke this session?</AlertDialogTitle>
                        <AlertDialogDescription>
                          The device will be signed out immediately.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={() => revoke.mutate(session.id)}>
                          Revoke session
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                )}
              </div>
            );
          })}

          {data && data.sessions.length === 0 && (
            <p className="py-6 text-center text-sm text-muted-foreground">No active sessions.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
