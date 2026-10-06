import { type PropsWithChildren } from "react";

import { AppContent } from "@/components/app-content";
import { AppShell } from "@/components/app-shell";
import { AppSidebar } from "@/components/app-sidebar";
import { AppSidebarHeader } from "@/components/app-sidebar-header";
import { type BreadcrumbItem } from "@/types";

export default function AppSidebarLayout({
  children,
  breadcrumbs = [],
}: PropsWithChildren<{ breadcrumbs?: BreadcrumbItem[] }>) {
  return (
    <AppShell variant="sidebar">
      <AppSidebar />
      <AppContent variant="sidebar">
        <AppSidebarHeader breadcrumbs={breadcrumbs} />
        {/* Content layout: "compact" caps the width, "full" lifts it. */}
        <div className="mx-auto flex w-full max-w-7xl flex-1 flex-col layout-full:max-w-none">
          {children}
        </div>
      </AppContent>
    </AppShell>
  );
}
