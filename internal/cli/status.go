// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"fmt"

	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/Bonus520/newapi-tools/internal/docker"
	"github.com/Bonus520/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show new-api status",
	Long:  `Display the current status of the new-api container and related services.`,
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().Bool("json", false, "output in JSON format")

	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	defer client.Close()

	if !client.IsAvailable() {
		ui.L().Error("docker daemon is not running")
		return fmt.Errorf("docker daemon is not accessible")
	}

	// Find the new-api container
	container, err := client.FindContainerByName(cmd.Context(), "new-api")
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")

	if container == nil {
		if jsonOutput {
			fmt.Println(`{"status": "not_installed"}`)
		} else {
			fmt.Println("new-api is not installed.")
			fmt.Println("Run 'newapi-tools install' to deploy new-api.")
		}
		return nil
	}

	// Get detailed state
	state, _ := client.ContainerInspect(cmd.Context(), "new-api")

	if jsonOutput {
		fmt.Printf(`{"status": "%s", "container": "%s", "image": "%s", "state": "%s"}`+"\n",
			state, container.Name, container.Image, container.State)
		return nil
	}

	// Display status using the table UI
	tbl := ui.NewTable("Property", "Value")
	tbl.AddRow("Container", container.Name)
	tbl.AddRow("Image", container.Image)
	tbl.AddRow("State", container.State)
	tbl.AddRow("Status", container.Status)
	tbl.AddRow("Health", state)
	tbl.AddRow("Home", cfg.NewAPI.Home)
	tbl.AddRow("Port", fmt.Sprintf("%d", cfg.NewAPI.Port))
	tbl.Render()

	return nil
}
