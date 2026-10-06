package app

import (
	"bytes"
	"html/template"
	"os"
	"strings"
	"testing"
)

func TestThemeSettingsNormalize(t *testing.T) {
	got := ThemeSettings{Preset: "nope", Font: "comic-sans", Radius: "xl", Scale: "lg", Layout: "full", SidebarVariant: "x", SidebarMode: "icon"}.normalize()
	want := defaultThemeSettings
	want.Preset = ""
	want.Scale, want.Layout, want.SidebarMode = "lg", "full", "icon"
	if got != want {
		t.Fatalf("normalize = %+v, want %+v", got, want)
	}
	if got := (ThemeSettings{Preset: "nature"}).normalize().Preset; got != "nature" {
		t.Fatalf("a known preset was dropped: %q", got)
	}
}

func TestParseThemeSettings(t *testing.T) {
	bad := "{not json"
	if got := parseThemeSettings(&bad); got != defaultThemeSettings.normalize() {
		t.Fatalf("invalid JSON should give the defaults, got %+v", got)
	}
	if got := parseThemeSettings(nil); got != defaultThemeSettings.normalize() {
		t.Fatalf("NULL should give the defaults, got %+v", got)
	}
	saved := ThemeSettings{Preset: "slack", Font: "inter", Radius: "none", Scale: "sm", Layout: "full", SidebarVariant: "floating", SidebarMode: "full", V: 42}.encode()
	got := parseThemeSettings(&saved)
	if got.Preset != "slack" || got.Font != "inter" || got.Radius != "none" || got.V != 42 {
		t.Fatalf("round trip lost values: %+v", got)
	}
}

func TestThemeForUser(t *testing.T) {
	guest := themeForUser(nil)
	if guest.Appearance != "system" || guest.Preset != defaultThemePresetID || guest.PresetChosen {
		t.Fatalf("guest theme = %+v", guest)
	}

	// The session store hands the user back as a map...
	fromStore := themeForUser(map[string]any{
		"appearance":     "dark",
		"theme_settings": map[string]any{"preset": "nature", "font": "lora", "radius": "lg"},
	})
	if fromStore.Appearance != "dark" || fromStore.Preset != "nature" || !fromStore.PresetChosen ||
		fromStore.Font != "lora" || fromStore.Radius != "lg" || fromStore.Light != themePresets["nature"].Light {
		t.Fatalf("theme from stored user = %+v", fromStore)
	}

	// ...and as the *AuthUser updateThemeHandler just put there.
	justSaved := themeForUser(&AuthUser{Appearance: "light", ThemeSettings: ThemeSettings{Preset: "claude"}})
	if justSaved.Appearance != "light" || justSaved.Preset != "claude" {
		t.Fatalf("theme from just-saved user = %+v", justSaved)
	}
}

func TestThemeCSS(t *testing.T) {
	theme := newResolvedTheme(ThemeSettings{Preset: "caffeine", Font: "inter"}.normalize(), "dark")
	css := theme.css()
	if !strings.HasPrefix(css, ":root{--background:"+themePresets["caffeine"].Light.Background+";") {
		t.Fatalf("css should start with the light colors: %s", css[:80])
	}
	if !strings.Contains(css, ".dark{--background:"+themePresets["caffeine"].Dark.Background+";") {
		t.Fatal("css has no .dark block with the dark colors")
	}
	if !strings.Contains(css, `--app-font:"Inter", ui-sans-serif`) {
		t.Fatal("css doesn't set the font")
	}
	// Presets that don't define a color leave app.css's value alone.
	if strings.Contains(css, "--destructive-foreground:;") {
		t.Fatal("an empty color was written")
	}
}

func TestThemeFonts(t *testing.T) {
	if got := themeFonts["geist-pixel"].href(); got != "" {
		t.Fatalf("Geist Pixel is self-hosted, got href %q", got)
	}
	if got := themeFonts["dm-sans"].href(); got != "https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700&display=swap" {
		t.Fatalf("dm-sans href = %q", got)
	}
	if len(themeFontOrder) != len(themeFonts) || len(themePresetOrder) != len(themePresets) {
		t.Fatal("a catalog's display order doesn't list every entry")
	}
}

// root.html must render with the theme, and without it (a render that
// skipped SharedProps), where it falls back to "system".
func TestRootTemplateTheme(t *testing.T) {
	src, err := os.ReadFile("../resources/views/root.html")
	if err != nil {
		t.Fatal(err)
	}
	tmpl := template.Must(template.New("root").Funcs(template.FuncMap{
		"vite": func(string) (string, error) { return "/build/x", nil },
	}).Parse(string(src)))

	render := func(data map[string]any) string {
		var out bytes.Buffer
		if err := tmpl.Execute(&out, data); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}

	theme := newResolvedTheme(ThemeSettings{Font: "inter", Radius: "lg", Scale: "sm", Layout: "full"}.normalize(), "dark")
	data := map[string]any(theme.templateData())
	html := render(data)
	for _, want := range []string{
		`class="dark"`, `data-appearance="dark"`, `data-radius="lg"`, `data-scale="sm"`, `data-layout="full"`,
		`<style id="theme-style">:root{`, `<link id="theme-font" rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" />`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered root.html is missing %s", want)
		}
	}

	bare := render(map[string]any{})
	if !strings.Contains(bare, `data-appearance="system"`) {
		t.Error("root.html without a theme should fall back to system")
	}
}
