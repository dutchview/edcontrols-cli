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

	// Commands
	Whoami    cmd.WhoamiCmd    `cmd:"" help:"Show current user info (-j for JSON)"`
	Contracts cmd.ContractsCmd `cmd:"" help:"Manage contracts/clients (list, projects)"`
	Projects  cmd.ProjectsCmd  `cmd:"" help:"Manage projects (list, get) with search and glacier support"`
	Tickets   cmd.TicketsCmd   `cmd:"" help:"Manage tickets (list, get, update, assign, open, close, archive, unarchive, delete)"`
	Audits    cmd.AuditsCmd    `cmd:"" help:"Manage audits (list, get, create from template)"`
	Templates cmd.TemplatesCmd `cmd:"" help:"Manage audit templates (list, get, create, update, publish, unpublish) and groups (list, create)"`
	Maps      cmd.MapsCmd      `cmd:"" help:"Manage maps/drawings (list, get, add, delete, tags) and groups (list)"`
	Files     cmd.FilesCmd     `cmd:"" help:"Manage files (list, get, add, download, archive, unarchive, delete, tags, to-map) and groups (list)"`
	Configure ConfigureCmd     `cmd:"" help:"Show configuration help and setup instructions"`
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
		// If token is provided via flag, we can skip config file
		if CLI.Token != "" {
			cfg = &config.Config{
				Token: CLI.Token,
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Override config with command line flag if provided
	if CLI.Token != "" {
		cfg.Token = CLI.Token
	}

	// Create API client
	client := api.NewClient(cfg)

	// Run the command with the client
	err = ctx.Run(client)
	ctx.FatalIfErrorf(err)
}
