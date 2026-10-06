package app

import (
	"context"
	"encoding/json"
	"html/template"
	"strings"
	"time"

	"github.com/martin3zra/forge/session"
	"github.com/romsar/gonertia/v3"
)

// The theme customizer: per-user preset, font, color mode, radius, scale,
// content layout and sidebar settings, rendered by the server into every
// page's <html> and <head> so the first paint is already right (no flash).
//
// What is stored where:
//   - users.appearance: light | dark | system (kiosk apps already have it).
//   - users.theme_settings: everything else, as ThemeSettings JSON.
//
// How it reaches the page:
//   - root.html: data-appearance/radius/scale/layout attributes and the
//     .dark class on <html>, the preset's colors and the font in
//     {{ .themeStyle }}, and the font's stylesheet in {{ .themeFont }}.
//   - The "theme" shared prop (resolvedTheme), which resources/js/lib/theme.ts
//     re-applies on Inertia navigation and during live preview.
//
// Radius, scale and layout are plain attributes: resources/css/app.css owns
// their values. Colors and fonts come from the catalogs in
// theme_presets.go.

// ThemeSettings is the stored shape of users.theme_settings. An empty field
// means "use the default"; an empty Preset also lets a layer above the user
// (kiosk's company preset) pick one.
type ThemeSettings struct {
	Preset         string `json:"preset,omitempty"`
	Font           string `json:"font,omitempty"`
	Radius         string `json:"radius,omitempty"`
	Scale          string `json:"scale,omitempty"`
	Layout         string `json:"layout,omitempty"`
	SidebarVariant string `json:"sidebarVariant,omitempty"`
	SidebarMode    string `json:"sidebarMode,omitempty"`
	// V is when the settings were last saved (Unix ms). The client uses it
	// to ignore older copies of the theme prop, e.g. after a back
	// navigation restores a page from history.
	V int64 `json:"v,omitempty"`
}

// defaultThemeSettings is what a user who never opened the customizer sees.
// Each field must be one of the allowed values below. Preset is empty: no
// choice, so the default preset (or a company's, with kiosk) applies.
var defaultThemeSettings = ThemeSettings{
	Font:           "geist",
	Radius:         "md",
	Scale:          "md",
	Layout:         "compact",
	SidebarVariant: "inset",
	SidebarMode:    "icon",
}

const defaultAppearance = "system"

var (
	themeAppearances     = []string{"light", "dark", "system"}
	themeRadii           = []string{"none", "sm", "md", "lg"}
	themeScales          = []string{"sm", "md", "lg"}
	themeLayouts         = []string{"compact", "full"}
	themeSidebarVariants = []string{"floating", "inset"}
	themeSidebarModes    = []string{"default", "icon", "full"}
)

func oneOf(value string, allowed []string, fallback string) string {
	for _, a := range allowed {
		if value == a {
			return value
		}
	}
	return fallback
}

// normalize replaces any unknown or empty value with the default, so a bad
// stored value can never break a page. Preset stays empty when unset (see
// ThemeSettings).
func (t ThemeSettings) normalize() ThemeSettings {
	d := defaultThemeSettings
	if _, ok := themePresets[t.Preset]; !ok {
		t.Preset = ""
	}
	if _, ok := themeFonts[t.Font]; !ok {
		t.Font = d.Font
	}
	t.Radius = oneOf(t.Radius, themeRadii, d.Radius)
	t.Scale = oneOf(t.Scale, themeScales, d.Scale)
	t.Layout = oneOf(t.Layout, themeLayouts, d.Layout)
	t.SidebarVariant = oneOf(t.SidebarVariant, themeSidebarVariants, d.SidebarVariant)
	t.SidebarMode = oneOf(t.SidebarMode, themeSidebarModes, d.SidebarMode)
	return t
}

// normalizeAppearance falls back to "system" for an empty or unknown value.
func normalizeAppearance(appearance string) string {
	return oneOf(appearance, themeAppearances, defaultAppearance)
}

// parseThemeSettings decodes users.theme_settings. NULL, empty or invalid
// JSON gives the defaults.
func parseThemeSettings(raw *string) ThemeSettings {
	var t ThemeSettings
	if raw != nil && *raw != "" {
		_ = json.Unmarshal([]byte(*raw), &t)
	}
	return t.normalize()
}

// encode is the users.theme_settings column value for t.
func (t ThemeSettings) encode() string {
	b, _ := json.Marshal(t)
	return string(b)
}

// resolvedTheme is the "theme" shared prop and the input to the template
// data. Field names mirror ResolvedTheme in resources/js/lib/theme.ts.
type resolvedTheme struct {
	// Preset is the preset in effect; PresetChosen says whether the user
	// picked it (false: a default or a company preset).
	Preset       string      `json:"preset"`
	PresetChosen bool        `json:"presetChosen"`
	Light        themeColors `json:"light"`
	Dark         themeColors `json:"dark"`
	Appearance   string      `json:"appearance"`

	Font       string `json:"font"`
	FontFamily string `json:"fontFamily"`
	FontHref   string `json:"fontHref"`

	Radius         string `json:"radius"`
	Scale          string `json:"scale"`
	Layout         string `json:"layout"`
	SidebarVariant string `json:"sidebarVariant"`
	SidebarMode    string `json:"sidebarMode"`
	V              int64  `json:"v"`
}

// newResolvedTheme builds the theme for already-normalized settings.
func newResolvedTheme(settings ThemeSettings, appearance string) resolvedTheme {
	presetID := settings.Preset
	if presetID == "" {
		presetID = defaultThemePresetID
	}
	preset := themePresets[presetID]
	font := themeFonts[settings.Font]
	return resolvedTheme{
		Preset:         preset.ID,
		PresetChosen:   settings.Preset != "",
		Light:          preset.Light,
		Dark:           preset.Dark,
		Appearance:     normalizeAppearance(appearance),
		Font:           font.ID,
		FontFamily:     font.Family,
		FontHref:       font.href(),
		Radius:         settings.Radius,
		Scale:          settings.Scale,
		Layout:         settings.Layout,
		SidebarVariant: settings.SidebarVariant,
		SidebarMode:    settings.SidebarMode,
		V:              settings.V,
	}
}

// withPreset swaps the colors for another preset, keeping everything else.
func (t resolvedTheme) withPreset(id string) resolvedTheme {
	preset := themePresetByID(id)
	t.Preset, t.Light, t.Dark = preset.ID, preset.Light, preset.Dark
	return t
}

// currentTheme is the theme for this request: SharedProps renders it into
// every page, and updateThemeHandler returns it after a save. It's the one
// place to layer anything above the user's own choices (kiosk:install adds
// the company's preset here).
func (s *Server) currentTheme(ctx context.Context, sess *session.Session) resolvedTheme {
	return resolveTheme(sess)
}

// resolveTheme reads the signed-in user's settings from the session's cached
// user (refreshed on save, see updateThemeHandler), so it costs no query.
// Guests get the defaults with "system", so they follow the browser.
func resolveTheme(sess *session.Session) resolvedTheme {
	return themeForUser(sess.Get("user"))
}

// themeForUser is resolveTheme for the session's "user" value: a map once it
// has been through the session store, an *AuthUser in the request that just
// saved it (JSON reads both), or nil for a guest.
func themeForUser(raw any) resolvedTheme {
	var user struct {
		Appearance    string        `json:"appearance"`
		ThemeSettings ThemeSettings `json:"theme_settings"`
	}
	if raw == nil {
		return newResolvedTheme(defaultThemeSettings.normalize(), defaultAppearance)
	}
	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &user)
	return newResolvedTheme(user.ThemeSettings.normalize(), user.Appearance)
}

// templateData is what root.html reads: the <html> attributes (theme), the
// colors and font (themeStyle) and the font's stylesheet (themeFont).
// template.HTML is safe here because every value comes from the fixed
// catalogs in theme_presets.go, never from user input.
func (t resolvedTheme) templateData() gonertia.TemplateData {
	font := template.HTML("")
	if t.FontHref != "" {
		font = template.HTML(`<link id="theme-font" rel="stylesheet" href="` + t.FontHref + `" />`)
	}
	return gonertia.TemplateData{
		"theme":      t,
		"themeStyle": template.HTML(`<style id="theme-style">` + t.css() + `</style>`),
		"themeFont":  font,
	}
}

// css mirrors themeCSS in resources/js/lib/theme.ts. The .dark block follows
// :root, so it wins whenever <html> has the class (set by the server for
// "dark", and by root.html's script for "system").
func (t resolvedTheme) css() string {
	return ":root{" + cssVarBlock(t.Light) + "--app-font:" + t.FontFamily + ";}" +
		".dark{" + cssVarBlock(t.Dark) + "}"
}

// themeColors is one mode (light or dark) of a preset, one field per shadcn
// color variable in resources/css/app.css. JSON keys are the Go names (the
// kiosk Settings page reads them that way). An empty field is left out of
// the CSS, so app.css's value stays.
type themeColors struct {
	Background               string
	Foreground               string
	Card                     string
	CardForeground           string
	Popover                  string
	PopoverForeground        string
	Primary                  string
	PrimaryForeground        string
	Secondary                string
	SecondaryForeground      string
	Muted                    string
	MutedForeground          string
	Accent                   string
	AccentForeground         string
	Destructive              string
	DestructiveForeground    string
	Border                   string
	Input                    string
	Ring                     string
	Chart1                   string
	Chart2                   string
	Chart3                   string
	Chart4                   string
	Chart5                   string
	Sidebar                  string
	SidebarForeground        string
	SidebarPrimary           string
	SidebarPrimaryForeground string
	SidebarAccent            string
	SidebarAccentForeground  string
	SidebarBorder            string
	SidebarRing              string
}

// cssVarBlock renders c as --name:value; declarations in a fixed order.
func cssVarBlock(c themeColors) string {
	var b strings.Builder
	pairs := [...][2]string{
		{"--background", c.Background}, {"--foreground", c.Foreground},
		{"--card", c.Card}, {"--card-foreground", c.CardForeground},
		{"--popover", c.Popover}, {"--popover-foreground", c.PopoverForeground},
		{"--primary", c.Primary}, {"--primary-foreground", c.PrimaryForeground},
		{"--secondary", c.Secondary}, {"--secondary-foreground", c.SecondaryForeground},
		{"--muted", c.Muted}, {"--muted-foreground", c.MutedForeground},
		{"--accent", c.Accent}, {"--accent-foreground", c.AccentForeground},
		{"--destructive", c.Destructive}, {"--destructive-foreground", c.DestructiveForeground},
		{"--border", c.Border}, {"--input", c.Input}, {"--ring", c.Ring},
		{"--chart-1", c.Chart1}, {"--chart-2", c.Chart2}, {"--chart-3", c.Chart3},
		{"--chart-4", c.Chart4}, {"--chart-5", c.Chart5},
		{"--sidebar", c.Sidebar}, {"--sidebar-foreground", c.SidebarForeground},
		{"--sidebar-primary", c.SidebarPrimary}, {"--sidebar-primary-foreground", c.SidebarPrimaryForeground},
		{"--sidebar-accent", c.SidebarAccent}, {"--sidebar-accent-foreground", c.SidebarAccentForeground},
		{"--sidebar-border", c.SidebarBorder}, {"--sidebar-ring", c.SidebarRing},
	}
	for _, p := range pairs {
		if p[1] == "" {
			continue
		}
		b.WriteString(p[0])
		b.WriteByte(':')
		b.WriteString(p[1])
		b.WriteByte(';')
	}
	return b.String()
}

// themePreset is one selectable color palette.
type themePreset struct {
	ID    string
	Name  string
	Light themeColors
	Dark  themeColors
}

const defaultThemePresetID = "default"

// themePresetByID falls back to the default preset for an unknown id.
func themePresetByID(id string) themePreset {
	if p, ok := themePresets[id]; ok {
		return p
	}
	return themePresets[defaultThemePresetID]
}

// themePresetIDs is the catalog joined for an "in:" validator rule.
func themePresetIDs() string {
	return strings.Join(themePresetOrder, ",")
}

// themePresetSummary is one entry of the catalog sent to the customizer.
type themePresetSummary struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Light themeColors `json:"light"`
	Dark  themeColors `json:"dark"`
}

// themePresetCatalog returns every preset in display order.
func themePresetCatalog() []themePresetSummary {
	catalog := make([]themePresetSummary, 0, len(themePresetOrder))
	for _, id := range themePresetOrder {
		p := themePresets[id]
		catalog = append(catalog, themePresetSummary{ID: p.ID, Name: p.Name, Light: p.Light, Dark: p.Dark})
	}
	return catalog
}

// themeFont is one selectable font. Google is the Google Fonts family
// parameter; an empty Google means the font is self-hosted through an
// @font-face rule in app.css.
type themeFont struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Group  string `json:"group"`
	Family string `json:"family"`
	Google string `json:"-"`
}

// href is the stylesheet that loads the font, or "" when app.css has it.
func (f themeFont) href() string {
	if f.Google == "" {
		return ""
	}
	return "https://fonts.googleapis.com/css2?family=" + f.Google + ":wght@400;500;600;700&display=swap"
}

// themeFontSummary is one entry of the font catalog sent to the customizer.
type themeFontSummary struct {
	themeFont
	Href string `json:"href"`
}

// themeFontCatalog returns every font in display order.
func themeFontCatalog() []themeFontSummary {
	catalog := make([]themeFontSummary, 0, len(themeFontOrder))
	for _, id := range themeFontOrder {
		f := themeFonts[id]
		catalog = append(catalog, themeFontSummary{themeFont: f, Href: f.href()})
	}
	return catalog
}

// newThemeVersion stamps a save; see ThemeSettings.V.
func newThemeVersion() int64 {
	return time.Now().UnixMilli()
}
