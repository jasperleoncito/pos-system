"use client";

import { useAuth } from "@/hooks/use-auth";
import { can } from "@/lib/rbac";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ProductsPanel } from "@/components/menu/products-panel";
import { CategoriesPanel } from "@/components/menu/categories-panel";
import { ModifiersPanel } from "@/components/menu/modifiers-panel";
import { TaxesPanel } from "@/components/menu/taxes-panel";

export default function MenuPage() {
  const { auth } = useAuth();
  const canWrite = can(auth?.activeTenant?.role, "catalog:write");

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold tracking-tight">Menu</h1>
        <p className="text-muted-foreground">
          Products, categories, modifiers, and taxes
        </p>
      </header>

      <Tabs defaultValue="products" className="space-y-4">
        <TabsList className="h-11">
          <TabsTrigger value="products" className="min-h-9 px-4">Products</TabsTrigger>
          <TabsTrigger value="categories" className="min-h-9 px-4">Categories</TabsTrigger>
          <TabsTrigger value="modifiers" className="min-h-9 px-4">Modifiers</TabsTrigger>
          <TabsTrigger value="taxes" className="min-h-9 px-4">Taxes</TabsTrigger>
        </TabsList>

        <TabsContent value="products">
          <ProductsPanel canWrite={canWrite} />
        </TabsContent>
        <TabsContent value="categories">
          <CategoriesPanel canWrite={canWrite} />
        </TabsContent>
        <TabsContent value="modifiers">
          <ModifiersPanel canWrite={canWrite} />
        </TabsContent>
        <TabsContent value="taxes">
          <TaxesPanel canWrite={canWrite} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
