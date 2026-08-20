import { Link, usePage } from "@inertiajs/react";
import { ChevronDown, ChevronsUpDown } from "lucide-react";

import AppLogo from "@/components/app-logo";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { UserInfo } from "@/components/user-info";
import { UserMenuContent } from "@/components/user-menu-content";
import { mainNavItems } from "@/nav-items";
import { cn } from "@/lib/utils";
import { type NavItem, type SharedProps } from "@/types";

function TopNavItem({ item, active }: { item: NavItem; active: boolean }) {
  if (item.children?.length) {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            className={cn(
              "gap-2",
              active && "bg-accent text-accent-foreground",
            )}
          >
            {item.icon && <item.icon className="size-4" />}
            {item.title}
            <ChevronDown className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          {item.children.map((child) => (
            <DropdownMenuItem key={child.title} asChild>
              <Link href={child.href}>{child.title}</Link>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    );
  }

  return (
    <Button
      variant="ghost"
      asChild
      className={cn("gap-2", active && "bg-accent text-accent-foreground")}
    >
      <Link href={item.href}>
        {item.icon && <item.icon className="size-4" />}
        {item.title}
      </Link>
    </Button>
  );
}

export function AppTopNav() {
  const page = usePage<SharedProps>();
  const { auth } = page.props;

  return (
    <header className="border-b">
      <div className="mx-auto flex h-16 max-w-7xl items-center gap-6 px-6">
        <Link href="/home" className="flex items-center">
          <AppLogo />
        </Link>

        <nav className="flex items-center gap-1">
          {mainNavItems.map((item) => (
            <TopNavItem
              key={item.title}
              item={item}
              active={page.url.startsWith(item.href)}
            />
          ))}
        </nav>

        {auth?.user && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="ml-auto gap-2 px-2">
                <UserInfo user={auth.user} />
                <ChevronsUpDown className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56 rounded-lg">
              <UserMenuContent user={auth.user} />
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </header>
  );
}
