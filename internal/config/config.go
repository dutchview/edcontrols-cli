package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Token string
}

// ConfigLocations returns the list of config file locations that are checked
// in order of priority (first found wins).
func ConfigLocations() []string {
	locations := []string{
		".env", // Current directory
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		// XDG-style config directory
		locations = append(locations, filepath.Join(homeDir, ".config", "edcontrols-cli", ".env"))
	}

	return locations
}

// Load loads configuration from environment variables and optional .env files.
// The configFile parameter allows specifying a custom config file path.
// If empty, the default locations are checked in order:
//  1. .env in current directory
//  2. ~/.config/edcontrols-cli/.env
//
// Environment variables always take precedence over file values.
func Load(configFile string) (*Config, error) {
	// If a specific config file is provided, load only that one
	if configFile != "" {
		if err := godotenv.Load(configFile); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configFile, err)
		}
	} else {
		// Try default locations in order (first found wins)
		for _, loc := range ConfigLocations() {
			if _, err := os.Stat(loc); err == nil {
				_ = godotenv.Load(loc)
				break
			}
		}
	}

	token := os.Getenv("EDCONTROLS_ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("EDCONTROLS_ACCESS_TOKEN not set.\n\n%s", configHelp())
	}

	return &Config{
		Token: token,
	}, nil
}

func configHelp() string {
	locations := ConfigLocations()
	var sb strings.Builder

	sb.WriteString("Configuration can be provided via:\n")
	sb.WriteString("  1. Environment variable EDCONTROLS_ACCESS_TOKEN\n")
	sb.WriteString("  2. A .env file in one of these locations:\n")
	for _, loc := range locations {
		sb.WriteString(fmt.Sprintf("     - %s\n", loc))
	}
	sb.WriteString("  3. A custom config file via --config flag\n")
	sb.WriteString("  4. Command line via --token flag\n")
	sb.WriteString("\nExample .env file:\n")
	sb.WriteString("  EDCONTROLS_ACCESS_TOKEN=your_bearer_token\n")
	sb.WriteString("\nGet your token from the EdControls web interface.")

	return sb.String()
}

// PrintConfigHelp prints the configuration help message.
func PrintConfigHelp() {
	fmt.Println("EdControls CLI Configuration")
	fmt.Println("============================")
	fmt.Println()
	fmt.Println(configHelp())
	fmt.Println()
	fmt.Println("Token format: UUID (36 characters)")
	fmt.Println("Example: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
}
