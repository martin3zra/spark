package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/martin3zra/kiosk"
	"github.com/martin3zra/playsql"
)

func kioskInstallCommand() {
	root, err := os.Getwd()
	if err != nil {
		exitErr(err)
	}

	plan, err := kiosk.PlanInstall(root)
	if err != nil {
		exitErr(err)
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
		exitErr(err)
	}

	fmt.Printf("\nKiosk %s installed — %d file(s) tracked in .kiosk-lock.\n", lock.KioskVersion, len(lock.Files))
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add a developer allowlist to app/config.go — kiosk.Options.Allowlist")
	fmt.Println("     below needs it, and it doesn't exist on a fresh clone. See the README's")
	fmt.Println("     \"Optional: Kiosk ops console\" section for the exact Config/LoadConfig/")
	fmt.Println("     parseAllowlist snippet.")
	fmt.Println("  2. Add to app/routes.go:")
	fmt.Println(`       import "github.com/martin3zra/kiosk"`)
	fmt.Println(`       kiosk.Register(s.route, kiosk.Options{ DB: s.play, Middleware: []routing.Middleware{s.AuthenticatedMiddleware}, Allowlist: s.config.kiosk.developers })`)
	fmt.Println("  3. Add KIOSK_DEVELOPERS to .env")
	fmt.Println("  4. Run: camel migrate && npm run build")
	fmt.Println("\nOptional — redirect kiosk developers to /kiosks until they impersonate a")
	fmt.Println("company member, and show a \"stop impersonating\" banner elsewhere in the app:")
	fmt.Println("see the README's \"Company-aware access control\" subsection for the full")
	fmt.Println("login/middleware/frontend wiring.")
}

// kioskUpdateCommand bumps the kiosk dependency, then re-runs kiosk:install
// to pick up whatever migrations/pages changed in that version. args[0], if
// present, pins a version ("v0.2.0"); otherwise it updates to @latest.
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
func kioskUpdateCommand(args []string) {
	root, err := os.Getwd()
	if err != nil {
		exitErr(err)
	}

	version := "latest"
	if len(args) > 0 {
		version = args[0]
	}
	target := "github.com/martin3zra/kiosk@" + version

	fmt.Println("go get " + target)
	if err := runCmd(root, "go", "get", target); err != nil {
		exitErr(err)
	}
	if err := runCmd(root, "go", "mod", "tidy"); err != nil {
		exitErr(err)
	}

	fmt.Println("\nDependency updated. Re-running kiosk:install in a fresh process to pick up any new or changed migrations/pages...")
	if err := runCmd(root, "go", "run", "./cmd/cli", "kiosk:install"); err != nil {
		exitErr(err)
	}
}

func runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func kioskUninstallCommand(db *playsql.DB) {
	root, err := os.Getwd()
	if err != nil {
		exitErr(err)
	}

	ctx := context.Background()
	plan, err := kiosk.PlanUninstall(ctx, root, db)
	if err != nil {
		exitErr(err)
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
		return
	}

	if err := kiosk.ApplyUninstall(root, plan, skip); err != nil {
		exitErr(err)
	}

	fmt.Println("\nKiosk's copied files removed. Now:")
	fmt.Println("  1. Remove the kiosk import + kiosk.Register(...) call from app/routes.go")
	fmt.Println("  2. Remove KIOSK_DEVELOPERS from .env")
	fmt.Println("  3. Run: go mod tidy")
}

func confirm(reader *bufio.Reader, prompt string) bool {
	fmt.Print(prompt + " [y/N] ")
	answer, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(answer)) == "y"
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
