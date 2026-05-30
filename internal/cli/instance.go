// NewAPI Tools - instance management commands
package cli

import (
	"fmt"
	"path/filepath"

	"github.com/simenty/newapi-tools/internal/apperr"
	"github.com/simenty/newapi-tools/internal/core"
	"github.com/simenty/newapi-tools/internal/instance"
	"github.com/simenty/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

var instanceCmd = &cobra.Command{
	Use:   "instance",
	Short: "Manage new-api instances",
	Long:  `Manage multiple new-api instances. Each instance has its own configuration and Docker Compose project.`,
}

var instanceAddCmd = &cobra.Command{
	Use:   "add NAME",
	Short: "Add a new instance",
	Long:  `Add a new instance with the given name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runInstanceAdd,
}

var instanceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all instances",
	Long:  `List all configured instances with their status.`,
	RunE:  runInstanceList,
}

var instanceSwitchCmd = &cobra.Command{
	Use:   "switch NAME",
	Short: "Switch to a different active instance",
	Long:  `Set the given instance as the active one.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runInstanceSwitch,
}

var instanceRemoveCmd = &cobra.Command{
	Use:   "remove NAME",
	Short: "Remove an instance",
	Long:  `Remove the specified instance. Cannot remove the active instance.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runInstanceRemove,
}

func init() {
	instanceAddCmd.Flags().String("home", "", "home directory for the instance (default: /opt/newapi-NAME)")
	instanceAddCmd.Flags().Int("port", 3000, "port for the instance")
	instanceAddCmd.Flags().String("image", "calciumion/new-api:latest", "Docker image to use")

	instanceCmd.AddCommand(instanceAddCmd)
	instanceCmd.AddCommand(instanceListCmd)
	instanceCmd.AddCommand(instanceSwitchCmd)
	instanceCmd.AddCommand(instanceRemoveCmd)

	rootCmd.AddCommand(instanceCmd)
}

func runInstanceAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	home, _ := cmd.Flags().GetString("home")
	port, _ := cmd.Flags().GetInt("port")
	image, _ := cmd.Flags().GetString("image")

	// Validate name
	if !isValidInstanceName(name) {
		return apperr.New(apperr.CodeInstanceExists, "invalid instance name: must be lowercase letters, numbers, and hyphens, starting with a letter", "", nil)
	}

	// Default home directory
	if home == "" {
		home = filepath.Join("/opt", fmt.Sprintf("newapi-%s", name))
	}

	store := instance.NewStore("")

	// Create new instance
	inst := instance.NewInstance(name, home, port, image)

	if err := store.Add(*inst); err != nil {
		return err
	}

	fmt.Printf("Instance '%s' added successfully!\n", name)
	fmt.Printf("  Home:     %s\n", home)
	fmt.Printf("  Port:     %d\n", port)
	fmt.Printf("  Image:    %s\n", image)
	fmt.Printf("  Compose:  newapi-%s\n", name)
	fmt.Println()
	fmt.Printf("Run 'newapi-tools instance switch %s' to activate it\n", name)

	return nil
}

func runInstanceList(cmd *cobra.Command, args []string) error {
	store := instance.NewStore("")
	instances, err := store.List()
	if err != nil {
		return err
	}

	if len(instances) == 0 {
		fmt.Println("No instances configured.")
		fmt.Println("Use 'newapi-tools instance add NAME' to create one.")
		return nil
	}

	table := ui.NewTable("NAME", "HOME", "PORT", "IMAGE", "CREATED", "ACTIVE")
	for _, inst := range instances {
		active := " "
		if inst.Active {
			active = "*"
		}
		table.AddRow(
			inst.Name,
			inst.Home,
			fmt.Sprintf("%d", inst.Port),
			inst.DockerImage,
			inst.CreatedAt,
			active,
		)
	}
	table.Render()

	return nil
}

func runInstanceSwitch(cmd *cobra.Command, args []string) error {
	name := args[0]

	store := instance.NewStore("")

	if err := store.SetActive(name); err != nil {
		return err
	}

	// Update the config
	cfg := core.GetConfig()
	if cfg == nil {
		cfg = core.DefaultConfig()
	}
	cfg.Instance.Active = name

	configPath := core.ConfigFileUsed()
	if configPath == "" {
		configPath = core.ConfigFilePath()
	}

	if err := core.WriteConfig(cfg, configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("Switched to instance '%s'\n", name)
	fmt.Println("Configuration updated.")

	return nil
}

func runInstanceRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	store := instance.NewStore("")

	if err := store.Remove(name); err != nil {
		return err
	}

	fmt.Printf("Instance '%s' removed successfully\n", name)
	fmt.Println("Note: This does not delete the Docker containers or home directory.")
	fmt.Printf("To clean up manually, run 'docker compose -p newapi-%s down' and delete %s\n", name, fmt.Sprintf("/opt/newapi-%s", name))

	return nil
}

func isValidInstanceName(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Must start with a letter
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')) {
		return false
	}
	// Rest can be letters, numbers, hyphens
	for _, c := range name[1:] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}
