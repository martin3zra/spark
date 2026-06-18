import { SidebarProvider } from "@/components/ui/sidebar";

export function AppShell({ children, variant = "sidebar" }) {
  if (variant === "header") {
    return <div className="flex min-h-screen w-full flex-col">{children}</div>;
  }

  return <SidebarProvider defaultOpen>{children}</SidebarProvider>;
}
