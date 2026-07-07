"use client";

import { useRef, useState } from "react";
import { ImagePlus, Loader2, Pencil, Plus, Search, Trash2 } from "lucide-react";

import {
  useCategories,
  useDeleteProduct,
  useProducts,
  useUploadProductImage,
} from "@/hooks/use-catalog";
import { formatCentavos } from "@/lib/currency";
import type { Product } from "@/types/catalog";
import { ProductFormDialog } from "@/components/menu/product-form-dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const ALL_CATEGORIES = "all";

export function ProductsPanel({ canWrite }: { canWrite: boolean }) {
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState(ALL_CATEGORIES);
  const [page, setPage] = useState(1);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Product | null>(null);
  const [deleting, setDeleting] = useState<Product | null>(null);
  const [uploadTarget, setUploadTarget] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { data: categories } = useCategories();
  const { data, isLoading } = useProducts({
    search,
    categoryId: categoryId === ALL_CATEGORIES ? "" : categoryId,
    page,
    limit: 25,
  });
  const deleteProduct = useDeleteProduct();
  const uploadImage = useUploadProductImage();

  const total = data?.meta?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / (data?.meta?.limit ?? 25)));

  const pickImage = (productId: string) => {
    setUploadTarget(productId);
    fileInputRef.current?.click();
  };

  return (
    <div className="space-y-4">
      <input
        ref={fileInputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file && uploadTarget) {
            uploadImage.mutate({ id: uploadTarget, file });
          }
          e.target.value = "";
        }}
      />

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-48 flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input
            placeholder="Search products or SKU…"
            className="pl-9"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <Select
          value={categoryId}
          onValueChange={(v) => {
            setCategoryId(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-48">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_CATEGORIES}>All categories</SelectItem>
            {(categories ?? []).map((c) => (
              <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        {canWrite && (
          <Button
            onClick={() => {
              setEditing(null);
              setFormOpen(true);
            }}
          >
            <Plus className="size-4" aria-hidden />
            New product
          </Button>
        )}
      </div>

      <Card className="py-0">
        <CardContent className="overflow-x-auto p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-14"></TableHead>
                <TableHead>Product</TableHead>
                <TableHead className="hidden sm:table-cell">Category</TableHead>
                <TableHead className="text-right">Price</TableHead>
                <TableHead className="hidden sm:table-cell">Status</TableHead>
                {canWrite && <TableHead className="w-28 text-right">Actions</TableHead>}
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading &&
                Array.from({ length: 6 }, (_, i) => (
                  <TableRow key={i}>
                    <TableCell colSpan={canWrite ? 6 : 5}>
                      <Skeleton className="h-10 w-full" />
                    </TableCell>
                  </TableRow>
                ))}

              {data?.products.map((p) => (
                <TableRow key={p.id}>
                  <TableCell>
                    <div className="flex size-10 items-center justify-center overflow-hidden rounded-lg border bg-muted">
                      {p.thumb_url ? (
                        // eslint-disable-next-line @next/next/no-img-element -- MinIO-served
                        <img src={p.thumb_url} alt="" width={40} height={40} className="size-full object-cover" />
                      ) : (
                        <ImagePlus className="size-4 text-muted-foreground" aria-hidden />
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <p className="font-medium">{p.name}</p>
                    {(p.variants?.length ?? 0) > 0 && (
                      <p className="text-xs text-muted-foreground">
                        {p.variants!.length} variants
                      </p>
                    )}
                  </TableCell>
                  <TableCell className="hidden text-muted-foreground sm:table-cell">
                    {p.category_name}
                  </TableCell>
                  <TableCell className="text-right font-medium tabular-nums">
                    {formatCentavos(p.base_price)}
                  </TableCell>
                  <TableCell className="hidden sm:table-cell">
                    <Badge variant={p.is_active ? "default" : "secondary"} className={p.is_active ? "bg-emerald-600" : undefined}>
                      {p.is_active ? "Active" : "Hidden"}
                    </Badge>
                  </TableCell>
                  {canWrite && (
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Upload image for ${p.name}`}
                          disabled={uploadImage.isPending && uploadTarget === p.id}
                          onClick={() => pickImage(p.id)}
                        >
                          {uploadImage.isPending && uploadTarget === p.id ? (
                            <Loader2 className="size-4 animate-spin" aria-hidden />
                          ) : (
                            <ImagePlus className="size-4" aria-hidden />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Edit ${p.name}`}
                          onClick={() => {
                            setEditing(p);
                            setFormOpen(true);
                          }}
                        >
                          <Pencil className="size-4" aria-hidden />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label={`Delete ${p.name}`}
                          onClick={() => setDeleting(p)}
                        >
                          <Trash2 className="size-4 text-destructive" aria-hidden />
                        </Button>
                      </div>
                    </TableCell>
                  )}
                </TableRow>
              ))}

              {data && data.products.length === 0 && (
                <TableRow>
                  <TableCell colSpan={canWrite ? 6 : 5} className="py-10 text-center text-muted-foreground">
                    No products found.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {pageCount > 1 && (
        <div className="flex items-center justify-between">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {page} of {pageCount} · {total} products
          </span>
          <Button variant="outline" size="sm" disabled={page >= pageCount} onClick={() => setPage((p) => p + 1)}>
            Next
          </Button>
        </div>
      )}

      <ProductFormDialog open={formOpen} onOpenChange={setFormOpen} product={editing} />

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {deleting?.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              The product is removed from the menu. Past sales keep their records.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deleting) deleteProduct.mutate(deleting.id);
                setDeleting(null);
              }}
            >
              Delete product
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
