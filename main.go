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
	Token  string `help:"Access token (overrides config file)" env:"EDCONTROLS_ACCESS_TOKEN"`
	Email  string `short:"e" help:"User email (overrides config file)" env:"EDCONTROLS_USER_EMAIL"`

	// Commands
	Contracts cmd.ContractsCmd `cmd:"" help:"List contracts (clients)"`
	Projects  cmd.ProjectsCmd  `cmd:"" help:"Manage projects"`
	Tickets   cmd.TicketsCmd   `cmd:"" help:"Manage tickets"`
	Audits    cmd.AuditsCmd    `cmd:"" help:"Manage audits"`
	Templates cmd.TemplatesCmd `cmd:"" help:"Manage audit templates"`
	Maps      cmd.MapsCmd      `cmd:"" help:"Manage maps (drawings)"`
	Files     cmd.FilesCmd     `cmd:"" help:"Manage files"`
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
		// If token and email are provided via flags, we can skip config file
		if CLI.Token != "" && CLI.Email != "" {
			cfg = &config.Config{
				Token: CLI.Token,
				Email: CLI.Email,
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Override config with command line flags if provided
	if CLI.Token != "" {
		cfg.Token = CLI.Token
	}
	if CLI.Email != "" {
		cfg.Email = CLI.Email
	}

	// Create API client
	client := api.NewClient(cfg)

	// Run the command with the client
	err = ctx.Run(client)
	ctx.FatalIfErrorf(err)
}
