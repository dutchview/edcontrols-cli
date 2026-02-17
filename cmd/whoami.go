package cmd

import (
	"fmt"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type WhoamiCmd struct {
	JSON bool `short:"j" help:"Output as JSON"`
}

func (c *WhoamiCmd) Run(client *api.Client) error {
	userInfo, err := client.GetCurrentUser()
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(userInfo)
	}

	fmt.Printf("Email: %s\n", userInfo.Email)

	if userInfo.Name.FirstName != "" || userInfo.Name.LastName != "" {
		fmt.Printf("Name: %s %s\n", userInfo.Name.FirstName, userInfo.Name.LastName)
	}

	if userInfo.CompanyName != "" {
		fmt.Printf("Company: %s\n", userInfo.CompanyName)
	}

	if len(userInfo.Roles) > 0 {
		fmt.Printf("Roles: %v\n", userInfo.Roles)
	}

	return nil
}
