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
	"github.com/martin3zra/playsql"
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

	if err := run(os.Args); err != nil {
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

func run(args []string) error {
	var assets embed.FS
	server := app.NewServer(assets, resources.Views)
	db := server.Boot()

	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	menu(args, db)

	return server.Shutdown(shutdownCtx)
}

func menu(args []string, db *playsql.DB) {
	command := args[1]
	args = args[2:]

	switch command {
	case "version":
		fmt.Println("spark CLI 0.1.0")
	case "generate:key":
		fmt.Println("generate key", str.GenerateRandom())
	case "kiosk:install":
		kioskInstallCommand()
	case "kiosk:update":
		kioskUpdateCommand(args)
	case "kiosk:uninstall":
		kioskUninstallCommand(db)
	default:
		fmt.Printf("Unknown command: %s\n", command)
	}
}
