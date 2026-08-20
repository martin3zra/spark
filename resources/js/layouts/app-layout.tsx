import { type PropsWithChildren } from "react";

import AppSidebarLayout from "@/layouts/app/app-sidebar-layout";
import AppTopNavLayout from "@/layouts/app/app-topnav-layout";
import { type BreadcrumbItem } from "@/types";

// VITE_APP_LAYOUT selects the app shell: "sidebar" (default) or "topnav".
// Set it in .env and rebuild — it's read at build time, not per-request.
const AppLayoutTemplate =
  import.meta.env.VITE_APP_LAYOUT === "topnav"
    ? AppTopNavLayout
    : AppSidebarLayout;

interface AppLayoutProps extends PropsWithChildren {
  breadcrumbs?: BreadcrumbItem[];
}

export default function AppLayout({
  children,
  breadcrumbs = [],
  ...props
}: AppLayoutProps) {
  return (
    <AppLayoutTemplate breadcrumbs={breadcrumbs} {...props}>
      {children}
    </AppLayoutTemplate>
  );
}
