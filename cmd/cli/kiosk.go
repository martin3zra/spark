package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/martin3zra/kiosk"
)

type kioskInstallCmd struct{}

func (kioskInstallCmd) Name() string { return "kiosk:install" }
func (kioskInstallCmd) Description() string {
	return "Copy Kiosk's migrations/pages into this project and wire it into config/server/routes/.env"
}

const manualConfigInstructions = `    kiosk kiosk.AllowlistConfig  // field on your Config struct
    mail mailer.Config  // field on your Config struct
    kiosk: kiosk.AllowlistConfig{Developers: kiosk.ParseAllowlist(os.Getenv("KIOSK_DEVELOPERS"))}  // in LoadConfig()
    mail: mailer.Config{Driver: mailDriverOrDefault(os.Getenv("MAIL_DRIVER")), Host: os.Getenv("MAIL_HOST"), Port: os.Getenv("MAIL_PORT"), FromAddress: os.Getenv("MAIL_FROM_ADDRESS"), FromName: os.Getenv("MAIL_FROM_NAME"), Username: os.Getenv("MAIL_USERNAME"), Password: os.Getenv("MAIL_PASSWORD"), Encryption: os.Getenv("MAIL_ENCRYPTION")}  // in LoadConfig()
    func mailDriverOrDefault(driver string) mailer.MailDriver { if driver == "" { return mailer.Log }; return mailer.MailDriver(driver) }
`

const manualServerInstructions = `    mailer mailer.Mailer  // field on your Server struct
    mailer: mailer.New(config.mail, resources)  // in NewServer() — resources is the same embed.FS already passed in
`

const manualRoutesInstructions = `    import "github.com/martin3zra/kiosk"
    kiosk.Register(s.route, kiosk.Options{ DB: s.play, Middleware: []routing.Middleware{s.AuthenticatedMiddleware}, Allowlist: s.config.kiosk.Developers, Mailer: s.mailer, AppURL: s.config.host })
`

const manualAuthHandlersInstructions = `    See the README's "Company-aware access control" section for
    app/auth-handlers.go: a companyMembership type + resolveCompanyID
    helper, called in authHandler() right after ReGenerate, and a
    session.Delete("company_id") in logoutHandler() right after Invalidate.
`

const manualMiddlewareInstructions = `    See the README's "Company-aware access control" section for
    app/middleware.go: isDeveloper/isImpersonating added to SharedProps'
    "auth" map, plus the isDeveloperSession and RedirectDeveloperToKiosk
    functions.
`

const manualRoutesAccessControlInstructions = `    Add s.RedirectDeveloperToKiosk to the authenticated route group's
    middleware list, and .WithoutMiddleware(s.RedirectDeveloperToKiosk) on
    POST /logout. See the README's "Company-aware access control" section.
`

const manualAuthTypeInstructions = `    Add isDeveloper: boolean; isImpersonating: boolean; to the Auth
    interface in resources/js/types/index.d.ts.
`

const manualTopnavLayoutInstructions = `    Render <ImpersonationBanner /> as the first child of <AppShell
    variant="header"> in resources/js/layouts/app/app-topnav-layout.tsx,
    above <AppTopNav />. See the README's "Company-aware access control"
    section.
`

const manualSidebarLayoutInstructions = `    Render <ImpersonationBanner /> as the first child inside <AppContent
    variant="sidebar"> in resources/js/layouts/app/app-sidebar-layout.tsx —
    NOT above <AppShell>, since the sidebar is fixed/inset-y-0 and would
    render on top of a banner placed there. See the README's "Company-aware
    access control" section.
`

const manualThemeCurrentInstructions = `    In app/theme.go, make currentTheme return
    s.withCompanyTheme(ctx, sess, resolveTheme(sess)), so a company's
    preset applies when the user hasn't picked one (app/theme_company.go).
`

const manualSettingsFormsInstructions = `    See the README's "Tenant theme + Settings" section for app/forms.go:
    the clearEmptyStrings helper plus UpdateCompanySettingsForm,
    UpdateProfileForm, UpdatePasswordForm, UpdateEmailForm, and
    UpdateAppearanceForm.
`

const manualSettingsRoutesInstructions = `    See the README's "Tenant theme + Settings" section for app/routes.go:
    /verify-email, /email/verify, /settings, /company, /profile,
    /profile/password, /profile/email, and /profile/appearance routes in
    the authenticated group, plus a new s.Signed-only group for
    GET /email/confirm.
`

// applyWireStep runs one auto-wiring step and reports its outcome. An
// anchorNotFoundError (the target file has diverged from the template)
// degrades to printing manual instructions instead of failing the install.
func applyWireStep(root, label string, apply func(string) (bool, error), manual string) error {
	applied, err := apply(root)
	if err != nil {
		var anchorErr *anchorNotFoundError
		if errors.As(err, &anchorErr) {
			fmt.Printf("  ! %s has diverged from the template — couldn't auto-wire. Add by hand:\n", label)
			fmt.Print(manual)
			return nil
		}
		return fmt.Errorf("wiring %s: %w", label, err)
	}
	if applied {
		fmt.Printf("  ✓ wired %s\n", label)
	}
	return nil
}

func (kioskInstallCmd) Run(_ *CommandContext, _ []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	plan, err := kiosk.PlanInstall(root)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	skip := map[string]bool{}
	for _, f := range plan.Files {
		switch f.Status {
		case kiosk.StatusNew:
			fmt.Printf("  + %s (%s)\n", f.Path, f.Kind)
		case kiosk.StatusUpdated:
			fmt.Printf("  ~ %s (%s) — updating\n", f.Path, f.Kind)
		case kiosk.StatusUnchanged:
			// nothing to report
		case kiosk.StatusConflict:
			fmt.Printf("  ! %s has local changes kiosk:install didn't make.\n", f.Path)
			if !confirm(reader, "    Overwrite it?") {
				skip[f.Path] = true
				fmt.Println("    kept your version.")
			}
		}
	}

	lock, err := kiosk.ApplyInstall(root, plan, skip)
	if err != nil {
		return err
	}

	fmt.Printf("\nKiosk %s installed — %d file(s) tracked in .kiosk-lock.\n", lock.KioskVersion, len(lock.Files))

	fmt.Println("\nWiring Kiosk into the app:")
	if err := applyWireStep(root, "app/config.go", wireKioskConfig, manualConfigInstructions); err != nil {
		return err
	}
	if err := applyWireStep(root, "app/server.go", wireKioskServer, manualServerInstructions); err != nil {
		return err
	}
	if err := applyWireStep(root, "app/routes.go", wireKioskRoutes, manualRoutesInstructions); err != nil {
		return err
	}
	if applied, err := wireKioskEnv(root); err != nil {
		return fmt.Errorf("wiring .env: %w", err)
	} else if applied {
		fmt.Println("  ✓ wired .env")
	}
	if err := applyWireStep(root, "app/auth-handlers.go", wireKioskAuthHandlers, manualAuthHandlersInstructions); err != nil {
		return err
	}
	if err := applyWireStep(root, "app/middleware.go", wireKioskMiddleware, manualMiddlewareInstructions); err != nil {
		return err
	}
	if err := applyWireStep(root, "app/routes.go (developer redirect)", wireKioskRoutesAccessControl, manualRoutesAccessControlInstructions); err != nil {
		return err
	}
	if err := applyWireStep(root, "resources/js/types/index.d.ts", wireKioskAuthType, manualAuthTypeInstructions); err != nil {
		return err
	}
	if applied, err := wireKioskImpersonationBanner(root); err != nil {
		return fmt.Errorf("wiring resources/js/components/impersonation-banner.tsx: %w", err)
	} else if applied {
		fmt.Println("  ✓ wired resources/js/components/impersonation-banner.tsx")
	}
	if err := applyWireStep(root, "resources/js/layouts/app/app-topnav-layout.tsx", wireKioskTopnavLayout, manualTopnavLayoutInstructions); err != nil {
		return err
	}
	if err := applyWireStep(root, "resources/js/layouts/app/app-sidebar-layout.tsx", wireKioskSidebarLayout, manualSidebarLayoutInstructions); err != nil {
		return err
	}

	fmt.Println("\nWiring the company theme + Settings/Profile baseline:")
	if applied, err := wireKioskThemeCompany(root); err != nil {
		return fmt.Errorf("wiring app/theme_company.go: %w", err)
	} else if applied {
		fmt.Println("  ✓ wired app/theme_company.go")
	}
	if err := applyWireStep(root, "app/theme.go (company preset)", wireKioskThemeCurrent, manualThemeCurrentInstructions); err != nil {
		return err
	}
	if applied, err := wireKioskCompanyMembers(root); err != nil {
		return fmt.Errorf("wiring app/company-members.go: %w", err)
	} else if applied {
		fmt.Println("  ✓ wired app/company-members.go")
	}
	if applied, err := wireKioskCompanySettings(root); err != nil {
		return fmt.Errorf("wiring app/company-settings.go: %w", err)
	} else if applied {
		fmt.Println("  ✓ wired app/company-settings.go")
	}
	if applied, err := wireKioskProfile(root); err != nil {
		return fmt.Errorf("wiring app/profile.go: %w", err)
	} else if applied {
		fmt.Println("  ✓ wired app/profile.go")
	}
	if err := applyWireStep(root, "app/forms.go (settings)", wireKioskSettingsForms, manualSettingsFormsInstructions); err != nil {
		return err
	}
	if err := applyWireStep(root, "app/routes.go (settings)", wireKioskSettingsRoutes, manualSettingsRoutesInstructions); err != nil {
		return err
	}
	if applied, err := wireKioskVerifyEmailPage(root); err != nil {
		return fmt.Errorf("wiring resources/js/Pages/Auth/VerifyEmail.tsx: %w", err)
	} else if applied {
		fmt.Println("  ✓ wired resources/js/Pages/Auth/VerifyEmail.tsx")
	}
	if applied, err := wireKioskSettingsPage(root); err != nil {
		return fmt.Errorf("wiring resources/js/Pages/Settings/Index.tsx: %w", err)
	} else if applied {
		fmt.Println("  ✓ wired resources/js/Pages/Settings/Index.tsx")
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Set KIOSK_DEVELOPERS in .env to your own email (comma-separated for more).")
	fmt.Println("  2. Once you have real mail credentials, set MAIL_DRIVER/MAIL_HOST/MAIL_PORT/")
	fmt.Println("     MAIL_FROM_ADDRESS/MAIL_FROM_NAME/MAIL_USERNAME/MAIL_PASSWORD/MAIL_ENCRYPTION")
	fmt.Println("     in .env (leave MAIL_DRIVER unset and invitations just get logged instead")
	fmt.Println("     of sent).")
	fmt.Println("  3. Run: camel migrate && npm run build")
	fmt.Println("\nA kiosk developer (an email in KIOSK_DEVELOPERS) is now redirected to")
	fmt.Println("/kiosks on login until they impersonate a company member — see the README's")
	fmt.Println("\"Company-aware access control\" section if that's not what you want (e.g.")
	fmt.Println("Kiosk is just an admin console for you, not a login boundary).")
	return nil
}

type kioskUpdateCmd struct{}

func (kioskUpdateCmd) Name() string { return "kiosk:update" }
func (kioskUpdateCmd) Description() string {
	return "Bump the kiosk dependency, then re-run kiosk:install"
}

// Run bumps the kiosk dependency, then re-runs kiosk:install to pick up
// whatever migrations/pages changed in that version. args[0], if present,
// pins a version ("v0.2.0"); otherwise it updates to @latest.
//
// Deliberately `go get <module>@version`, not `go get -u`: -u would also
// upgrade every other already-required module (forge, playsql, ...) to
// their latest compatible versions as a side effect — surprising for a
// command whose only job is moving kiosk's version.
//
// The install step has to run as a fresh `go run` subprocess rather than a
// plain function call: Go resolves which version of an imported package's
// code runs at compile time, so this already-running binary is still built
// against the pre-update kiosk — only a new process picks up go.mod's
// change.
func (kioskUpdateCmd) Run(_ *CommandContext, args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	version := "latest"
	if len(args) > 0 {
		version = args[0]
	}
	target := "github.com/martin3zra/kiosk@" + version

	fmt.Println("go get " + target)
	if err := runCmd(root, "go", "get", target); err != nil {
		return err
	}
	if err := runCmd(root, "go", "mod", "tidy"); err != nil {
		return err
	}

	fmt.Println("\nDependency updated. Re-running kiosk:install in a fresh process to pick up any new or changed migrations/pages...")
	return runCmd(root, "go", "run", "./cmd/cli", "kiosk:install")
}

func runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type kioskUninstallCmd struct{}

func (kioskUninstallCmd) Name() string        { return "kiosk:uninstall" }
func (kioskUninstallCmd) Description() string { return "Remove the files kiosk:install copied in" }

func (kioskUninstallCmd) Run(ctx *CommandContext, _ []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	planCtx := context.Background()
	plan, err := kiosk.PlanUninstall(planCtx, root, ctx.DB)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	skip := map[string]bool{}
	for _, f := range plan.Files {
		if f.Status != kiosk.StatusConflict {
			continue
		}
		fmt.Printf("  ! %s has local changes kiosk:install didn't make.\n", f.Path)
		if !confirm(reader, "    Delete it anyway?") {
			skip[f.Path] = true
			fmt.Println("    kept.")
		}
	}

	anyData := false
	for _, tc := range plan.TableCounts {
		if tc.Count > 0 {
			anyData = true
			fmt.Printf("  ⚠ %s: %d row(s) — not deleted, will be orphaned\n", tc.Table, tc.Count)
		}
	}
	if anyData && !confirm(reader, "\nContinue uninstalling Kiosk's code?") {
		fmt.Println("aborted.")
		return nil
	}

	if err := kiosk.ApplyUninstall(root, plan, skip); err != nil {
		return err
	}

	fmt.Println("\nKiosk's copied files removed. Now:")
	fmt.Println("  1. Remove the kiosk import + kiosk.Register(...) call from app/routes.go")
	fmt.Println("  2. Remove KIOSK_DEVELOPERS from .env")
	fmt.Println("  3. Run: go mod tidy")
	return nil
}

func confirm(reader *bufio.Reader, prompt string) bool {
	fmt.Print(prompt + " [y/N] ")
	answer, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(answer)) == "y"
}
