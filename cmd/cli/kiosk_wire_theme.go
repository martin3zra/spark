package main

import (
	"os"
	"path/filepath"
	"strings"
)

// The theme engine itself (app/theme.go, resources/js/lib/theme.ts,
// users.appearance and users.theme_settings, the root.html wiring) ships with
// the base template. kiosk:install only layers the company's preset on top:
// companies.theme_preset applies whenever the signed-in user hasn't picked a
// preset of their own in the theme customizer.

// themeCompanyGo is app/theme_company.go, a new file, so a plain write.
const themeCompanyGo = `package app

import (
	"context"

	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/kiosk"
	"github.com/martin3zra/playsql"
)

// companyThemeRow is a narrow, read-only shadow of companies used only for
// the theme lookup that runs on every request (SharedProps), before any
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
// back to "default" on any lookup failure (missing row, no DB bound, etc.),
// since theme resolution must never fail a page load.
func resolveCompanyThemePreset(ctx context.Context, db *playsql.DB, companyID int) string {
	var row companyThemeRow
	if err := db.Model(&companyThemeRow{}).WhereEq("id", companyID).First(ctx, &row); err != nil {
		return defaultThemePresetID
	}
	return row.ThemePreset
}

// withCompanyTheme gives theme the company's preset when the user hasn't
// picked one in the theme customizer (app/theme.go, currentTheme).
//
// company_id alone isn't a reliable "is this a tenant page" signal: a
// kiosk-allowlisted developer who is also a real member of some company
// still carries that company_id while browsing Kiosk's own cross-tenant
// pages, and Kiosk must never take on one tenant's brand. Impersonation is
// the one case where "acting as this company" means seeing its theme.
func (s *Server) withCompanyTheme(ctx context.Context, sess *session.Session, theme resolvedTheme) resolvedTheme {
	if theme.PresetChosen {
		return theme
	}
	if isDeveloperSession(sess, s.config.kiosk.Developers) && sess.Get(kiosk.ImpersonateSessionKey) == nil {
		return theme
	}
	companyID, ok := companyIDFromSession(sess)
	if !ok {
		return theme
	}
	return theme.withPreset(resolveCompanyThemePreset(ctx, s.play, companyID))
}
`

// wireKioskThemeCompany writes app/theme_company.go if it isn't there yet.
// Idempotent like wireKioskImpersonationBanner.
func wireKioskThemeCompany(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "theme_company.go")
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.WriteFile(path, []byte(themeCompanyGo), 0o644); err != nil {
		return false, err
	}
	return true, gofmtFile(path)
}

// wireKioskThemeCurrent routes the base template's currentTheme (app/theme.go)
// through withCompanyTheme. Runs after wireKioskMiddleware, which adds
// isDeveloperSession.
func wireKioskThemeCurrent(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "theme.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "withCompanyTheme(") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `func (s *Server) currentTheme(ctx context.Context, sess *session.Session) resolvedTheme {
	return resolveTheme(sess)
}`,
			replacement: `func (s *Server) currentTheme(ctx context.Context, sess *session.Session) resolvedTheme {
	return s.withCompanyTheme(ctx, sess, resolveTheme(sess))
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
	return true, gofmtFile(path)
}
