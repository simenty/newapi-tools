// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Bonus520/newapi-tools/internal/apperr"
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
	statusCmd.Flags().Bool("watch", false, "watch mode: continuously refresh status")
	statusCmd.Flags().Int("interval", 2, "refresh interval in seconds (for --watch)")
	statusCmd.Flags().Bool("all", false, "show all newapi-related containers")

	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := core.GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	watch, _ := cmd.Flags().GetBool("watch")
	interval, _ := cmd.Flags().GetInt("interval")
	showAll, _ := cmd.Flags().GetBool("all")

	if interval < 1 {
		interval = 2
	}

	// Watch mode: loop until Ctrl+C
	if watch {
		return runWatchMode(cmd, cfg, interval, showAll)
	}

	// Single-shot mode
	return showStatus(cmd, cfg, showAll)
}

// runWatchMode continuously refreshes the status display until the user presses Ctrl+C.
func runWatchMode(cmd *cobra.Command, cfg *core.Config, interval int, showAll bool) error {
	// Set up signal handling for graceful exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// Show initial status immediately
	clearScreen()
	if err := showStatus(cmd, cfg, showAll); err != nil {
		return err
	}

	for {
		select {
		case <-sigCh:
			fmt.Println("\nStopped watching.")
			return nil
		case <-ticker.C:
			clearScreen()
			if err := showStatus(cmd, cfg, showAll); err != nil {
				// In watch mode, print the error but keep watching
				ui.PrintError(err)
			}
		}
	}
}

// clearScreen clears the terminal for watch mode refresh.
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// showStatus displays the status once (called by both single-shot and watch modes).
func showStatus(cmd *cobra.Command, cfg *core.Config, showAll bool) error {
	client, err := docker.NewClient()
	if err != nil {
		return apperr.Wrap(apperr.CodeDockerNotFound, "", err)
	}
	defer client.Close()

	if !client.IsAvailable() {
		ui.L().Error("docker daemon is not running")
		return apperr.New(apperr.CodeDockerDaemonDown, "Docker daemon 不可访问", "", nil)
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")

	if showAll {
		return showAllContainers(cmd.Context(), client, cfg, jsonOutput)
	}

	// Single container mode
	container, err := client.FindContainerByName(cmd.Context(), "new-api")
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

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

	// Try to get resource stats
	var stats *docker.ContainerStats
	if container.State == "running" {
		stats, _ = docker.GetContainerStats("new-api")
	}

	if jsonOutput {
		statusJSON := fmt.Sprintf(`{"status": "%s", "container": "%s", "image": "%s", "state": "%s"`,
			state, container.Name, container.Image, container.State)
		if stats != nil {
			statusJSON += fmt.Sprintf(`, "cpu": "%s", "mem": "%s"`, stats.CPUPerc, stats.MemPerc)
		}
		statusJSON += "}"
		fmt.Println(statusJSON)
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
	if stats != nil {
		tbl.AddRow("CPU%", stats.CPUPerc)
		tbl.AddRow("MEM%", stats.MemPerc)
		tbl.AddRow("Mem Usage", stats.MemUsage)
		tbl.AddRow("Net I/O", stats.NetIO)
		tbl.AddRow("Block I/O", stats.BlockIO)
	}
	tbl.Render()

	return nil
}

// showAllContainers displays the status of all newapi-related containers.
func showAllContainers(ctx context.Context, client *docker.Client, cfg *core.Config, jsonOutput bool) error {
	containers, err := client.ContainerList(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Filter to newapi-related containers
	var related []docker.ContainerInfo
	for _, c := range containers {
		if isRelatedContainer(c.Name) {
			related = append(related, c)
		}
	}

	if len(related) == 0 {
		if jsonOutput {
			fmt.Println(`[]`)
		} else {
			fmt.Println("No newapi-related containers found.")
			fmt.Println("Run 'newapi-tools install' to deploy new-api.")
		}
		return nil
	}

	if jsonOutput {
		fmt.Println("[")
		for i, c := range related {
			comma := ","
			if i == len(related)-1 {
				comma = ""
			}
			fmt.Printf(`  {"name": %q, "image": %q, "state": %q, "status": %q}%s`+"\n",
				c.Name, c.Image, c.State, c.Status, comma)
		}
		fmt.Println("]")
		return nil
	}

	// Table display with stats
	tbl := ui.NewTable("Container", "Image", "State", "Status", "CPU%", "MEM%")
	for _, c := range related {
		cpuPerc := "-"
		memPerc := "-"
		if c.State == "running" {
			if stats, err := docker.GetContainerStats(c.Name); err == nil {
				cpuPerc = stats.CPUPerc
				memPerc = stats.MemPerc
			}
		}
		tbl.AddRow(c.Name, c.Image, c.State, c.Status, cpuPerc, memPerc)
	}
	tbl.Render()

	return nil
}

// isRelatedContainer checks if a container name is related to newapi.
func isRelatedContainer(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "new-api") ||
		strings.Contains(lower, "newapi") ||
		strings.Contains(lower, "mysql") ||
		strings.Contains(lower, "redis")
}
