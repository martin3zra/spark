package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// anchorNotFoundError means a wiring step's expected anchor text wasn't
// found in the target file — it has diverged from what a fresh spark new
// project looks like. The caller falls back to printing manual instructions
// for that one step rather than guessing at a patch.
type anchorNotFoundError struct {
	file string
}

func (e *anchorNotFoundError) Error() string {
	return fmt.Sprintf("%s: expected content not found, skipping auto-wiring", e.file)
}

type wirePatch struct {
	anchor      string
	replacement string
}

// applyPatches runs every patch against content in memory and returns the
// fully patched result, or an *anchorNotFoundError on the first missing
// anchor. It never touches disk — callers only write the result once every
// patch in the set has matched, so a file is either fully wired or left
// completely untouched, never partially patched (e.g. an import added
// without the code that uses it, which wouldn't compile).
func applyPatches(path, content string, patches []wirePatch) (string, error) {
	for _, p := range patches {
		if !strings.Contains(content, p.anchor) {
			return "", &anchorNotFoundError{file: path}
		}
		content = strings.Replace(content, p.anchor, p.replacement, 1)
	}
	return content, nil
}

func gofmtFile(path string) error {
	return exec.Command("gofmt", "-w", path).Run()
}

// wireKioskConfig adds the developer allowlist and mail config to
// app/config.go: Config struct fields, their LoadConfig() wiring, and the
// mailDriverOrDefault helper. Idempotent — a re-run (e.g. via kiosk:update)
// is a no-op once config.go already has the wiring.
func wireKioskConfig(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "config.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "kiosk.AllowlistConfig") {
		content := string(data)
		if strings.Contains(content, `os.Getenv("RESEND_API_KEY")`) {
			return false, nil
		}
		anchor := "mail: mailer.Config{"
		if !strings.Contains(content, anchor) {
			return false, fmt.Errorf("cannot find mail configuration; add APIKey: os.Getenv(\"RESEND_API_KEY\") manually")
		}
		content = strings.Replace(content, anchor, anchor+"\nAPIKey: os.Getenv(\"RESEND_API_KEY\"),", 1)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return false, err
		}
		return true, gofmtFile(path)
	}

	patches := []wirePatch{
		{
			anchor: `import (
	"encoding/base64"
	"log"
	"os"
	"strconv"
)`,
			replacement: `import (
	"encoding/base64"
	"log"
	"os"
	"strconv"

	"github.com/martin3zra/forge/mailer"
	"github.com/martin3zra/kiosk"
)`,
		},
		{
			anchor: `type Config struct {
	appName      string
	isProduction bool
	host         string
	port         string
	db           Database
	session      Session
	secretKey    []byte
}`,
			replacement: `type Config struct {
	appName      string
	isProduction bool
	host         string
	port         string
	db           Database
	session      Session
	secretKey    []byte
	kiosk        kiosk.AllowlistConfig
	mail         mailer.Config
}`,
		},
		{
			anchor: `		session: Session{
			lifetime: lifetime,
			cookie:   "spark_session",
			domain:   os.Getenv("APP_URL"),
			secure:   isProduction,
			httpOnly: true,
		},
	}
}`,
			replacement: `		session: Session{
			lifetime: lifetime,
			cookie:   "spark_session",
			domain:   os.Getenv("APP_URL"),
			secure:   isProduction,
			httpOnly: true,
		},
		kiosk: kiosk.AllowlistConfig{
			Developers: kiosk.ParseAllowlist(os.Getenv("KIOSK_DEVELOPERS")),
		},
		mail: mailer.Config{
			Driver:      mailDriverOrDefault(os.Getenv("MAIL_DRIVER")),
			APIKey:      os.Getenv("RESEND_API_KEY"),
			Host:        os.Getenv("MAIL_HOST"),
			Port:        os.Getenv("MAIL_PORT"),
			FromAddress: os.Getenv("MAIL_FROM_ADDRESS"),
			FromName:    os.Getenv("MAIL_FROM_NAME"),
			Username:    os.Getenv("MAIL_USERNAME"),
			Password:    os.Getenv("MAIL_PASSWORD"),
			Encryption:  os.Getenv("MAIL_ENCRYPTION"),
		},
	}
}`,
		},
		{
			anchor: `func dbDriverOrDefault(driver string) string {
	if driver == "" {
		return "postgres"
	}
	return driver
}`,
			replacement: `func dbDriverOrDefault(driver string) string {
	if driver == "" {
		return "postgres"
	}
	return driver
}

// mailDriverOrDefault defaults to mailer.Log — logging the rendered email
// (to LOG_FILE) instead of sending it — so a fresh clone works with zero
// mail setup. Set MAIL_DRIVER once you have real credentials.
func mailDriverOrDefault(driver string) mailer.MailDriver {
	if driver == "" {
		return mailer.Log
	}
	return mailer.MailDriver(driver)
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

// wireKioskServer adds the mailer.Mailer field to Server and builds it in
// NewServer from config.mail. Idempotent like wireKioskConfig.
func wireKioskServer(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "server.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "mailer.Mailer") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `	"github.com/martin3zra/forge/i18n"
	"github.com/martin3zra/forge/inertia"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/forge/store"
	"github.com/martin3zra/playsql"
)`,
			replacement: `	"github.com/martin3zra/forge/i18n"
	"github.com/martin3zra/forge/inertia"
	"github.com/martin3zra/forge/mailer"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/forge/store"
	"github.com/martin3zra/playsql"
)`,
		},
		{
			anchor: `type Server struct {
	qs             store.Query
	db             *sql.DB
	play           *playsql.DB
	config         *Config
	sessionManager *session.SessionManager
	assets         embed.FS
	resources      embed.FS
	translator     *i18n.Translator
	route          *routing.Router
	httpServer     *http.Server
}`,
			replacement: `type Server struct {
	qs             store.Query
	db             *sql.DB
	play           *playsql.DB
	config         *Config
	sessionManager *session.SessionManager
	assets         embed.FS
	resources      embed.FS
	translator     *i18n.Translator
	route          *routing.Router
	httpServer     *http.Server
	mailer         mailer.Mailer
}`,
		},
		{
			anchor: `	return &Server{
		qs:         qs,
		config:     LoadConfig(),
		translator: i18n.NewTranslator(localesFS, defaultLang, fallbackLang, trans("auth")),
		assets:     assets,
		resources:  resources,
	}
}`,
			replacement: `	config := LoadConfig()
	return &Server{
		qs:         qs,
		config:     config,
		mailer:     mailer.New(config.mail, resources),
		translator: i18n.NewTranslator(localesFS, defaultLang, fallbackLang, trans("auth")),
		assets:     assets,
		resources:  resources,
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

// wireKioskRoutes adds the kiosk import and the kiosk.Register(...) call at
// the end of bootRoutes(). Idempotent like wireKioskConfig.
func wireKioskRoutes(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "routes.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "kiosk.Register(") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `import (
	"net/http"

	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/routing"
)`,
			replacement: `import (
	"net/http"

	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/kiosk"
)`,
		},
		{
			anchor: `		route.GET("/passwords/create", s.createPasswordPrompt).
			WithoutMiddleware(Verified)
		route.POST("/passwords", s.createPasswordHandler()).
			WithoutMiddleware(Verified)
	})
}`,
			replacement: `		route.GET("/passwords/create", s.createPasswordPrompt).
			WithoutMiddleware(Verified)
		route.POST("/passwords", s.createPasswordHandler()).
			WithoutMiddleware(Verified)
	})

	kiosk.Register(s.route, kiosk.Options{
		DB:         s.play,
		Middleware: []routing.Middleware{s.AuthenticatedMiddleware},
		Allowlist:  s.config.kiosk.Developers,
		Mailer:     s.mailer,
		AppURL:     s.config.host,
	})
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

// wireKioskEnv appends the KIOSK_DEVELOPERS + MAIL_* keys .env needs but
// .env.example doesn't carry by default (they're only meaningful once Kiosk
// is installed). Idempotent like the other wire steps.
func wireKioskEnv(root string) (applied bool, err error) {
	path := filepath.Join(root, ".env")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	content := string(data)
	if strings.Contains(content, "KIOSK_DEVELOPERS=") {
		if strings.Contains(content, "RESEND_API_KEY=") {
			return false, nil
		}
		content = strings.TrimRight(content, "\n") + "\nRESEND_API_KEY=\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return false, err
		}
		return true, nil
	}

	addition := "\n# Kiosk — developer allowlist + mail (added by kiosk:install). Leave\n" +
		"# MAIL_DRIVER unset to log invitations (to LOG_FILE) instead of sending them.\n" +
		"KIOSK_DEVELOPERS=\n" +
		"MAIL_DRIVER=\n" +
		"RESEND_API_KEY=\n" +
		"MAIL_HOST=\n" +
		"MAIL_PORT=\n" +
		"MAIL_FROM_ADDRESS=\n" +
		"MAIL_FROM_NAME=\n" +
		"MAIL_USERNAME=\n" +
		"MAIL_PASSWORD=\n" +
		"MAIL_ENCRYPTION=\n"

	content = strings.TrimRight(content, "\n") + "\n" + addition
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// wireKioskAuthHandlers adds company_id session tracking to app/auth-
// handlers.go: resolved at login (so a kiosk developer's redirect guard has
// something to check), cleared at logout. Idempotent like the other steps.
func wireKioskAuthHandlers(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "auth-handlers.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "companyMembership") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `import (
	"net/http"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/forge/support"
)`,
			replacement: `import (
	"net/http"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/forge/support"
	"github.com/martin3zra/playsql"
)`,
		},
		{
			anchor: `	if err := s.sessionManager.ReGenerate(ctx.Request, user, map[string]any{}); err != nil {
		ctx.Error(err)
		return
	}

	if intended := sess.Get("intended"); intended != nil {`,
			replacement: `	if err := s.sessionManager.ReGenerate(ctx.Request, user, map[string]any{}); err != nil {
		ctx.Error(err)
		return
	}

	if companyID, ok := resolveCompanyID(ctx, s.play, user.GetAuthIdentifier()); ok {
		sess.Put("company_id", companyID)
	} else {
		sess.Delete("company_id")
	}

	if intended := sess.Get("intended"); intended != nil {`,
		},
		{
			anchor: `func (s *Server) logoutHandler(ctx *routing.Context) {
	if err := s.sessionManager.Invalidate(ctx.Request); err != nil {
		ctx.Error(err)
		return
	}
	ctx.Redirect("/login", http.StatusSeeOther)
}`,
			replacement: `func (s *Server) logoutHandler(ctx *routing.Context) {
	if err := s.sessionManager.Invalidate(ctx.Request); err != nil {
		ctx.Error(err)
		return
	}
	session.GetSession(ctx.Request).Delete("company_id")
	ctx.Redirect("/login", http.StatusSeeOther)
}`,
		},
		{
			anchor: `func (s *Server) createPasswordHandler() routing.HandlerFunc {
	return routing.WithRequest(func(ctx *routing.Context, form *CreatePasswordForm) {
		user := AuthUserFromContext(ctx.Request.Context())
		if err := user.MarkPasswordAsChanged(s.play, form.Password); err != nil {
			ctx.BackWith("password", err.Error())
			return
		}
		ctx.Redirect("/home", http.StatusSeeOther)
	})
}`,
			replacement: `func (s *Server) createPasswordHandler() routing.HandlerFunc {
	return routing.WithRequest(func(ctx *routing.Context, form *CreatePasswordForm) {
		user := AuthUserFromContext(ctx.Request.Context())
		if err := user.MarkPasswordAsChanged(s.play, form.Password); err != nil {
			ctx.BackWith("password", err.Error())
			return
		}
		ctx.Redirect("/home", http.StatusSeeOther)
	})
}

// companyMembership mirrors just enough of kiosk's company_users pivot to
// read which company (if any) a logging-in user belongs to. Kiosk owns that
// table; importing the kiosk package here would point the module
// relationship backwards, so this is a narrow read-only shadow of its shape.
type companyMembership struct {
	CompanyID int ` + "`db:\"company_id\"`" + `
}

func (companyMembership) TableName() string { return "company_users" }

func resolveCompanyID(ctx *routing.Context, db *playsql.DB, userID int) (int, bool) {
	var row companyMembership
	if err := db.Model(&companyMembership{}).WhereEq("user_id", userID).First(ctx.Request.Context(), &row); err != nil {
		return 0, false
	}
	return row.CompanyID, true
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

// wireKioskMiddleware adds isDeveloper/isImpersonating to SharedProps and
// the RedirectDeveloperToKiosk guard to app/middleware.go. Idempotent like
// the other steps.
func wireKioskMiddleware(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "middleware.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "RedirectDeveloperToKiosk") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `import (
	"context"
	"net/http"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/database"
	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/romsar/gonertia/v3"
)`,
			replacement: `import (
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
		},
		{
			anchor: `		props := map[string]any{
			"appName": s.config.appName,
			"auth": map[string]any{
				"user": sess.Get("user"),
			},`,
			replacement: `		props := map[string]any{
			"appName": s.config.appName,
			"auth": map[string]any{
				"user":            sess.Get("user"),
				"isDeveloper":     isDeveloperSession(sess, s.config.kiosk.Developers),
				"isImpersonating": sess.Get(kiosk.ImpersonateSessionKey) != nil,
			},`,
		},
		{
			anchor: `// Signed verifies a temporary signed URL.
func (s *Server) Signed(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		if routing.VerifyRequest(ctx.Request, string(s.config.secretKey)) {
			next(ctx)
			return
		}
		ctx.Error(foundation.Unauthorized{})
	}
}`,
			replacement: `// Signed verifies a temporary signed URL.
func (s *Server) Signed(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		if routing.VerifyRequest(ctx.Request, string(s.config.secretKey)) {
			next(ctx)
			return
		}
		ctx.Error(foundation.Unauthorized{})
	}
}

// isDeveloperSession reads the session's raw "user" payload directly rather
// than through AuthUserFromContext/auth.ContextUserID{} — SharedProps runs
// before AuthenticatedMiddleware seeds that context value.
func isDeveloperSession(sess *session.Session, developers map[string]bool) bool {
	raw, ok := sess.Get("user").(map[string]any)
	if !ok {
		return false
	}
	email, _ := raw["email"].(string)
	return developers[strings.ToLower(email)]
}

// RedirectDeveloperToKiosk sends a kiosk developer to /kiosks until they're
// impersonating a company member — company_id in session is the signal
// (set/cleared by kiosk's impersonateStart/Stop), not a separate flag:
// developers never get one at login, only by impersonating.
func (s *Server) RedirectDeveloperToKiosk(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		sess := session.GetSession(ctx.Request)
		if isDeveloperSession(sess, s.config.kiosk.Developers) && sess.Get("company_id") == nil {
			ctx.Redirect("/kiosks", http.StatusSeeOther)
			return
		}
		next(ctx)
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

// wireKioskRoutesAccessControl adds RedirectDeveloperToKiosk to the
// authenticated route group (excluded from /logout). Separate from
// wireKioskRoutes (which adds kiosk.Register) since this piece is reported
// as its own step — it changes app-wide login behavior, not just Kiosk's own
// routes. Idempotent like the other steps.
func wireKioskRoutesAccessControl(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "routes.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "RedirectDeveloperToKiosk") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `	// Authenticated routes.
	s.route.GroupWithMiddleware([]routing.Middleware{s.AuthenticatedMiddleware, Verified}, func(route *routing.Router) {
		route.GET("/home", s.homeHandler).Can("view:dashboard")
		route.POST("/logout", s.logoutHandler)

		route.GET("/passwords/create", s.createPasswordPrompt).
			WithoutMiddleware(Verified)
		route.POST("/passwords", s.createPasswordHandler()).
			WithoutMiddleware(Verified)
	})`,
			replacement: `	// Authenticated routes.
	s.route.GroupWithMiddleware([]routing.Middleware{s.AuthenticatedMiddleware, Verified, s.RedirectDeveloperToKiosk}, func(route *routing.Router) {
		route.GET("/home", s.homeHandler).Can("view:dashboard")
		route.POST("/logout", s.logoutHandler).
			WithoutMiddleware(s.RedirectDeveloperToKiosk)

		route.GET("/passwords/create", s.createPasswordPrompt).
			WithoutMiddleware(Verified)
		route.POST("/passwords", s.createPasswordHandler()).
			WithoutMiddleware(Verified)
	})`,
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

// wireKioskAuthType adds isDeveloper/isImpersonating to the frontend's Auth
// type (resources/js/types/index.d.ts). Idempotent like the other steps.
// Not Go, so no gofmt pass — the anchor/replacement are already formatted
// to match the file's existing 2-space style.
func wireKioskAuthType(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "js", "types", "index.d.ts")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "isDeveloper") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `export interface Auth {
  user: User | null;
}`,
			replacement: `export interface Auth {
  user: User | null;
  isDeveloper: boolean;
  isImpersonating: boolean;
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
	return true, nil
}

// impersonationBannerTSX is resources/js/components/impersonation-banner.tsx
// — a new file, so it's a plain write rather than a patch. See the README's
// "Company-aware access control" section for the source of truth this
// mirrors.
const impersonationBannerTSX = `import { usePage } from "@inertiajs/react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { useCsrfHeaders } from "@/lib/csrf";
import { type SharedProps } from "@/types";

export function ImpersonationBanner() {
  const { auth } = usePage<SharedProps>().props;
  const headers = useCsrfHeaders();
  const [stopping, setStopping] = useState(false);

  if (!auth.isImpersonating) return null;

  async function stop() {
    setStopping(true);
    await fetch("/kiosks/impersonate", { method: "DELETE", headers });
    // Hard navigation — the session identity just changed back to the
    // developer's, so every prop needs a fresh full-page load.
    window.location.assign("/kiosks");
  }

  return (
    <div className="flex items-center justify-between gap-3 bg-amber-500 px-4 py-2 text-sm font-medium text-amber-950">
      <span>Impersonating <strong>{auth.user?.email}</strong></span>
      <Button type="button" size="xs" variant="outline" disabled={stopping} onClick={stop}>
        {stopping ? "Stopping…" : "Stop impersonating"}
      </Button>
    </div>
  );
}
`

// wireKioskImpersonationBanner writes impersonation-banner.tsx if it isn't
// there yet. Idempotent (a pre-existing file — however it got there — is
// left alone rather than overwritten).
func wireKioskImpersonationBanner(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "js", "components", "impersonation-banner.tsx")
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	if err := os.WriteFile(path, []byte(impersonationBannerTSX), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// wireKioskTopnavLayout renders ImpersonationBanner above everything in the
// topnav shell — AppTopNav is a normal-flow element (no fixed/sticky
// positioning), so a banner placed before it just pushes the nav down, no
// overlap. Idempotent like the other steps.
func wireKioskTopnavLayout(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "js", "layouts", "app", "app-topnav-layout.tsx")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "ImpersonationBanner") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `import { AppContent } from "@/components/app-content";
import { AppShell } from "@/components/app-shell";
import { AppTopNav } from "@/components/app-topnav";
import { AppTopNavHeader } from "@/components/app-topnav-header";`,
			replacement: `import { AppContent } from "@/components/app-content";
import { AppShell } from "@/components/app-shell";
import { AppTopNav } from "@/components/app-topnav";
import { AppTopNavHeader } from "@/components/app-topnav-header";
import { ImpersonationBanner } from "@/components/impersonation-banner";`,
		},
		{
			anchor: `  return (
    <AppShell variant="header">
      <AppTopNav />`,
			replacement: `  return (
    <AppShell variant="header">
      <ImpersonationBanner />
      <AppTopNav />`,
		},
	}

	patched, err := applyPatches(path, string(data), patches)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// wireKioskSidebarLayout renders ImpersonationBanner inside the content
// column (SidebarInset), not above the shell — the sidebar itself is
// `fixed inset-y-0`, anchored to the viewport regardless of DOM position, so
// a banner placed above the whole shell would sit at the same y=0 the
// sidebar does and get visually covered by it rather than pushing it down.
// Idempotent like the other steps.
func wireKioskSidebarLayout(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "js", "layouts", "app", "app-sidebar-layout.tsx")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "ImpersonationBanner") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `import { AppContent } from "@/components/app-content";
import { AppShell } from "@/components/app-shell";
import { AppSidebar } from "@/components/app-sidebar";
import { AppSidebarHeader } from "@/components/app-sidebar-header";`,
			replacement: `import { AppContent } from "@/components/app-content";
import { AppShell } from "@/components/app-shell";
import { AppSidebar } from "@/components/app-sidebar";
import { AppSidebarHeader } from "@/components/app-sidebar-header";
import { ImpersonationBanner } from "@/components/impersonation-banner";`,
		},
		{
			anchor: `      <AppContent variant="sidebar">
        <AppSidebarHeader breadcrumbs={breadcrumbs} />`,
			replacement: `      <AppContent variant="sidebar">
        <ImpersonationBanner />
        <AppSidebarHeader breadcrumbs={breadcrumbs} />`,
		},
	}

	patched, err := applyPatches(path, string(data), patches)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
