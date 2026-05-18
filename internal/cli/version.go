// NewAPI Tools - Docker management platform for newapi
package cli

import (
	"fmt"

	"github.com/Bonus520/newapi-tools/internal/core"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display version, git commit, and build date for newapi-tools.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("newapi-tools %s\n", core.Version)
		fmt.Printf("  commit: %s\n", core.GitCommit)
		fmt.Printf("  built:  %s\n", core.BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
