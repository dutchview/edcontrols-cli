package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/mauricejumelet/edcontrols-cli/internal/api"
)

type ContractsCmd struct {
	List ContractsListCmd `cmd:"" help:"List contracts (clients)"`
}

type ContractsListCmd struct {
	JSON bool `short:"j" help:"Output as JSON"`
}

// ContractInfo represents contract info for display
type ContractInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProjectCount int    `json:"projectCount"`
	Active       bool   `json:"active"`
	PricePlan    string `json:"pricePlan,omitempty"`
}

func (c *ContractsListCmd) Run(client *api.Client) error {
	projects, _, err := client.ListProjects(api.ListProjectsOptions{})
	if err != nil {
		return err
	}

	// Extract unique contract IDs and count projects per contract
	contractProjects := make(map[string]int)
	for _, p := range projects {
		if p.Contract != "" {
			contractProjects[p.Contract]++
		}
	}

	// Fetch contract details for each unique contract
	var contracts []ContractInfo
	for contractID, projectCount := range contractProjects {
		contract, err := client.GetContract(contractID)
		if err != nil {
			// If we can't fetch the contract, still show it with ID only
			contracts = append(contracts, ContractInfo{
				ID:           contractID,
				Name:         "(unknown)",
				ProjectCount: projectCount,
			})
			continue
		}

		contracts = append(contracts, ContractInfo{
			ID:           contractID,
			Name:         contract.Name,
			ProjectCount: projectCount,
			Active:       contract.Active,
			PricePlan:    contract.PricePlan,
		})
	}

	// Sort by name
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Name < contracts[j].Name
	})

	if c.JSON {
		return printJSON(contracts)
	}

	if len(contracts) == 0 {
		fmt.Println("No contracts found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPROJECTS\tACTIVE\tPLAN")
	fmt.Fprintln(w, "----\t--------\t------\t----")

	for _, contract := range contracts {
		active := "Yes"
		if !contract.Active {
			active = "No"
		}
		plan := contract.PricePlan
		if plan == "" {
			plan = "-"
		}
		name := truncate(contract.Name, 40)
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", name, contract.ProjectCount, active, plan)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d contracts\n", len(contracts))

	return nil
}
