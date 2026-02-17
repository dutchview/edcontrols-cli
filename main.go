package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/mauricejumelet/edcontrols-cli/cmd"
	"github.com/mauricejumelet/edcontrols-cli/internal/api"
	"github.com/mauricejumelet/edcontrols-cli/internal/config"
)

var version = "1.0.0"

var CLI struct {
	// Global flags
	Config string `short:"c" help:"Path to config file (.env format)" type:"path"`

	// Commands
	Projects  cmd.ProjectsCmd  `cmd:"" help:"Manage projects"`
	Tickets   cmd.TicketsCmd   `cmd:"" help:"Manage tickets"`
	Audits    cmd.AuditsCmd    `cmd:"" help:"Manage audits"`
	Templates cmd.TemplatesCmd `cmd:"" help:"Manage audit templates"`
	Maps      cmd.MapsCmd      `cmd:"" help:"Manage maps (drawings)"`
	Configure ConfigureCmd     `cmd:"" help:"Show configuration help"`
}

type ConfigureCmd struct{}

func (c *ConfigureCmd) Run() error {
	config.PrintConfigHelp()
	return nil
}

func main() {
	// Handle version flag early
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--version" {
			fmt.Printf("ec v%s\n", version)
			return
		}
	}

	ctx := kong.Parse(&CLI,
		kong.Name("ec"),
		kong.Description("EdControls CLI - A command-line interface for EdControls (v"+version+")"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	// Commands that don't need the API client
	switch ctx.Command() {
	case "configure":
		err := ctx.Run()
		ctx.FatalIfErrorf(err)
		return
	}

	// Load configuration
	cfg, err := config.Load(CLI.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create API client
	client := api.NewClient(cfg)

	// Run the command with the client
	err = ctx.Run(client)
	ctx.FatalIfErrorf(err)
}
