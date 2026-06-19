package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iyaki/ralphex/internal/config"
	"github.com/iyaki/ralphex/internal/prompt"
)

// NewPromptsCommand creates the prompts command for listing and viewing prompts.
func NewPromptsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prompts",
		Short: "List and view available prompts",
		Long:  `List and view available built-in and custom prompts.`,
	}

	cmd.AddCommand(NewPromptsListCommand())
	cmd.AddCommand(NewPromptsShowCommand())

	return cmd
}

// NewPromptsListCommand creates the prompts list subcommand.
func NewPromptsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available prompts",
		Long:  `List all available built-in and custom prompts with descriptions.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var cfg config.Config
			if err := cfg.LoadConfig(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			return runPromptsList(cmd.OutOrStdout(), &cfg)
		},
	}
}

// NewPromptsShowCommand creates the prompts show subcommand.
func NewPromptsShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show full prompt content",
		Long:  `Display the full content of a built-in or custom prompt.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg config.Config
			if err := cfg.LoadConfig(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			promptName := args[0]

			return runPromptsShow(cmd.OutOrStdout(), &cfg, promptName)
		},
	}
}

type promptInfo struct {
	Name string
	Desc string
	Path string
}

func runPromptsList(output io.Writer, cfg *config.Config) error {
	// Display built-in prompts
	const (
		maxDescLen  = 80
		maxShortLen = 70
	)
	buildDesc := "Implement a task from IMPLEMENTATION_PLAN.md after studying specs, then validate, commit, update plan."
	planDesc := "Generate/update IMPLEMENTATION_PLAN.md with phase-based plan after studying specs and gaps."

	_, _ = fmt.Fprintln(output, "Built-in Prompts:")
	_, _ = fmt.Fprintf(output, "  build      %s\n", truncateString(buildDesc, maxDescLen))
	_, _ = fmt.Fprintf(output, "  plan       %s\n", truncateString(planDesc, maxDescLen))

	// Discover custom prompts
	customPrompts := discoverCustomPrompts(cfg)
	if len(customPrompts) > 0 {
		_, _ = fmt.Fprintln(output, "")
		_, _ = fmt.Fprintln(output, "Custom Prompts:")
		for _, p := range customPrompts {
			_, _ = fmt.Fprintf(output, "  %-10s %s\n", p.Name, truncateString(p.Desc, maxShortLen))
			_, _ = fmt.Fprintf(output, "             %s\n", p.Path)
		}
	}

	_, _ = fmt.Fprintln(output, "")
	_, _ = fmt.Fprintln(output, "Use 'ralph run <prompt-name>' to execute a prompt.")

	return nil
}

func discoverCustomPrompts(cfg *config.Config) []promptInfo {
	var prompts []promptInfo

	// Read prompts directory
	entries, err := os.ReadDir(cfg.PromptsDir)
	if err != nil {
		return prompts // return empty if directory doesn't exist
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) < 4 || !strings.HasSuffix(name, ".md") {
			continue
		}

		promptName := name[:len(name)-3]
		promptPath := cfg.PromptsDir + "/" + name

		// Extract description from file
		desc := extractDescription(promptPath)

		prompts = append(prompts, promptInfo{
			Name: promptName,
			Desc: desc,
			Path: "./prompts/" + name,
		})
	}

	return prompts
}

func extractDescription(path string) string {
	content, err := os.ReadFile(path) // #nosec G304 -- path is from discovered prompts
	if err != nil {
		return "(cannot read file)"
	}

	// Parse frontmatter to get clean body
	_, body, err := prompt.ParseFrontMatter(string(content))
	if err != nil {
		return "(invalid frontmatter)"
	}

	// Get first non-empty, non-heading line
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if len(trimmed) > 0 && trimmed[0] == '#' {
			continue
		}

		return trimmed
	}

	return "(no description)"
}
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Add ellipsis
	return s[:maxLen-3] + "..."
}

func runPromptsShow(output io.Writer, cfg *config.Config, promptName string) error {
	// Try built-in prompts first
	switch promptName {
	case "build":
		_, _ = fmt.Fprint(output, prompt.BuildPrompt(cfg))

		return nil
	case "plan":
		// Plan prompt needs a scope - use empty string for display
		_, _ = fmt.Fprint(output, prompt.PlanPrompt(cfg, ""))

		return nil
	}

	// Try custom prompt file
	promptPath := cfg.PromptsDir + "/" + promptName + ".md"
	foundPath := findFileUpwards(promptPath)
	if foundPath == "" {
		return fmt.Errorf("prompt %q not found", promptName)
	}

	content, err := os.ReadFile(foundPath) // #nosec G304 -- path is from findFileUpwards
	if err != nil {
		return fmt.Errorf("failed to read prompt file %q: %w", foundPath, err)
	}

	// Parse and strip frontmatter
	_, body, err := prompt.ParseFrontMatter(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse front matter in %q: %w", foundPath, err)
	}

	_, _ = fmt.Fprint(output, body)

	return nil
}

// findFileUpwards searches for a file from the current directory upwards.
func findFileUpwards(path string) string {
	// Check if file exists at the given path
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Search upwards from current directory
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		fullPath := dir + "/" + path
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return ""
}
