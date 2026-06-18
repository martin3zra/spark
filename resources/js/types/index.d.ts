import type { LucideIcon } from "lucide-react";

export interface User {
  id: number;
  uuid: string;
  name: string;
  email: string;
  role: string;
  email_verified_at: string | null;
  must_change_password: boolean;
  created_at: string;
  updated_at: string;
}

export interface Auth {
  user: User | null;
}

// Props shared with every page by the server's SharedProps middleware.
export interface SharedProps {
  auth: Auth;
  csrf_token: string;
  errors: Record<string, string[]> | null;
  flash: Record<string, unknown> | null;
  translations: Record<string, string>;
  [key: string]: unknown;
}

export interface NavItem {
  title: string;
  href: string;
  icon?: LucideIcon;
}

export interface BreadcrumbItem {
  title: string;
  href: string;
}
