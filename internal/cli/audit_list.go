// NewAPI Tools - audit list command
package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Bonus520/newapi-tools/internal/audit"
	"github.com/Bonus520/newapi-tools/internal/ui"
	"github.com/spf13/cobra"
)

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit log entries",
	Long:  `List audit log entries with optional filtering. Shows the most recent entries first.`,
	RunE:  runAuditList,
}

func init() {
	auditListCmd.Flags().Int("last", 0, "show last N entries (0 = all)")
	auditListCmd.Flags().String("cmd", "", "filter by command name (substring match)")
	auditListCmd.Flags().String("since", "", "filter entries after this time (format: 2006-01-02 or 2006-01-02T15:04:05)")
	auditListCmd.Flags().Bool("json", false, "output as JSON")

	auditCmd.AddCommand(auditListCmd)
}

func runAuditList(cmd *cobra.Command, args []string) error {
	last, _ := cmd.Flags().GetInt("last")
	cmdFilter, _ := cmd.Flags().GetString("cmd")
	sinceStr, _ := cmd.Flags().GetString("since")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Parse since time
	var since time.Time
	if sinceStr != "" {
		// Try parsing both formats
		var err error
		since, err = time.Parse("2006-01-02", sinceStr)
		if err != nil {
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				return fmt.Errorf("invalid time format: use 2006-01-02 or 2006-01-02T15:04:05")
			}
		}
	}

	// Read audit log
	reader := audit.NewAuditReader("")
	entries, err := reader.List(audit.ListOption{
		Last:    last,
		Command: cmdFilter,
		Since:   since,
	})
	if err != nil {
		return err
	}

	if jsonOutput {
		// Output as JSON array
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Output as table
	if len(entries) == 0 {
		fmt.Println("No audit log entries found.")
		return nil
	}

	table := ui.NewTable("TIME", "CMD", "USER", "RESULT", "DURATION")
	for _, entry := range entries {
		result := entry.Result
		if entry.Error != "" {
			result += fmt.Sprintf(" (%s)", truncate(entry.Error, 30))
		}
		duration := fmt.Sprintf("%dms", entry.DurationMs)
		table.AddRow(
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Command,
			entry.User,
			result,
			duration,
		)
	}
	table.Render()

	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
