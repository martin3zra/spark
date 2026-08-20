import { Link } from "@inertiajs/react";
import { ChevronRight, Home } from "lucide-react";

import { Breadcrumbs } from "@/components/breadcrumbs";
import { type BreadcrumbItem as BreadcrumbItemType } from "@/types";

export function AppTopNavHeader({
  breadcrumbs = [],
}: {
  breadcrumbs?: BreadcrumbItemType[];
}) {
  if (breadcrumbs.length === 0) return null;

  return (
    <div className="dark border-b bg-background text-foreground">
      <div className="mx-auto flex max-w-7xl items-center gap-1.5 px-6 py-2 text-sm text-muted-foreground">
        <Link
          href="/home"
          className="flex items-center transition-colors hover:text-foreground"
        >
          <Home className="size-3.5" />
        </Link>
        <ChevronRight className="size-3.5" />
        <Breadcrumbs breadcrumbs={breadcrumbs} />
      </div>
    </div>
  );
}
