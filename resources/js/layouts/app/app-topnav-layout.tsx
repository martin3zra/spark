import { type PropsWithChildren } from "react";

import { AppContent } from "@/components/app-content";
import { AppShell } from "@/components/app-shell";
import { AppTopNav } from "@/components/app-topnav";
import { AppTopNavHeader } from "@/components/app-topnav-header";
import { type BreadcrumbItem } from "@/types";

export default function AppTopNavLayout({
  children,
  breadcrumbs = [],
}: PropsWithChildren<{ breadcrumbs?: BreadcrumbItem[] }>) {
  return (
    <AppShell variant="header">
      <AppTopNav />
      <AppTopNavHeader breadcrumbs={breadcrumbs} />
      <AppContent variant="header">{children}</AppContent>
    </AppShell>
  );
}
