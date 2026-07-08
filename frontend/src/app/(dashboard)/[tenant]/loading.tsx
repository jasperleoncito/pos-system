import { Skeleton } from "@/components/ui/skeleton";

/**
 * Instant route-transition feedback: shown the moment a sidebar link is
 * clicked, while the next page's code and data load. Without this the
 * old page freezes silently and navigation feels laggy.
 */
export default function TenantRouteLoading() {
  return (
    <div className="space-y-6" aria-busy="true" aria-label="Loading page">
      <div className="space-y-2">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-4 w-80" />
      </div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-24 w-full rounded-xl" />
        ))}
      </div>
      <Skeleton className="h-72 w-full rounded-xl" />
    </div>
  );
}
