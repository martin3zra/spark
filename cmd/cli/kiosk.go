package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/martin3zra/kiosk"
)

type kioskInstallCmd struct{}

func (kioskInstallCmd) Name() string { return "kiosk:install" }
func (kioskInstallCmd) Description() string {
	return "Copy Kiosk's migrations and pages into this project"
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
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add a developer allowlist to app/config.go — kiosk.Options.Allowlist")
	fmt.Println("     below needs it, and it doesn't exist on a fresh clone:")
	fmt.Println(`       kiosk kiosk.AllowlistConfig  // field on your Config struct`)
	fmt.Println(`       kiosk: kiosk.AllowlistConfig{Developers: kiosk.ParseAllowlist(os.Getenv("KIOSK_DEVELOPERS"))}  // in LoadConfig()`)
	fmt.Println("  2. Add mail config to app/config.go and app/server.go — creating a company")
	fmt.Println("     always needs an owner, and Kiosk emails an invitation whenever that owner")
	fmt.Println("     isn't an existing user, so kiosk.Options.Mailer below needs a real one:")
	fmt.Println(`       mail mailer.Config  // field on your Config struct`)
	fmt.Println(`       mail: mailer.Config{Driver: mailDriverOrDefault(os.Getenv("MAIL_DRIVER")), Host: os.Getenv("MAIL_HOST"), Port: os.Getenv("MAIL_PORT"), FromAddress: os.Getenv("MAIL_FROM_ADDRESS"), FromName: os.Getenv("MAIL_FROM_NAME"), Username: os.Getenv("MAIL_USERNAME"), Password: os.Getenv("MAIL_PASSWORD"), Encryption: os.Getenv("MAIL_ENCRYPTION")}  // in LoadConfig()`)
	fmt.Println(`       func mailDriverOrDefault(driver string) mailer.MailDriver { if driver == "" { return mailer.Log }; return mailer.MailDriver(driver) }  // logs the email instead of sending it until MAIL_DRIVER is set`)
	fmt.Println(`       mailer mailer.Mailer  // field on your Server struct`)
	fmt.Println(`       mailer: mailer.New(config.mail, resources)  // in NewServer() — resources is the same embed.FS already passed in`)
	fmt.Println("  3. Add to app/routes.go:")
	fmt.Println(`       import "github.com/martin3zra/kiosk"`)
	fmt.Println(`       kiosk.Register(s.route, kiosk.Options{ DB: s.play, Middleware: []routing.Middleware{s.AuthenticatedMiddleware}, Allowlist: s.config.kiosk.Developers, Mailer: s.mailer, AppURL: s.config.host })`)
	fmt.Println("  4. Add KIOSK_DEVELOPERS to .env, plus MAIL_DRIVER/MAIL_HOST/MAIL_PORT/")
	fmt.Println("     MAIL_FROM_ADDRESS/MAIL_FROM_NAME/MAIL_USERNAME/MAIL_PASSWORD/MAIL_ENCRYPTION")
	fmt.Println("     once you have real mail credentials (leave MAIL_DRIVER unset and")
	fmt.Println("     invitations just get logged instead of sent)")
	fmt.Println("  5. Run: camel migrate && npm run build")
	fmt.Println("\nSee the README's \"Mail\" subsection for the full config.go/server.go code.")
	fmt.Println("\nOptional — redirect kiosk developers to /kiosks until they impersonate a")
	fmt.Println("company member, and show a \"stop impersonating\" banner elsewhere in the app:")
	fmt.Println("see the README's \"Company-aware access control\" subsection for the full")
	fmt.Println("login/middleware/frontend wiring.")
	return nil
}

type kioskUpdateCmd struct{}

func (kioskUpdateCmd) Name() string        { return "kiosk:update" }
func (kioskUpdateCmd) Description() string { return "Bump the kiosk dependency, then re-run kiosk:install" }

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
