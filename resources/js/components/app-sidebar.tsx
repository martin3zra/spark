import { Link, usePage } from "@inertiajs/react";

import AppLogo from "@/components/app-logo";
import { NavMain } from "@/components/nav-main";
import { NavUser } from "@/components/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { sidebarCollapsible, useTheme } from "@/lib/theme";
import { mainNavItems } from "@/nav-items";
import { type SharedProps } from "@/types";

export function AppSidebar() {
  const theme = useTheme(usePage<SharedProps>().props.theme);

  return (
    <Sidebar
      collapsible={sidebarCollapsible[theme.sidebarMode]}
      variant={theme.sidebarVariant}
    >
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" asChild>
              <Link href="/home">
                <AppLogo />
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        <NavMain items={mainNavItems} />
      </SidebarContent>

      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
    </Sidebar>
  );
}
