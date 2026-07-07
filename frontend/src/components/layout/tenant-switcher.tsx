"use client";

import { Check, ChevronsUpDown, Store } from "lucide-react";

import { cn } from "@/lib/utils";
import { useSwitchTenant } from "@/hooks/use-auth";
import type { Membership } from "@/types/auth";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";

interface TenantSwitcherProps {
  memberships: Membership[];
  activeTenant: Membership | null;
}

export function TenantSwitcher({ memberships, activeTenant }: TenantSwitcherProps) {
  const switchTenant = useSwitchTenant();

  if (!activeTenant) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="w-full justify-between gap-2 px-3 hover:bg-sidebar-accent"
        >
          <span className="flex min-w-0 items-center gap-2">
            <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Store className="size-4" aria-hidden />
            </span>
            <span className="truncate text-sm font-semibold">
              {activeTenant.tenant_name}
            </span>
          </span>
          <ChevronsUpDown className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-64">
        <DropdownMenuLabel>Your businesses</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {memberships.map((m) => (
          <DropdownMenuItem
            key={m.tenant_id}
            className={cn("gap-2", switchTenant.isPending && "pointer-events-none opacity-60")}
            onSelect={() => {
              if (m.tenant_id !== activeTenant.tenant_id) {
                switchTenant.mutate(m.tenant_id);
              }
            }}
          >
            <span className="flex-1 truncate">{m.tenant_name}</span>
            <span className="text-xs capitalize text-muted-foreground">{m.role}</span>
            {m.tenant_id === activeTenant.tenant_id && (
              <Check className="size-4 text-primary" aria-hidden />
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
