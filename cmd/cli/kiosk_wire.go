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
		return false, nil
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
		return false, nil
	}

	addition := "\n# Kiosk — developer allowlist + mail (added by kiosk:install). Leave\n" +
		"# MAIL_DRIVER unset to log invitations (to LOG_FILE) instead of sending them.\n" +
		"KIOSK_DEVELOPERS=\n" +
		"MAIL_DRIVER=\n" +
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
