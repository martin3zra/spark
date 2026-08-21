package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/martin3zra/forge/foundation/str"
	"github.com/martin3zra/spark/app"
	"github.com/martin3zra/spark/resources"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: spark <command> [arguments]")
		return
	}

	loadEnv()
	_ = os.Setenv("RUNNING_IN_CLI", "YES")

	if err := dispatch(os.Args[1], os.Args[2:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func loadEnv() {
	wd, _ := os.Getwd()
	if err := godotenv.Load(filepath.Join(wd, ".env")); err == nil {
		return
	}
	exePath, _ := os.Executable()
	root := filepath.Join(filepath.Dir(exePath), "..", "..")
	if err := godotenv.Load(filepath.Join(root, ".env")); err != nil {
		log.Printf("no .env loaded: %v", err)
	}
}

// dispatch runs a command. "generate:key" and "version" are handled before
// booting a server on purpose: booting loads Config, which requires APP_KEY
// to already be set (LoadConfig calls log.Fatal otherwise) — exactly the
// value generate:key exists to produce for a brand-new project that has no
// .env yet. Booting first would make the command that generates APP_KEY
// unable to run until APP_KEY already exists.
func dispatch(command string, args []string) error {
	switch command {
	case "version":
		fmt.Println("spark CLI 0.1.0")
		return nil
	case "generate:key":
		fmt.Println("generate key", str.GenerateRandom())
		return nil
	}

	var assets embed.FS
	server := app.NewServer(assets, resources.Views)
	db := server.Boot()

	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	switch command {
	case "kiosk:install":
		kioskInstallCommand()
	case "kiosk:update":
		kioskUpdateCommand(args)
	case "kiosk:uninstall":
		kioskUninstallCommand(db)
	default:
		fmt.Printf("Unknown command: %s\n", command)
	}

	return server.Shutdown(shutdownCtx)
}
