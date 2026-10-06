import { usePage } from "@inertiajs/react";
import { Ban, Palette, RotateCcw } from "lucide-react";
import { useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useIsMobile } from "@/hooks/use-mobile";
import {
  applyTheme,
  useTheme,
  type ResolvedTheme,
  type ThemeCatalog,
} from "@/lib/theme";
import { type SharedProps } from "@/types";

// The catalog (every preset's colors, every font) is only needed once the
// panel opens, so it's fetched then, once per page load.
let catalogRequest: Promise<ThemeCatalog> | null = null;
function loadCatalog(): Promise<ThemeCatalog> {
  catalogRequest ??= fetch("/settings/theme/catalog", {
    headers: { Accept: "application/json" },
  }).then((r) => {
    if (!r.ok) throw new Error(`catalog: ${r.status}`);
    return r.json();
  });
  catalogRequest.catch(() => (catalogRequest = null));
  return catalogRequest;
}

// What PUT /settings/theme takes (app/theme-handlers.go).
function saveBody(t: ResolvedTheme) {
  return {
    appearance: t.appearance,
    preset: t.presetChosen ? t.preset : "",
    font: t.font,
    radius: t.radius,
    scale: t.scale,
    layout: t.layout,
    sidebarVariant: t.sidebarVariant,
    sidebarMode: t.sidebarMode,
  };
}

const fontGroups = [
  { group: "sans", label: "Sans" },
  { group: "mono", label: "Mono" },
  { group: "serif", label: "Serif" },
] as const;

interface ThemeCustomizerProps {
  // False in layouts without a sidebar (the top-nav layout), which hides the
  // two sidebar settings.
  sidebar?: boolean;
}

// ThemeCustomizer is the header's theme panel: every change shows at once
// and is saved to the user's account (app/theme-handlers.go).
export function ThemeCustomizer({ sidebar = true }: ThemeCustomizerProps) {
  const page = usePage<SharedProps>();
  const theme = useTheme(page.props.theme);
  const { csrf_token } = page.props;
  const isMobile = useIsMobile();
  const [catalog, setCatalog] = useState<ThemeCatalog | null>(null);

  // Saves are serialized by sequence number: only the newest request's
  // answer is applied, so a slow response can't undo a later click.
  const seq = useRef(0);

  function open(next: boolean) {
    if (next && !catalog) {
      loadCatalog()
        .then(setCatalog)
        .catch(() => {
          // The panel still works without the catalog for everything but
          // the preset and font pickers.
        });
    }
  }

  async function save(body: object) {
    const mine = ++seq.current;
    try {
      const res = await fetch("/settings/theme", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
          "X-CSRF-Token": csrf_token,
        },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(`save: ${res.status}`);
      const saved: ResolvedTheme = await res.json();
      if (mine === seq.current) applyTheme(saved);
    } catch {
      // Keep the preview; the next page load shows what was really saved.
    }
  }

  function change(patch: Partial<ResolvedTheme>) {
    const next = { ...theme, ...patch };
    applyTheme(next);
    void save(saveBody(next));
  }

  function choosePreset(id: string) {
    const preset = catalog?.presets.find((p) => p.id === id);
    if (!preset) return;
    change({
      preset: id,
      presetChosen: true,
      light: preset.light,
      dark: preset.dark,
    });
  }

  function chooseFont(id: string) {
    const font = catalog?.fonts.find((f) => f.id === id);
    if (!font) return;
    change({ font: id, fontFamily: font.family, fontHref: font.href });
  }

  // An empty body resets every setting to the app's defaults; the response
  // is the resulting theme.
  function reset() {
    void save({});
  }

  // Settings are saved to the account, so guests don't get the panel; they
  // follow the browser's light or dark mode.
  if (!page.props.auth?.user) return null;

  const presets = catalog?.presets ?? [];
  const fonts = catalog?.fonts ?? [];
  const isDark =
    typeof document !== "undefined" &&
    document.documentElement.classList.contains("dark");

  return (
    <Popover onOpenChange={open}>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="icon" className="size-8">
          <Palette />
          <span className="sr-only">Theme customizer</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 p-0">
        <div className="flex items-start justify-between gap-2 px-4 pt-4">
          <div>
            <h2 className="text-base font-semibold">Theme Customizer</h2>
            <p className="text-sm text-muted-foreground">
              Customize your theme to your liking.
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="size-7 shrink-0"
            onClick={reset}
            title="Reset to defaults"
          >
            <RotateCcw />
            <span className="sr-only">Reset to defaults</span>
          </Button>
        </div>

        <div className="flex max-h-[70vh] flex-col gap-4 overflow-y-auto p-4">
          <Field label="Theme Preset">
            <Select
              value={theme.preset}
              onValueChange={choosePreset}
              disabled={!catalog}
            >
              <SelectTrigger className="w-full">
                <SelectValue>
                  <Swatch
                    color={isDark ? theme.dark.Primary : theme.light.Primary}
                  />
                  {presets.find((p) => p.id === theme.preset)?.name ??
                    theme.preset}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {presets.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    <Swatch
                      color={isDark ? p.dark.Primary : p.light.Primary}
                    />
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>

          <Field label="Font">
            <Select
              value={theme.font}
              onValueChange={chooseFont}
              disabled={!catalog}
            >
              <SelectTrigger className="w-full">
                <SelectValue>
                  {fonts.find((f) => f.id === theme.font)?.name ?? theme.font}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {fontGroups.map(({ group, label }) => (
                  <SelectGroup key={group}>
                    <SelectLabel>{label}</SelectLabel>
                    {fonts
                      .filter((f) => f.group === group)
                      .map((f) => (
                        <SelectItem key={f.id} value={f.id}>
                          {f.name}
                        </SelectItem>
                      ))}
                  </SelectGroup>
                ))}
              </SelectContent>
            </Select>
          </Field>

          <Field label="Color Mode">
            <Segmented
              value={theme.appearance}
              onChange={(appearance) => change({ appearance })}
              options={[
                ["light", "Light"],
                ["dark", "Dark"],
                ["system", "System"],
              ]}
            />
          </Field>

          <Field label="Radius">
            <Segmented
              value={theme.radius}
              onChange={(radius) => change({ radius })}
              options={[
                ["none", <Ban key="none" aria-label="None" />],
                ["sm", "SM"],
                ["md", "MD"],
                ["lg", "LG"],
              ]}
            />
          </Field>

          <Field label="Content Layout">
            <Segmented
              value={theme.layout}
              onChange={(layout) => change({ layout })}
              options={[
                ["compact", "Compact"],
                ["full", "Full"],
              ]}
            />
          </Field>

          <Field label="Scale">
            <Segmented
              value={theme.scale}
              onChange={(scale) => change({ scale })}
              options={[
                ["sm", "SM"],
                ["md", "MD"],
                ["lg", "LG"],
              ]}
            />
          </Field>

          {/* On phones the sidebar is always a sheet, so these don't apply. */}
          {sidebar && !isMobile && (
            <>
              <Field label="Sidebar Variant">
                <Segmented
                  value={theme.sidebarVariant}
                  onChange={(sidebarVariant) => change({ sidebarVariant })}
                  options={[
                    ["floating", "Floating"],
                    ["inset", "Inset"],
                  ]}
                />
              </Field>

              <Field label="Sidebar Mode">
                <Segmented
                  value={theme.sidebarMode}
                  onChange={(sidebarMode) => change({ sidebarMode })}
                  options={[
                    ["default", "Default"],
                    ["icon", "Icon"],
                    ["full", "Full"],
                  ]}
                />
              </Field>
            </>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-sm font-medium">{label}</span>
      {children}
    </div>
  );
}

function Segmented<T extends string>({
  value,
  onChange,
  options,
}: {
  value: T;
  onChange: (value: T) => void;
  options: [T, React.ReactNode][];
}) {
  return (
    <ToggleGroup
      type="single"
      value={value}
      // Radix sends "" when the pressed item is clicked again; keep it.
      onValueChange={(v) => v && v !== value && onChange(v as T)}
    >
      {options.map(([v, label]) => (
        <ToggleGroupItem key={v} value={v}>
          {label}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  );
}

function Swatch({ color }: { color: string }) {
  return (
    <span
      className="size-3 shrink-0 rounded-full border"
      style={{ background: color }}
    />
  );
}
