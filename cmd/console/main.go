package main

import (
	"fmt"
	"location-bot/internal/console"
	"log/slog"
	"os"
)

type Commands []console.Command

func main() {
	slog.Info("Starting console command")

	//conf := config.GetConfig()
	commands := initCommands()
	if len(os.Args) > 1 {
		runCommand(commands, os.Args[1])
	}

	slog.Info("Command finished")
}

func initCommands() *Commands {
	return &Commands{
		console.NewHelpCommand(),
	}
}

func runCommand(commands *Commands, arg string) {
	found := false
	for _, cmd := range *commands {
		if arg == cmd.Name() {
			slog.Info("Command found", "command", cmd.Name())
			found = true
			if err := cmd.Run(); err != nil {
				slog.Error(err.Error())
				os.Exit(1)
			}
			break
		}
	}
	if !found {
		fmt.Printf("Command '%s' not found\n", arg)
	}

	if !found {
		fmt.Println("Usage: location_console <command>")
		for _, cmd := range *commands {
			fmt.Printf("\t%s - %s\n", cmd.Name(), cmd.Description())
		}
	}
}
