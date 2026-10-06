import { usePage } from "@inertiajs/react";

import { Breadcrumbs } from "@/components/breadcrumbs";
import { ThemeCustomizer } from "@/components/theme-customizer";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { useTheme } from "@/lib/theme";
import { cn } from "@/lib/utils";
import {
  type BreadcrumbItem as BreadcrumbItemType,
  type SharedProps,
} from "@/types";

export function AppSidebarHeader({
  breadcrumbs = [],
}: {
  breadcrumbs?: BreadcrumbItemType[];
}) {
  const theme = useTheme(usePage<SharedProps>().props.theme);

  return (
    <header className="border-sidebar-border/50 flex h-16 shrink-0 items-center gap-2 border-b px-6 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
      <div className="flex items-center gap-2">
        {/* Sidebar mode "default" never collapses on desktop; phones still
            need the trigger to open the sheet. */}
        <SidebarTrigger
          className={cn("-ml-1", theme.sidebarMode === "default" && "md:hidden")}
        />
        <Breadcrumbs breadcrumbs={breadcrumbs} />
      </div>
      <div className="ml-auto">
        <ThemeCustomizer />
      </div>
    </header>
  );
}
