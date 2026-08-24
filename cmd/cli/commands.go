package main

import (
	"fmt"

	"github.com/martin3zra/forge/foundation/str"
)

// commands is every subcommand spark's CLI knows about. Add a new command
// by writing a type that implements PreBootCommand or BootCommand (see
// command.go) and appending it here — nothing else needs to change to make
// it runnable, listed, and documented.
var commands = []Command{
	versionCommand{},
	generateKeyCommand{},
	kioskInstallCmd{},
	kioskUpdateCmd{},
	kioskUninstallCmd{},
}

func findCommand(name string) Command {
	for _, c := range commands {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func listCommands() {
	fmt.Println("Available commands:")
	width := 0
	for _, c := range commands {
		if len(c.Name()) > width {
			width = len(c.Name())
		}
	}
	for _, c := range commands {
		fmt.Printf("  %-*s  %s\n", width, c.Name(), c.Description())
	}
}

type versionCommand struct{}

func (versionCommand) Name() string        { return "version" }
func (versionCommand) Description() string { return "Print the spark CLI version" }
func (versionCommand) RunPreBoot(args []string) error {
	fmt.Println("spark CLI 0.1.0")
	return nil
}

type generateKeyCommand struct{}

func (generateKeyCommand) Name() string        { return "generate:key" }
func (generateKeyCommand) Description() string { return "Generate a fresh APP_KEY" }
func (generateKeyCommand) RunPreBoot(args []string) error {
	fmt.Println("generate key", str.GenerateRandom())
	return nil
}
