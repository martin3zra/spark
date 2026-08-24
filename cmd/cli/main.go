package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/martin3zra/spark/app"
	"github.com/martin3zra/spark/resources"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: spark <command> [arguments]")
		fmt.Println()
		listCommands()
		return
	}

	loadEnv()
	if f := app.ConfigureLogging(os.Getenv("LOG_FILE")); f != nil {
		defer f.Close()
	}
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

// dispatch looks a command up in the registry (commands.go) and runs it.
// PreBootCommand runs immediately — no Config, no DB, so it works even
// before APP_KEY exists (generate:key's whole reason for being one).
// BootCommand needs a live server: Config loaded, DB connected, torn down
// again once the command returns.
//
// "list"/"help" bypass the registry — they're about the registry, not in it.
func dispatch(name string, args []string) error {
	if name == "list" || name == "help" {
		listCommands()
		return nil
	}

	cmd := findCommand(name)
	if cmd == nil {
		listCommands()
		return fmt.Errorf("unknown command: %s", name)
	}

	if pre, ok := cmd.(PreBootCommand); ok {
		return pre.RunPreBoot(args)
	}

	boot, ok := cmd.(BootCommand)
	if !ok {
		return fmt.Errorf("%s: registered but implements neither PreBootCommand nor BootCommand", name)
	}

	var assets embed.FS
	server := app.NewServer(assets, resources.Views)
	db := server.Boot()

	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := boot.Run(&CommandContext{Server: server, DB: db}, args); err != nil {
		return err
	}
	return server.Shutdown(shutdownCtx)
}
