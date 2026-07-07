"use client";

import { motion } from "motion/react";

import { useAuth } from "@/hooks/use-auth";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

export default function DashboardPage() {
  const { auth } = useAuth();
  if (!auth) return null;

  const firstName = auth.user.full_name.split(" ")[0];

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
      className="space-y-6"
    >
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">
          Welcome back, {firstName}
        </h1>
        <p className="text-muted-foreground">
          {auth.activeTenant?.tenant_name}{" "}
          <Badge variant="secondary" className="ml-1 capitalize">
            {auth.activeTenant?.role}
          </Badge>
        </p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Sales analytics</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            The analytics dashboard arrives in a later phase — today&apos;s sales,
            top products, hourly heatmaps, and more will live here.
          </p>
        </CardContent>
      </Card>
    </motion.div>
  );
}
