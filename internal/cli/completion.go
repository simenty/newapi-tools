package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Output shell completion code for the specified shell.

To use the completions in your shell:

  Bash:
    source <(newapi-tools completion bash)
    # To load completions for each session, add to ~/.bashrc:
    echo 'source <(newapi-tools completion bash)' >> ~/.bashrc

  Zsh:
    source <(newapi-tools completion zsh)
    # To load completions for each session, add to ~/.zshrc:
    echo 'source <(newapi-tools completion zsh)' >> ~/.zshrc

  Fish:
    newapi-tools completion fish | source
    # To load completions for each session:
    newapi-tools completion fish > ~/.config/fish/completions/newapi-tools.fish

  PowerShell:
    newapi-tools completion powershell | Out-String | Invoke-Expression
    # To load completions for each session, add to your PowerShell profile:
    newapi-tools completion powershell >> $PROFILE`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	completionCmd.AddCommand(&cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenBashCompletion(os.Stdout)
		},
	})

	completionCmd.AddCommand(&cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenZshCompletion(os.Stdout)
		},
	})

	completionCmd.AddCommand(&cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenFishCompletion(os.Stdout, true)
		},
	})

	completionCmd.AddCommand(&cobra.Command{
		Use:   "powershell",
		Short: "Generate powershell completion",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		},
	})
}

// GetCompletionCmd returns the completion command for testing.
func GetCompletionCommand() *cobra.Command {
	return completionCmd
}

func ExampleCompletion_bash() {
	_ = rootCmd.GenBashCompletion(os.Stdout)
}