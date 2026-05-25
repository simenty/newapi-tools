// NewAPI Tools - Docker registry mirror management commands
package cli

import (
	"fmt"
	"strings"

	"github.com/Bonus520/newapi-tools/internal/apperr"
	"github.com/Bonus520/newapi-tools/internal/docker"
	"github.com/spf13/cobra"
)

var mirrorCmd = &cobra.Command{
	Use:   "mirror",
	Short: "Manage Docker registry mirrors",
	Long: `Manage Docker registry mirrors to accelerate image pulls in China.

Writes mirror settings to /etc/docker/daemon.json and reloads Docker daemon.

Built-in mirror shortcuts:
  tuna    - https://docker.mirrors.tuna.tsinghua.edu.cn (Tsinghua University TUNA)
  aliyun  - https://registry.cn-hangzhou.aliyuncs.com  (Alibaba Cloud)
  ustc    - https://docker.mirrors.ustc.edu.cn         (USTC)
  163     - https://hub-mirror.c.163.com               (NetEase 163)
  azure   - https://dockerhub.azk8s.cn                (Azure CN)
  daocloud- https://f1361db2.m.daocloud.io             (DaoCloud)

Examples:
  newapi-tools mirror add tuna               # add Tsinghua mirror
  newapi-tools mirror add tuna aliyun        # add multiple mirrors
  newapi-tools mirror list                   # list current mirrors
  newapi-tools mirror test tuna              # test if mirror is reachable
  newapi-tools mirror apply                  # write daemon.json + reload Docker
  newapi-tools mirror remove tuna            # remove a mirror
  newapi-tools mirror reset                  # clear all mirrors`,
}

var mirrorAddCmd = &cobra.Command{
	Use:   "add <mirror> [mirror...]",
	Short: "Add one or more registry mirrors",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runMirrorAdd,
}

var mirrorRemoveCmd = &cobra.Command{
	Use:     "remove <mirror>",
	Aliases: []string{"rm"},
	Short:   "Remove a registry mirror",
	Args:    cobra.ExactArgs(1),
	RunE:    runMirrorRemove,
}

var mirrorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List current registry mirrors from daemon.json",
	RunE:  runMirrorList,
}

var mirrorApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Write mirrors to /etc/docker/daemon.json and reload Docker",
	RunE:  runMirrorApply,
}

var mirrorTestCmd = &cobra.Command{
	Use:   "test [mirror...]",
	Short: "Test connectivity to registry mirrors",
	RunE:  runMirrorTest,
}

var mirrorResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Remove all registry mirrors from daemon.json",
	RunE:  runMirrorReset,
}

var mirrorBuiltinCmd = &cobra.Command{
	Use:   "builtin",
	Short: "List available built-in mirror shortcuts",
	RunE:  runMirrorBuiltin,
}

func init() {
	mirrorAddCmd.Flags().Bool("apply", true, "immediately write daemon.json and reload Docker")
	mirrorRemoveCmd.Flags().Bool("apply", true, "immediately write daemon.json and reload Docker")
	mirrorResetCmd.Flags().Bool("apply", true, "immediately write daemon.json and reload Docker")

	mirrorCmd.AddCommand(
		mirrorAddCmd,
		mirrorRemoveCmd,
		mirrorListCmd,
		mirrorApplyCmd,
		mirrorTestCmd,
		mirrorResetCmd,
		mirrorBuiltinCmd,
	)
	rootCmd.AddCommand(mirrorCmd)
}

// runMirrorAdd adds one or more mirrors and optionally reloads Docker.
func runMirrorAdd(cmd *cobra.Command, args []string) error {
	autoApply, _ := cmd.Flags().GetBool("apply")

	// Resolve short names to URLs
	resolved := make([]string, 0, len(args))
	for _, arg := range args {
		url, ok := docker.ResolveShortName(arg)
		if !ok {
			return apperr.New(apperr.CodeMirrorApply, fmt.Sprintf("未知镜像源 %q — 请使用 URL 或: %s",
				arg, builtinNamesList()), "", nil)
		}
		resolved = append(resolved, url)
		fmt.Printf("  Adding: %s -> %s\n", arg, url)
	}

	// Get current list and append
	current, err := docker.GetCurrentMirrors()
	if err != nil {
		return fmt.Errorf("failed to read current mirrors: %w", err)
	}
	merged := append(current, resolved...)
	if err := docker.SetMirrors(merged); err != nil {
		return apperr.Wrap(apperr.CodeMirrorApply, "", err)
	}

	fmt.Printf("  daemon.json updated (%d mirror(s) total)\n", len(merged))

	if autoApply {
		return applyAndReload()
	}
	fmt.Println("  Run 'newapi-tools mirror apply' to reload Docker daemon.")
	return nil
}

// runMirrorRemove removes a mirror and optionally reloads Docker.
func runMirrorRemove(cmd *cobra.Command, args []string) error {
	autoApply, _ := cmd.Flags().GetBool("apply")

	url, ok := docker.ResolveShortName(args[0])
	if !ok {
		return apperr.New(apperr.CodeMirrorApply, fmt.Sprintf("未知镜像源: %s", args[0]), "", nil)
	}

	if err := docker.RemoveMirror(url); err != nil {
		return apperr.Wrap(apperr.CodeMirrorApply, "", err)
	}
	fmt.Printf("  Removed: %s\n", url)

	if autoApply {
		return applyAndReload()
	}
	return nil
}

// runMirrorList shows mirrors currently configured in daemon.json.
func runMirrorList(cmd *cobra.Command, args []string) error {
	mirrors, err := docker.GetCurrentMirrors()
	if err != nil {
		return apperr.Wrap(apperr.CodeMirrorApply, "", err)
	}

	if len(mirrors) == 0 {
		fmt.Println("No registry mirrors configured.")
		fmt.Println("Run 'newapi-tools mirror add tuna' to add a mirror.")
		return nil
	}

	fmt.Printf("Registry mirrors (%s):\n", docker.DaemonJSONPath())
	for i, m := range mirrors {
		fmt.Printf("  %d. %s\n", i+1, m)
	}
	return nil
}

// runMirrorApply writes daemon.json and reloads Docker daemon.
func runMirrorApply(cmd *cobra.Command, args []string) error {
	mirrors, err := docker.GetCurrentMirrors()
	if err != nil {
		return apperr.Wrap(apperr.CodeMirrorApply, "", err)
	}
	if len(mirrors) == 0 {
		fmt.Println("No mirrors configured — nothing to apply.")
		fmt.Println("Add mirrors first: newapi-tools mirror add tuna")
		return nil
	}
	return applyAndReload()
}

// runMirrorTest checks connectivity to one or more mirrors.
func runMirrorTest(cmd *cobra.Command, args []string) error {
	toTest := args
	if len(toTest) == 0 {
		// Test all currently configured mirrors
		var err error
		toTest, err = docker.GetCurrentMirrors()
		if err != nil {
			return err
		}
		if len(toTest) == 0 {
			fmt.Println("No mirrors configured. Pass a mirror name or URL to test.")
			return nil
		}
	}

	allOK := true
	for _, m := range toTest {
		url, _ := docker.ResolveShortName(m)
		fmt.Printf("  Testing %-52s ... ", url)
		if err := docker.TestMirror(url); err != nil {
			fmt.Printf("FAIL (%v)\n", err)
			allOK = false
		} else {
			fmt.Println("OK")
		}
	}

	if !allOK {
		return fmt.Errorf("one or more mirrors are unreachable")
	}
	return nil
}

// runMirrorReset clears all mirrors from daemon.json.
func runMirrorReset(cmd *cobra.Command, args []string) error {
	autoApply, _ := cmd.Flags().GetBool("apply")

	if err := docker.SetMirrors(nil); err != nil {
		return apperr.Wrap(apperr.CodeMirrorApply, "", err)
	}
	fmt.Println("  Cleared all registry mirrors from daemon.json.")

	if autoApply {
		return applyAndReload()
	}
	return nil
}

// runMirrorBuiltin lists the available built-in mirror shortcuts.
func runMirrorBuiltin(cmd *cobra.Command, args []string) error {
	fmt.Println("Built-in mirror shortcuts:")
	fmt.Printf("  %-10s  %s\n", "NAME", "URL")
	fmt.Printf("  %-10s  %s\n", strings.Repeat("-", 10), strings.Repeat("-", 50))
	order := []string{"tuna", "aliyun", "ustc", "163", "azure", "daocloud"}
	for _, name := range order {
		if url, ok := docker.BuiltinMirrors[name]; ok {
			fmt.Printf("  %-10s  %s\n", name, url)
		}
	}
	return nil
}

// applyAndReload writes current mirrors and reloads Docker daemon.
func applyAndReload() error {
	mirrors, err := docker.GetCurrentMirrors()
	if err != nil {
		return err
	}
	fmt.Printf("  Applying %d mirror(s) to /etc/docker/daemon.json...\n", len(mirrors))
	fmt.Println("  Reloading Docker daemon...")
	if err := docker.ReloadDocker(); err != nil {
		return apperr.Wrap(apperr.CodeDockerDaemonDown, "", err)
	}
	fmt.Println("  Docker daemon reloaded.")
	fmt.Println("  Mirror(s) active. Next 'docker pull' will use the configured mirrors.")
	return nil
}

// builtinNamesList returns a comma-separated list of built-in mirror names.
func builtinNamesList() string {
	names := []string{"tuna", "aliyun", "ustc", "163", "azure", "daocloud"}
	return strings.Join(names, ", ")
}
