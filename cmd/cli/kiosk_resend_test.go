package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResendWiringExistingInstall(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app"), 0755); err != nil {
		t.Fatal(err)
	}
	config := `package app
import (
 "os"
 "github.com/martin3zra/forge/mailer"
 "github.com/martin3zra/kiosk"
)
var _ kiosk.AllowlistConfig
var cfg = struct {mail mailer.Config}{mail: mailer.Config{Password: os.Getenv("MAIL_PASSWORD")}}
`
	path := filepath.Join(root, "app", "config.go")
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte("KIOSK_DEVELOPERS=owner@example.com\nMAIL_DRIVER=smtp\nMAIL_PASSWORD=existing\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, wire := range []func(string) (bool, error){wireKioskConfig, wireKioskEnv} {
		if changed, err := wire(root); err != nil || !changed {
			t.Fatalf("first run: %v, %v", changed, err)
		}
		if changed, err := wire(root); err != nil || changed {
			t.Fatalf("second run: %v, %v", changed, err)
		}
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), `APIKey:   os.Getenv("RESEND_API_KEY")`) && !strings.Contains(string(got), `APIKey: os.Getenv("RESEND_API_KEY")`) {
		t.Fatalf("missing API key: %s", got)
	}
	env, _ := os.ReadFile(envPath)
	if strings.Count(string(env), "RESEND_API_KEY=") != 1 || !strings.Contains(string(env), "MAIL_PASSWORD=existing") || !strings.Contains(string(env), "MAIL_DRIVER=smtp") {
		t.Fatalf("changed credentials: %s", env)
	}
}
