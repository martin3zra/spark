import { useSyncExternalStore } from "react";

// The client half of the theme customizer (app/theme.go). The first paint
// never needs this: the server renders the theme into <html> and <head>.
// It's for live preview in the customizer and for re-applying the theme
// after an Inertia navigation.

// Field names are the Go struct's (app/theme.go, themeColors).
export interface ThemeColors {
  Background: string;
  Foreground: string;
  Card: string;
  CardForeground: string;
  Popover: string;
  PopoverForeground: string;
  Primary: string;
  PrimaryForeground: string;
  Secondary: string;
  SecondaryForeground: string;
  Muted: string;
  MutedForeground: string;
  Accent: string;
  AccentForeground: string;
  Destructive: string;
  DestructiveForeground: string;
  Border: string;
  Input: string;
  Ring: string;
  Chart1: string;
  Chart2: string;
  Chart3: string;
  Chart4: string;
  Chart5: string;
  Sidebar: string;
  SidebarForeground: string;
  SidebarPrimary: string;
  SidebarPrimaryForeground: string;
  SidebarAccent: string;
  SidebarAccentForeground: string;
  SidebarBorder: string;
  SidebarRing: string;
}

export type Appearance = "light" | "dark" | "system";
export type Radius = "none" | "sm" | "md" | "lg";
export type Scale = "sm" | "md" | "lg";
export type ContentLayout = "compact" | "full";
export type SidebarVariant = "floating" | "inset";
export type SidebarMode = "default" | "icon" | "full";

// A sidebar mode as the shadcn sidebar's collapsible prop.
export const sidebarCollapsible = {
  default: "none",
  icon: "icon",
  full: "offcanvas",
} as const satisfies Record<SidebarMode, "none" | "icon" | "offcanvas">;

// The "theme" shared prop (app/theme.go, resolvedTheme).
export interface ResolvedTheme {
  preset: string;
  presetChosen: boolean;
  light: ThemeColors;
  dark: ThemeColors;
  appearance: Appearance;
  font: string;
  fontFamily: string;
  fontHref: string;
  radius: Radius;
  scale: Scale;
  layout: ContentLayout;
  sidebarVariant: SidebarVariant;
  sidebarMode: SidebarMode;
  v: number;
}

export interface ThemePresetSummary {
  id: string;
  name: string;
  light: ThemeColors;
  dark: ThemeColors;
}

export interface ThemeFont {
  id: string;
  name: string;
  group: "sans" | "mono" | "serif";
  family: string;
  href: string;
}

export interface ThemeCatalog {
  presets: ThemePresetSummary[];
  fonts: ThemeFont[];
}

const colorVars: [keyof ThemeColors, string][] = [
  ["Background", "--background"],
  ["Foreground", "--foreground"],
  ["Card", "--card"],
  ["CardForeground", "--card-foreground"],
  ["Popover", "--popover"],
  ["PopoverForeground", "--popover-foreground"],
  ["Primary", "--primary"],
  ["PrimaryForeground", "--primary-foreground"],
  ["Secondary", "--secondary"],
  ["SecondaryForeground", "--secondary-foreground"],
  ["Muted", "--muted"],
  ["MutedForeground", "--muted-foreground"],
  ["Accent", "--accent"],
  ["AccentForeground", "--accent-foreground"],
  ["Destructive", "--destructive"],
  ["DestructiveForeground", "--destructive-foreground"],
  ["Border", "--border"],
  ["Input", "--input"],
  ["Ring", "--ring"],
  ["Chart1", "--chart-1"],
  ["Chart2", "--chart-2"],
  ["Chart3", "--chart-3"],
  ["Chart4", "--chart-4"],
  ["Chart5", "--chart-5"],
  ["Sidebar", "--sidebar"],
  ["SidebarForeground", "--sidebar-foreground"],
  ["SidebarPrimary", "--sidebar-primary"],
  ["SidebarPrimaryForeground", "--sidebar-primary-foreground"],
  ["SidebarAccent", "--sidebar-accent"],
  ["SidebarAccentForeground", "--sidebar-accent-foreground"],
  ["SidebarBorder", "--sidebar-border"],
  ["SidebarRing", "--sidebar-ring"],
];

function cssVarBlock(c: ThemeColors): string {
  return colorVars
    .filter(([key]) => c[key])
    .map(([key, name]) => `${name}:${c[key]};`)
    .join("");
}

// Mirrors resolvedTheme.css() in app/theme.go.
function themeCSS(theme: ResolvedTheme): string {
  return (
    `:root{${cssVarBlock(theme.light)}--app-font:${theme.fontFamily};}` +
    `.dark{${cssVarBlock(theme.dark)}}`
  );
}

function prefersDark(): boolean {
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

function applyToDocument(theme: ResolvedTheme) {
  const root = document.documentElement;
  root.dataset.appearance = theme.appearance;
  root.dataset.radius = theme.radius;
  root.dataset.scale = theme.scale;
  root.dataset.layout = theme.layout;
  root.classList.toggle(
    "dark",
    theme.appearance === "dark" ||
      (theme.appearance === "system" && prefersDark()),
  );

  const style = document.getElementById("theme-style");
  const css = themeCSS(theme);
  if (style && style.textContent !== css) style.textContent = css;

  let link = document.getElementById("theme-font") as HTMLLinkElement | null;
  if (!theme.fontHref) {
    link?.remove();
  } else if (link?.getAttribute("href") !== theme.fontHref) {
    if (!link) {
      link = document.createElement("link");
      link.id = "theme-font";
      link.rel = "stylesheet";
      document.head.prepend(link);
    }
    link.href = theme.fontHref;
  }
}

// The theme in effect on this page. null until the first applyTheme; the
// hook falls back to the page's "theme" prop until then.
let current: ResolvedTheme | null = null;
const listeners = new Set<() => void>();

// applyTheme updates the page to theme. A partial theme (as kiosk's Settings
// page sends: preset colors and appearance only) is merged into the theme in
// effect.
export function applyTheme(
  theme: Partial<ResolvedTheme>,
  base?: ResolvedTheme,
) {
  const from = current ?? base;
  const next = { ...from, ...theme } as ResolvedTheme;
  if (!next.light || !next.dark) return;
  current = next;
  applyToDocument(next);
  listeners.forEach((listener) => listener());
}

// syncThemeFromServer applies the theme prop of a page the user navigated
// to, unless it's older than what's on screen: a page restored from history
// can carry a theme from before the last save.
export function syncThemeFromServer(theme: ResolvedTheme | undefined) {
  if (!theme) return;
  if (current && (theme.v ?? 0) < (current.v ?? 0)) return;
  current = null;
  applyTheme(theme);
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

// useTheme returns the theme in effect, including unsaved live-preview
// changes, so components like the sidebar follow the customizer at once.
export function useTheme(fromProps: ResolvedTheme): ResolvedTheme {
  return (
    useSyncExternalStore(
      subscribe,
      () => current,
      () => null,
    ) ?? fromProps
  );
}
