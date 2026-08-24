package main

import (
	"github.com/martin3zra/playsql"
	"github.com/martin3zra/spark/app"
)

// Command is a CLI subcommand — implement PreBootCommand or BootCommand
// (not this interface directly) to make one runnable. Registering a new
// command is: write the type, add it to the commands slice in commands.go.
type Command interface {
	Name() string        // e.g. "kiosk:install" — what's typed after `go run ./cmd/cli`
	Description() string // one line, shown by `list`/`help`
}

// PreBootCommand runs before the server boots — for anything that must work
// even when Config can't load yet. generate:key is the reason this exists:
// booting loads Config, which requires APP_KEY to already be set, so the
// command that produces APP_KEY can't itself require a boot.
type PreBootCommand interface {
	Command
	RunPreBoot(args []string) error
}

// BootCommand runs after the server has booted (Config loaded, DB
// connected) — most commands are this. Return an error rather than calling
// os.Exit directly: dispatch (main.go) is the single place that prints it
// and sets the exit code, so every command fails the same way.
type BootCommand interface {
	Command
	Run(ctx *CommandContext, args []string) error
}

// CommandContext is what a BootCommand gets instead of reaching for globals.
type CommandContext struct {
	Server *app.Server
	DB     *playsql.DB
}
