"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { ChevronRight, CreditCard, MonitorSmartphone, Palette, ScrollText, UsersRound } from "lucide-react";

import { useAuth } from "@/hooks/use-auth";
import { can } from "@/lib/rbac";
import { Card, CardContent } from "@/components/ui/card";

export default function SettingsPage() {
  const params = useParams<{ tenant: string }>();
  const { auth } = useAuth();

  const sections = [
    {
      href: `/${params.tenant}/settings/devices`,
      icon: MonitorSmartphone,
      title: "Devices",
      description: "Manage sessions signed in to your account",
    },
    {
      href: `/${params.tenant}/settings/branding`,
      icon: Palette,
      title: "Branding",
      description: "Logo, colors, and receipt details",
    },
    ...(can(auth?.activeTenant?.role, "billing:manage")
      ? [{
          href: `/${params.tenant}/settings/billing`,
          icon: CreditCard,
          title: "Billing",
          description: "Subscription plan, payments, and renewal",
        }]
      : []),
    ...(can(auth?.activeTenant?.role, "users:manage")
      ? [{
          href: `/${params.tenant}/settings/team`,
          icon: UsersRound,
          title: "Team",
          description: "Invite staff accounts and manage their roles",
        }]
      : []),
    ...(can(auth?.activeTenant?.role, "audit:read")
      ? [{
          href: `/${params.tenant}/settings/audit`,
          icon: ScrollText,
          title: "Audit log",
          description: "Every change made in this business, who made it, and when",
        }]
      : []),
  ];

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">Manage your business and account</p>
      </header>

      <div className="space-y-3">
        {sections.map(({ href, icon: Icon, title, description }) => (
          <Link key={href} href={href} className="block">
            <Card className="cursor-pointer py-0 transition-colors hover:bg-accent/5">
              <CardContent className="flex items-center gap-4 p-4">
                <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                  <Icon className="size-5 text-muted-foreground" aria-hidden />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">{title}</p>
                  <p className="truncate text-xs text-muted-foreground">{description}</p>
                </div>
                <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}
