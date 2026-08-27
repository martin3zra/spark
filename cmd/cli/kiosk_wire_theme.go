package main

import (
	"os"
	"path/filepath"
	"strings"
)

// themeEngineGo is app/theme.go — a new file, so it's a plain write rather
// than a patch. The whole per-tenant color theme engine: preset catalog,
// no-flash server-rendered <style> renderer, and company/appearance
// resolution. See the README's "Tenant theme" section for the source of
// truth this mirrors.
const themeEngineGo = `package app

import (
	"context"
	"strings"

	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/playsql"
)

// themeColors is one mode (light or dark) of a theme preset — one field per
// shadcn semantic CSS variable resources/css/app.css already defines.
// Deliberately excludes typography/radius/chart tokens: v1 only ever varies
// color.
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
	Sidebar                  string
	SidebarForeground        string
	SidebarPrimary           string
	SidebarPrimaryForeground string
	SidebarAccent            string
	SidebarAccentForeground  string
	SidebarBorder            string
	SidebarRing              string
}

// themePreset is one selectable brand identity — a company picks one via
// companies.theme_preset (see themePresetByID).
type themePreset struct {
	ID    string
	Name  string
	Light themeColors
	Dark  themeColors
}

const defaultThemePresetID = "default"

// defaultLightColors/defaultDarkColors are copied byte-for-byte from
// resources/css/app.css's existing :root/.dark blocks — a company that
// never sets a preset renders pixel-identical to before this feature
// existed. The placeholder presets below (blue/green/violet/rose) start
// from these same neutrals and only override the brand-carrying tokens —
// see withPrimary.
var defaultLightColors = themeColors{
	Background: "oklch(1 0 0)", Foreground: "oklch(0.145 0 0)",
	Card: "oklch(1 0 0)", CardForeground: "oklch(0.145 0 0)",
	Popover: "oklch(1 0 0)", PopoverForeground: "oklch(0.145 0 0)",
	Primary: "oklch(0.205 0 0)", PrimaryForeground: "oklch(0.985 0 0)",
	Secondary: "oklch(0.97 0 0)", SecondaryForeground: "oklch(0.205 0 0)",
	Muted: "oklch(0.97 0 0)", MutedForeground: "oklch(0.556 0 0)",
	Accent: "oklch(0.97 0 0)", AccentForeground: "oklch(0.205 0 0)",
	Destructive: "oklch(0.577 0.245 27.325)", DestructiveForeground: "oklch(0.985 0 0)",
	Border: "oklch(0.922 0 0)", Input: "oklch(0.922 0 0)", Ring: "oklch(0.708 0 0)",
	Sidebar: "oklch(0.985 0 0)", SidebarForeground: "oklch(0.145 0 0)",
	SidebarPrimary: "oklch(0.205 0 0)", SidebarPrimaryForeground: "oklch(0.985 0 0)",
	SidebarAccent: "oklch(0.97 0 0)", SidebarAccentForeground: "oklch(0.205 0 0)",
	SidebarBorder: "oklch(0.922 0 0)", SidebarRing: "oklch(0.708 0 0)",
}

var defaultDarkColors = themeColors{
	Background: "oklch(0.145 0 0)", Foreground: "oklch(0.985 0 0)",
	Card: "oklch(0.205 0 0)", CardForeground: "oklch(0.985 0 0)",
	Popover: "oklch(0.205 0 0)", PopoverForeground: "oklch(0.985 0 0)",
	Primary: "oklch(0.922 0 0)", PrimaryForeground: "oklch(0.205 0 0)",
	Secondary: "oklch(0.269 0 0)", SecondaryForeground: "oklch(0.985 0 0)",
	Muted: "oklch(0.269 0 0)", MutedForeground: "oklch(0.708 0 0)",
	Accent: "oklch(0.269 0 0)", AccentForeground: "oklch(0.985 0 0)",
	Destructive: "oklch(0.704 0.191 22.216)", DestructiveForeground: "oklch(0.985 0 0)",
	Border: "oklch(1 0 0 / 10%)", Input: "oklch(1 0 0 / 15%)", Ring: "oklch(0.556 0 0)",
	Sidebar: "oklch(0.205 0 0)", SidebarForeground: "oklch(0.985 0 0)",
	SidebarPrimary: "oklch(0.488 0.243 264.376)", SidebarPrimaryForeground: "oklch(0.985 0 0)",
	SidebarAccent: "oklch(0.269 0 0)", SidebarAccentForeground: "oklch(0.985 0 0)",
	SidebarBorder: "oklch(1 0 0 / 10%)", SidebarRing: "oklch(0.556 0 0)",
}

// withPrimary copies a base palette and overrides just the brand-carrying
// tokens — the placeholder presets vary only these, leaving
// backgrounds/borders/destructive neutral. Real reference presets will
// likely replace more than these fields; this is a v1 placeholder shape,
// not a permanent constraint.
func withPrimary(base themeColors, primary, primaryForeground, ring string) themeColors {
	base.Primary = primary
	base.PrimaryForeground = primaryForeground
	base.Ring = ring
	base.SidebarPrimary = primary
	base.SidebarPrimaryForeground = primaryForeground
	base.SidebarRing = ring
	return base
}

// themePresets is the whole v1 catalog. blue/green/violet/rose are
// placeholders with approximate, representative OKLCH hues — not copied
// from any specific reference theme.
var themePresets = map[string]themePreset{
	defaultThemePresetID: {
		ID: defaultThemePresetID, Name: "Default",
		Light: defaultLightColors, Dark: defaultDarkColors,
	},
	"blue": {
		ID: "blue", Name: "Blue",
		Light: withPrimary(defaultLightColors, "oklch(0.546 0.215 262.88)", "oklch(0.985 0 0)", "oklch(0.623 0.188 259.8)"),
		Dark:  withPrimary(defaultDarkColors, "oklch(0.623 0.188 259.8)", "oklch(0.145 0 0)", "oklch(0.546 0.215 262.88)"),
	},
	"green": {
		ID: "green", Name: "Green",
		Light: withPrimary(defaultLightColors, "oklch(0.554 0.171 142.5)", "oklch(0.985 0 0)", "oklch(0.696 0.17 162.48)"),
		Dark:  withPrimary(defaultDarkColors, "oklch(0.696 0.17 162.48)", "oklch(0.145 0 0)", "oklch(0.554 0.171 142.5)"),
	},
	"violet": {
		ID: "violet", Name: "Violet",
		Light: withPrimary(defaultLightColors, "oklch(0.541 0.221 292.72)", "oklch(0.985 0 0)", "oklch(0.606 0.25 292.72)"),
		Dark:  withPrimary(defaultDarkColors, "oklch(0.606 0.25 292.72)", "oklch(0.145 0 0)", "oklch(0.541 0.221 292.72)"),
	},
	"rose": {
		ID: "rose", Name: "Rose",
		Light: withPrimary(defaultLightColors, "oklch(0.586 0.221 17.58)", "oklch(0.985 0 0)", "oklch(0.645 0.246 16.44)"),
		Dark:  withPrimary(defaultDarkColors, "oklch(0.645 0.246 16.44)", "oklch(0.145 0 0)", "oklch(0.586 0.221 17.58)"),
	},
}

// themePresetOrder is the fixed display order for the Settings picker — Go
// map iteration order is randomized, and "which preset comes first"
// shouldn't shuffle between requests.
var themePresetOrder = []string{defaultThemePresetID, "blue", "green", "violet", "rose"}

// themePresetByID falls back to "default" for an unknown or empty id —
// resolution must never fail, since it runs on every request (see
// SharedProps).
func themePresetByID(id string) themePreset {
	if p, ok := themePresets[id]; ok {
		return p
	}
	return themePresets[defaultThemePresetID]
}

// themePresetIDs is themePresetOrder joined for the "in:" validator rule —
// see UpdateCompanySettingsForm.Rules() (forms.go). Only a submitted id
// that's actually in the catalog is accepted; never trust an arbitrary
// tenant-supplied string here.
func themePresetIDs() string {
	return strings.Join(themePresetOrder, ",")
}

// themePresetSummary is the JSON shape settingsPage sends down for the
// Appearance tab's picker — the whole catalog (id/name/light/dark), so the
// picker can render a live swatch per preset without a round trip per
// click. No custom json tags on Light/Dark: themeColors fields already
// serialize as their raw (capitalized) Go names.
type themePresetSummary struct {
	ID    string      ` + "`json:\"id\"`" + `
	Name  string      ` + "`json:\"name\"`" + `
	Light themeColors ` + "`json:\"light\"`" + `
	Dark  themeColors ` + "`json:\"dark\"`" + `
}

// themePresetCatalog returns every preset in themePresetOrder's fixed
// display order.
func themePresetCatalog() []themePresetSummary {
	catalog := make([]themePresetSummary, 0, len(themePresetOrder))
	for _, id := range themePresetOrder {
		p := themePresets[id]
		catalog = append(catalog, themePresetSummary{ID: p.ID, Name: p.Name, Light: p.Light, Dark: p.Dark})
	}
	return catalog
}

// companyThemeRow is a narrow, read-only shadow of companies used only for
// the theme lookup that runs on every request (SharedProps) — before any
// tenant-scoping middleware has run, so this queries by id directly.
type companyThemeRow struct {
	ID          int    ` + "`db:\"id\"`" + `
	ThemePreset string ` + "`db:\"theme_preset\"`" + `
}

func (companyThemeRow) TableName() string { return "companies" }

// companyIDFromSession reads session["company_id"] (set at login by
// resolveCompanyID, app/auth-handlers.go), tolerating both shapes it
// arrives in: an int when Put in the same request that set it, a float64
// once it's round-tripped through the database session store's JSON
// encoding.
func companyIDFromSession(sess *session.Session) (int, bool) {
	switch v := sess.Get("company_id").(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// resolveCompanyThemePreset looks up a company's chosen preset id, falling
// back to "default" on any lookup failure (missing row, no DB bound, etc.)
// — theme resolution must never fail a page load.
func resolveCompanyThemePreset(ctx context.Context, db *playsql.DB, companyID int) string {
	var row companyThemeRow
	if err := db.Model(&companyThemeRow{}).WhereEq("id", companyID).First(ctx, &row); err != nil {
		return defaultThemePresetID
	}
	return row.ThemePreset
}

const validAppearanceDefault = "system"

// normalizeAppearance guards against an empty/corrupt stored value the same
// way themePresetByID guards preset ids — resolution must never fail.
func normalizeAppearance(appearance string) string {
	switch appearance {
	case "light", "dark":
		return appearance
	default:
		return validAppearanceDefault
	}
}

// userAppearanceFromSession reads the raw session "user" payload directly —
// same pattern isDeveloperSession (middleware.go) uses — rather than
// through AuthUserFromContext/auth.ContextUserID{}, since SharedProps runs
// before AuthenticatedMiddleware seeds that context value.
func userAppearanceFromSession(sess *session.Session) string {
	raw, ok := sess.Get("user").(map[string]any)
	if !ok {
		return validAppearanceDefault
	}
	appearance, _ := raw["appearance"].(string)
	return normalizeAppearance(appearance)
}

// cssVarBlock renders one themeColors as ` + "`--name:value;`" + ` declarations, in
// a fixed field order — used by renderThemeStyle to build both the :root
// and the @media (prefers-color-scheme: dark) blocks.
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
		{"--sidebar", c.Sidebar}, {"--sidebar-foreground", c.SidebarForeground},
		{"--sidebar-primary", c.SidebarPrimary}, {"--sidebar-primary-foreground", c.SidebarPrimaryForeground},
		{"--sidebar-accent", c.SidebarAccent}, {"--sidebar-accent-foreground", c.SidebarAccentForeground},
		{"--sidebar-border", c.SidebarBorder}, {"--sidebar-ring", c.SidebarRing},
	}
	for _, p := range pairs {
		b.WriteString(p[0])
		b.WriteByte(':')
		b.WriteString(p[1])
		b.WriteByte(';')
	}
	return b.String()
}

// resolveTheme is SharedProps' single entry point into this file: given the
// current session, it resolves which company (if any) and which user (if
// any) are active, and returns the preset + appearance to render.
//
// applyCompanyTheme exists because company_id alone isn't a reliable "is
// this a tenant page" signal: a kiosk-allowlisted developer who is also a
// real member of some company still carries that company_id in session
// while browsing Kiosk's own cross-tenant pages, and Kiosk must never
// visually inherit one tenant's brand. SharedProps passes false when the
// session is a developer session that is not currently impersonating
// (kiosk.ImpersonateSessionKey unset) — impersonation is the one case
// where "acting as this company" genuinely means seeing its theme.
func resolveTheme(ctx context.Context, db *playsql.DB, sess *session.Session, applyCompanyTheme bool) (themePreset, string) {
	preset := themePresets[defaultThemePresetID]
	if companyID, ok := companyIDFromSession(sess); ok && applyCompanyTheme {
		preset = themePresetByID(resolveCompanyThemePreset(ctx, db, companyID))
	}
	return preset, userAppearanceFromSession(sess)
}

// renderThemeStyle builds the inline <style id="tenant-theme"> block
// SharedProps injects into every response's <head> (via gonertia's
// TemplateData, the same mechanism appName already uses — see root.html).
// This is the whole no-flash mechanism: Go already knows the resolved
// theme per request, so it renders the final CSS directly rather than
// shipping a client bootstrap script that would only run after first
// paint. "system" appearance resolves via a plain CSS media query — no
// matchMedia JS, no flash either way.
func renderThemeStyle(preset themePreset, appearance string) string {
	switch normalizeAppearance(appearance) {
	case "dark":
		return "<style id=\"tenant-theme\">:root{" + cssVarBlock(preset.Dark) + "}</style>"
	case "light":
		return "<style id=\"tenant-theme\">:root{" + cssVarBlock(preset.Light) + "}</style>"
	default: // "system"
		return "<style id=\"tenant-theme\">:root{" + cssVarBlock(preset.Light) +
			"}@media (prefers-color-scheme: dark){:root{" + cssVarBlock(preset.Dark) + "}}</style>"
	}
}
`

// wireKioskThemeEngine writes app/theme.go if it isn't there yet.
// Idempotent like wireKioskImpersonationBanner.
func wireKioskThemeEngine(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "theme.go")
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.WriteFile(path, []byte(themeEngineGo), 0o644); err != nil {
		return false, err
	}
	return true, gofmtFile(path)
}

// wireKioskThemeMiddleware adds theme resolution to app/middleware.go's
// SharedProps — the theme prop and the server-rendered themeStyle
// TemplateData (see root.html). Anchors match middleware.go's shape after
// wireKioskMiddleware has already run (isDeveloper/isImpersonating already
// present), since this step always runs after it in kioskInstallCmd.Run.
func wireKioskThemeMiddleware(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "middleware.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "resolveTheme(") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `import (
	"context"
	"net/http"
	"strings"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/database"
	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/kiosk"
	"github.com/romsar/gonertia/v3"
)`,
			replacement: `import (
	"context"
	"html/template"
	"net/http"
	"strings"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/database"
	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/kiosk"
	"github.com/romsar/gonertia/v3"
)`,
		},
		{
			anchor: `	return func(ctx *routing.Context) {
		sess := ctx.Request.Context().Value(session.SessionContextKey{}).(*session.Session)

		props := map[string]any{
			"appName": s.config.appName,
			"auth": map[string]any{
				"user":            sess.Get("user"),
				"isDeveloper":     isDeveloperSession(sess, s.config.kiosk.Developers),
				"isImpersonating": sess.Get(kiosk.ImpersonateSessionKey) != nil,
			},
			"csrf_token":   sess.Get("csrf_token"),
			"errors":       sess.Get("errors"),
			"flash":        sess.Get("flash"),
			"translations": s.translator.Translations,
		}

		s.sessionManager.AgeFlash(sess)

		ctxWithProps := gonertia.SetProps(ctx.Request.Context(), props)
		ctxWithProps = gonertia.SetTemplateData(ctxWithProps, gonertia.TemplateData{"appName": s.config.appName})
		next(ctx.WithContext(ctxWithProps))
	}
}`,
			replacement: `	return func(ctx *routing.Context) {
		sess := ctx.Request.Context().Value(session.SessionContextKey{}).(*session.Session)

		isDeveloper := isDeveloperSession(sess, s.config.kiosk.Developers)
		isImpersonating := sess.Get(kiosk.ImpersonateSessionKey) != nil
		preset, appearance := resolveTheme(ctx.Request.Context(), s.play, sess, !isDeveloper || isImpersonating)

		props := map[string]any{
			"appName": s.config.appName,
			"auth": map[string]any{
				"user":            sess.Get("user"),
				"isDeveloper":     isDeveloper,
				"isImpersonating": isImpersonating,
			},
			"csrf_token":   sess.Get("csrf_token"),
			"errors":       sess.Get("errors"),
			"flash":        sess.Get("flash"),
			"translations": s.translator.Translations,
			"theme": map[string]any{
				"preset":     preset.ID,
				"light":      preset.Light,
				"dark":       preset.Dark,
				"appearance": appearance,
			},
		}

		s.sessionManager.AgeFlash(sess)

		ctxWithProps := gonertia.SetProps(ctx.Request.Context(), props)
		ctxWithProps = gonertia.SetTemplateData(ctxWithProps, gonertia.TemplateData{
			"appName": s.config.appName,
			// template.HTML, not a plain string: root.html renders via
			// html/template (gonertia's own choice), which auto-escapes a
			// bare string. Safe here since renderThemeStyle's output is
			// built entirely from the fixed preset registry (theme.go).
			"themeStyle": template.HTML(renderThemeStyle(preset, appearance)),
		})
		next(ctx.WithContext(ctxWithProps))
	}
}`,
		},
	}

	patched, err := applyPatches(path, string(data), patches)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		return false, err
	}
	if err := gofmtFile(path); err != nil {
		return false, err
	}
	return true, nil
}

// wireKioskAuthUserAppearance adds the Appearance field to AuthUser/
// userModel/toAuthUser in app/auth-user.go — a personal light/dark/system
// preference, independent of the company's theme_preset.
func wireKioskAuthUserAppearance(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "auth-user.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "Appearance") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `	EmailVerifiedAt    *time.Time ` + "`json:\"email_verified_at\"`" + `
	MustChangePassword bool       ` + "`json:\"must_change_password\"`" + `
	foundation.Timestamps
}`,
			replacement: `	EmailVerifiedAt    *time.Time ` + "`json:\"email_verified_at\"`" + `
	MustChangePassword bool       ` + "`json:\"must_change_password\"`" + `
	// light | dark | system — a personal preference, independent of
	// whichever company's theme_preset is active. See app/theme.go.
	Appearance string ` + "`json:\"appearance\"`" + `
	foundation.Timestamps
}`,
		},
		{
			anchor: `	EmailVerifiedAt    *time.Time ` + "`db:\"email_verified_at\"`" + `
	MustChangePassword bool       ` + "`db:\"must_change_password\"`" + `
	CreatedAt          *time.Time ` + "`db:\"created_at\"`" + `
	UpdatedAt          *time.Time ` + "`db:\"updated_at\"`" + `
	DeletedAt          *time.Time ` + "`db:\"deleted_at\"`" + `
}`,
			replacement: `	EmailVerifiedAt    *time.Time ` + "`db:\"email_verified_at\"`" + `
	MustChangePassword bool       ` + "`db:\"must_change_password\"`" + `
	Appearance         string     ` + "`db:\"appearance\"`" + `
	CreatedAt          *time.Time ` + "`db:\"created_at\"`" + `
	UpdatedAt          *time.Time ` + "`db:\"updated_at\"`" + `
	DeletedAt          *time.Time ` + "`db:\"deleted_at\"`" + `
}`,
		},
		{
			anchor: `		EmailVerifiedAt:    m.EmailVerifiedAt,
		MustChangePassword: m.MustChangePassword,
		Timestamps: foundation.Timestamps{`,
			replacement: `		EmailVerifiedAt:    m.EmailVerifiedAt,
		MustChangePassword: m.MustChangePassword,
		Appearance:         m.Appearance,
		Timestamps: foundation.Timestamps{`,
		},
	}

	patched, err := applyPatches(path, string(data), patches)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		return false, err
	}
	if err := gofmtFile(path); err != nil {
		return false, err
	}
	return true, nil
}

// wireKioskRootHTMLTheme adds the server-rendered themeStyle block to
// resources/views/root.html, right after the app.css link so its rules win
// on cascade order. Not Go, so no gofmt pass.
func wireKioskRootHTMLTheme(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "views", "root.html")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "themeStyle") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor:      `    <link rel="stylesheet" href="{{ vite "resources/css/app.css" }}" />`,
			replacement: "    <link rel=\"stylesheet\" href=\"{{ vite \"resources/css/app.css\" }}\" />\n    {{ .themeStyle }}",
		},
	}

	patched, err := applyPatches(path, string(data), patches)
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(patched), 0o644)
}

// themeLibTS is resources/js/lib/theme.ts — a new file, so a plain write.
// Mirrors app/theme.go's cssVarBlock/renderThemeStyle exactly: the client
// side only ever needs this for a runtime update (an Inertia navigation
// landing on a different resolved theme, or a live preview while picking a
// preset), never for the initial paint — Go already rendered that.
const themeLibTS = `// Mirrors app/theme.go's themeColors/renderThemeStyle — the client-side
// half of the no-flash theme system. Initial page loads never need this:
// Go already renders the correct CSS into #tenant-theme server-side (see
// root.html). This module exists only for runtime updates.

// No custom json tags on the Go struct behind this — capitalized Go field
// names come through verbatim as JSON keys.
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

// The resolved theme shared prop — page.props.theme, set by every
// request's SharedProps (app/middleware.go).
export interface ResolvedTheme {
  preset: string;
  light: ThemeColors;
  dark: ThemeColors;
  appearance: Appearance;
}

function cssVarBlock(c: ThemeColors): string {
  const pairs: [string, string][] = [
    ["--background", c.Background],
    ["--foreground", c.Foreground],
    ["--card", c.Card],
    ["--card-foreground", c.CardForeground],
    ["--popover", c.Popover],
    ["--popover-foreground", c.PopoverForeground],
    ["--primary", c.Primary],
    ["--primary-foreground", c.PrimaryForeground],
    ["--secondary", c.Secondary],
    ["--secondary-foreground", c.SecondaryForeground],
    ["--muted", c.Muted],
    ["--muted-foreground", c.MutedForeground],
    ["--accent", c.Accent],
    ["--accent-foreground", c.AccentForeground],
    ["--destructive", c.Destructive],
    ["--destructive-foreground", c.DestructiveForeground],
    ["--border", c.Border],
    ["--input", c.Input],
    ["--ring", c.Ring],
    ["--sidebar", c.Sidebar],
    ["--sidebar-foreground", c.SidebarForeground],
    ["--sidebar-primary", c.SidebarPrimary],
    ["--sidebar-primary-foreground", c.SidebarPrimaryForeground],
    ["--sidebar-accent", c.SidebarAccent],
    ["--sidebar-accent-foreground", c.SidebarAccentForeground],
    ["--sidebar-border", c.SidebarBorder],
    ["--sidebar-ring", c.SidebarRing],
  ];
  return pairs.map(([k, v]) => ` + "`${k}:${v};`" + `).join("");
}

// renderThemeCSS mirrors app/theme.go's renderThemeStyle exactly — same
// three shapes (light/dark forced, or a prefers-color-scheme media query
// for "system"), so a client-side re-application never disagrees with what
// Go would have rendered for the same (preset, appearance) pair.
function renderThemeCSS(theme: ResolvedTheme): string {
  if (theme.appearance === "dark") return ` + "`:root{${cssVarBlock(theme.dark)}}`" + `;
  if (theme.appearance === "light") return ` + "`:root{${cssVarBlock(theme.light)}}`" + `;
  return (
    ` + "`:root{${cssVarBlock(theme.light)}}`" + ` +
    ` + "`@media (prefers-color-scheme: dark){:root{${cssVarBlock(theme.dark)}}}`" + `
  );
}

// applyTheme rewrites the #tenant-theme <style> tag Go injected into
// root.html — the single mechanism both the initial SSR paint and every
// runtime update share. Skips the DOM write when nothing changed.
export function applyTheme(theme: ResolvedTheme) {
  const styleTag = document.getElementById("tenant-theme");
  if (!styleTag) return;
  const next = renderThemeCSS(theme);
  if (styleTag.textContent !== next) {
    styleTag.textContent = next;
  }
}
`

// wireKioskThemeLib writes resources/js/lib/theme.ts if it isn't there yet.
func wireKioskThemeLib(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "js", "lib", "theme.ts")
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(themeLibTS), 0o644)
}

// wireKioskAppTSXTheme adds a router.on("navigate", ...) listener to
// resources/js/app.tsx that reapplies the theme on every Inertia
// navigation — a client-side SPA visit swaps page.props without touching
// <head>, so this is the one place that has to reapply it after the
// initial SSR paint. Whole-file patch: pristine spark's app.tsx is tiny
// (14 lines) and carries no router.on listeners of its own to interleave
// with (unlike shop's, which layers this alongside flash/error toasts —
// see the README if a later step adds those here too).
func wireKioskAppTSXTheme(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "js", "app.tsx")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "applyTheme") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `import { createInertiaApp, type ResolvedComponent } from "@inertiajs/react";
import { resolvePageComponent } from "laravel-vite-plugin/inertia-helpers";
import { createRoot } from "react-dom/client";

const pages = import.meta.glob<ResolvedComponent>("./Pages/**/*.tsx");

createInertiaApp({
  resolve: (name) =>
    resolvePageComponent<ResolvedComponent>(` + "`./Pages/${name}.tsx`" + `, pages),
  setup({ el, App, props }) {
    createRoot(el).render(<App {...props} />);
  },
});`,
			replacement: `import { createInertiaApp, router, type ResolvedComponent } from "@inertiajs/react";
import { resolvePageComponent } from "laravel-vite-plugin/inertia-helpers";
import { createRoot } from "react-dom/client";

import { applyTheme } from "@/lib/theme";
import { type SharedProps } from "@/types";

const pages = import.meta.glob<ResolvedComponent>("./Pages/**/*.tsx");

createInertiaApp({
  resolve: (name) =>
    resolvePageComponent<ResolvedComponent>(` + "`./Pages/${name}.tsx`" + `, pages),
  setup({ el, App, props }) {
    createRoot(el).render(<App {...props} />);
  },
});

// Keeps the theme in sync across Inertia navigation and after a save (see
// Settings/Index.tsx's Appearance tab, Profile's own appearance form) —
// the initial full page load never needs this, Go already rendered the
// right CSS into #tenant-theme (root.html), but a client-side SPA
// navigation swaps page.props without touching <head>, so this is the one
// place that has to reapply it. applyTheme (lib/theme.ts) is a no-op when
// the resolved theme hasn't actually changed.
router.on("navigate", (event) => {
  const theme = (event.detail.page.props as unknown as SharedProps).theme;
  if (theme) applyTheme(theme);
});`,
		},
	}

	patched, err := applyPatches(path, string(data), patches)
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(patched), 0o644)
}

// wireKioskTypesTheme adds the theme prop + appearance field to
// resources/js/types/index.d.ts. Not Go, so no gofmt pass.
func wireKioskTypesTheme(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "js", "types", "index.d.ts")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "ResolvedTheme") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor:      `import type { LucideIcon } from "lucide-react";`,
			replacement: "import type { LucideIcon } from \"lucide-react\";\nimport type { ResolvedTheme } from \"@/lib/theme\";",
		},
		{
			anchor: `  email_verified_at: string | null;
  must_change_password: boolean;
  created_at: string;
  updated_at: string;
}`,
			replacement: `  email_verified_at: string | null;
  must_change_password: boolean;
  // light | dark | system — see app/theme.go.
  appearance: "light" | "dark" | "system";
  created_at: string;
  updated_at: string;
}`,
		},
		{
			anchor: `  errors: Record<string, string[]> | null;
  flash: Record<string, unknown> | null;
  translations: Record<string, string>;
  [key: string]: unknown;
}`,
			replacement: `  errors: Record<string, string[]> | null;
  flash: Record<string, unknown> | null;
  translations: Record<string, string>;
  theme: ResolvedTheme;
  [key: string]: unknown;
}`,
		},
	}

	patched, err := applyPatches(path, string(data), patches)
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(patched), 0o644)
}
