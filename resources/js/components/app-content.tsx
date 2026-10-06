import { SidebarInset } from "@/components/ui/sidebar";

interface AppContentProps extends React.ComponentProps<"main"> {
  variant?: "header" | "sidebar";
}

export function AppContent({
  variant = "sidebar",
  children,
  ...props
}: AppContentProps) {
  // Content layout "compact" caps the page width; "full" (layout-full:,
  // see app.css) lifts the cap.
  if (variant === "sidebar") {
    return <SidebarInset {...props}>{children}</SidebarInset>;
  }

  return (
    <main
      className="mx-auto flex h-full w-full max-w-7xl flex-1 flex-col gap-4 rounded-xl layout-full:max-w-none"
      {...props}
    >
      {children}
    </main>
  );
}
