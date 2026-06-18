// Package cli provides CLI commands and execution flow for Ralphex.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/iyaki/ralphex/internal/config"
)

const maxPositionalArgs = 2

// NewRalphCommand creates the root command for Ralphex.
func NewRalphCommand() *cobra.Command {
	var cfg config.Config

	cmd := &cobra.Command{
		Use:   "ralph [options] [prompt] [scope]",
		Short: "POSIX-compliant AI Agentic Loop runner for spec-driven development",
		Long: `Ralphex is a POSIX-compliant AI agentic loop shell runner.
It is a Ralph-Wiggum inspired spec-driven agent runner.
It loads prompts from files (with optional inline overrides) and comes with build/plan presets.

The loop runs until the agent emits <promise>COMPLETE</promise> or max iterations is reached.
When writing custom prompts, include <promise>COMPLETE</promise> at the end to signal completion.

Custom Prompt Example:
  ralph --prompt "Implement feature X. When done, output <promise>COMPLETE</promise>"
  ralph --prompt "Task Y. Output <COMPLETION_SIGNAL>"

For extended documentation, examples, and configuration options, visit https://github.com/iyaki/ralphex.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: `  ralph build
  ralph plan my-feature
  ralph --max-iterations 10 build
  ralph --prompt "Custom prompt text"
  echo "prompt from stdin" | ralph -`,
		Args: cobra.MaximumNArgs(maxPositionalArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommandLogic(cmd, args, &cfg)
		},
	}

	setupSharedFlags(cmd, &cfg)

	// Register subcommands
	cmd.AddCommand(NewInitCommand())
	cmd.AddCommand(NewRunCommand())
	cmd.AddCommand(NewVersionCommand())

	return cmd
}
