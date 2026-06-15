package cli

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/iyaki/ralphex/internal/agent"
	"github.com/iyaki/ralphex/internal/config"
	"github.com/iyaki/ralphex/internal/logger"
	"github.com/iyaki/ralphex/internal/prompt"
)

var envFlagKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// NewRunCommand creates the run command for Ralphex.
func NewRunCommand() *cobra.Command {
	var cfg config.Config

	cmd := &cobra.Command{
		Use:   "run [prompt] [scope]",
		Short: "Run a prompt loop",
		Long:  `Run a prompt loop with the specified prompt and scope.`,
		Args:  cobra.MaximumNArgs(maxPositionalArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommandLogic(cmd, args, &cfg)
		},
	}

	setupSharedFlags(cmd, &cfg)

	return cmd
}

func runCommandLogic(cmd *cobra.Command, args []string, cfg *config.Config) error {
	promptName, scope := parsePositionalArgs(args)

	noLogOverride, err := readBoolFlagOverride(cmd, "no-log")
	if err != nil {
		return err
	}

	logTruncateOverride, err := readBoolFlagOverride(cmd, "log-truncate")
	if err != nil {
		return err
	}

	envFlagOverrides, err := readEnvFlagOverrides(cmd)
	if err != nil {
		return err
	}

	// Load configuration with proper precedence
	if err := cfg.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	applyBoolFlagOverrides(cfg, noLogOverride, logTruncateOverride)
	applyEnvFlagOverrides(cfg, envFlagOverrides)

	// Initialize logger
	appLogger, err := logger.NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() {
		_ = appLogger.Close()
	}()

	// Write to both logger and stdout
	writers := []io.Writer{cmd.OutOrStdout()}
	if appLogger.Enabled() {
		writers = append(writers, appLogger.File())
	}
	output := io.MultiWriter(writers...)

	// Get the prompt
	promptText, fmOverride, err := prompt.GetPrompt(cfg, promptName, scope, output)
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	// Apply configuration precedence
	applyEffectiveSettings(cfg, cmd, fmOverride, promptName)

	// Run the main loop
	return RunLoop(cfg, promptText, promptName, output)
}

type boolFlagOverride struct {
	changed bool
	value   bool
}

func readBoolFlagOverride(cmd *cobra.Command, flagName string) (boolFlagOverride, error) {
	if !cmd.Flags().Changed(flagName) {
		return boolFlagOverride{}, nil
	}

	value, err := cmd.Flags().GetBool(flagName)
	if err != nil {
		return boolFlagOverride{}, fmt.Errorf("failed to read --%s flag: %w", flagName, err)
	}

	return boolFlagOverride{changed: true, value: value}, nil
}

func applyBoolFlagOverrides(cfg *config.Config, noLogOverride, logTruncateOverride boolFlagOverride) {
	if noLogOverride.changed {
		cfg.NoLog = noLogOverride.value
	}

	if logTruncateOverride.changed {
		cfg.LogTruncate = logTruncateOverride.value
	}
}

func readEnvFlagOverrides(cmd *cobra.Command) (map[string]string, error) {
	rawEntries, err := cmd.Flags().GetStringArray("env")
	if err != nil {
		return nil, fmt.Errorf("failed to read --env flag: %w", err)
	}

	if len(rawEntries) == 0 {
		return nil, nil
	}

	overrides := make(map[string]string, len(rawEntries))
	for i, entry := range rawEntries {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --env entry #%d: expected KEY=VALUE", i+1)
		}
		if !envFlagKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid --env key %q at entry #%d", key, i+1)
		}

		overrides[key] = value
	}

	return overrides, nil
}

func applyEnvFlagOverrides(cfg *config.Config, overrides map[string]string) {
	if len(overrides) == 0 {
		return
	}

	if cfg.Env == nil {
		cfg.Env = make(map[string]string, len(overrides))
	}

	for key, value := range overrides {
		cfg.Env[key] = value
	}
}

func applyEffectiveSettings(
	cfg *config.Config,
	cmd *cobra.Command,
	fmOverride *config.PromptConfigOverride,
	promptName string,
) {
	applyModelSettings(cfg, cmd, fmOverride, promptName)
	applyAgentModeSettings(cfg, cmd, fmOverride, promptName)
}

func applyModelSettings(
	cfg *config.Config,
	cmd *cobra.Command,
	fmOverride *config.PromptConfigOverride,
	promptName string,
) {
	// Model Precedence: Flag > Env > Front Matter > Config Override > Global Config
	if cmd.Flags().Changed("model") || os.Getenv("RALPH_MODEL") != "" {
		return
	}

	if fmOverride != nil && fmOverride.Model != "" {
		cfg.Model = fmOverride.Model

		return
	}

	if promptOverride, ok := cfg.PromptOverrides[promptName]; ok && promptOverride.Model != "" {
		cfg.Model = promptOverride.Model
	}
}

func applyAgentModeSettings(
	cfg *config.Config,
	cmd *cobra.Command,
	fmOverride *config.PromptConfigOverride,
	promptName string,
) {
	// Agent Mode Precedence: Flag > Env > Front Matter > Config Override > Global Config
	if cmd.Flags().Changed("agent-mode") || os.Getenv("RALPH_AGENT_MODE") != "" {
		return
	}

	if fmOverride != nil && fmOverride.AgentMode != "" {
		cfg.AgentMode = fmOverride.AgentMode

		return
	}

	if promptOverride, ok := cfg.PromptOverrides[promptName]; ok && promptOverride.AgentMode != "" {
		cfg.AgentMode = promptOverride.AgentMode
	}
}

// RunLoop executes the main Ralphex iteration loop.
func RunLoop(cfg *config.Config, promptText, promptName string, output io.Writer) error {
	completionSignal := "<promise>COMPLETE</promise>"
	writef := func(format string, args ...any) {
		_, _ = fmt.Fprintf(output, format, args...)
	}
	writeln := func(args ...any) {
		_, _ = fmt.Fprintln(output, args...)
	}

	// Replace placeholders in prompt
	promptText = strings.ReplaceAll(promptText, "<COMPLETION_SIGNAL>", completionSignal)

	effectiveEnv, err := agent.BuildEffectiveEnv(cfg.Env)
	if err != nil {
		return fmt.Errorf("failed to build agent environment: %w", err)
	}

	// Get the configured agent
	agentInstance, err := agent.GetAgent(cfg.AgentName, cfg.Model, cfg.AgentMode, effectiveEnv)
	if err != nil {
		return err
	}

	// Check if agent is available
	if !agentInstance.IsAvailable() {
		writef("Warning: %s agent not found in PATH, will continue anyway...\n", agentInstance.Name())
	}

	writef("Starting Ralphex - Max iterations: %d\n", cfg.MaxIterations)
	writef("Using agent: %s\n", agentInstance.Name())

	for i := 1; i <= cfg.MaxIterations; i++ {
		writef("\n")
		writef("===============================================================\n")
		writef(" [%s] Iteration %d of %d (%s)\n", promptName, i, cfg.MaxIterations, time.Now().Format(time.RFC3339))
		writef("===============================================================\n")

		// Check if DEBUG mode (for testing)
		if os.Getenv("DEBUG") != "" {
			writeln(promptText)
			writef("\nAll planned tasks completed!\n")
			writef("Completed at iteration %d of %d\n", i, cfg.MaxIterations)

			return nil
		}

		// Execute the agent
		result, err := agentInstance.Execute(promptText, output)
		if err != nil {
			// Non-fatal error, continue to next iteration
			writef("Command execution warning: %v\n", err)
		}

		// Check for completion signal
		if hasCompletionSignal(result, completionSignal) {
			writef("\nAll planned tasks completed!\n")
			writef("Completed at iteration %d of %d\n", i, cfg.MaxIterations)

			return nil
		}

		writef("Iteration %d complete. Continuing...\n", i)
	}

	writef("\nReached max iterations (%d) without completing all planned tasks.\n", cfg.MaxIterations)

	return fmt.Errorf("max iterations reached")
}

func hasCompletionSignal(result, completionSignal string) bool {
	for line := range strings.SplitSeq(result, "\n") {
		if strings.TrimSpace(line) == completionSignal {
			return true
		}
	}

	return false
}

func parsePositionalArgs(args []string) (string, string) {
	promptName := "build"
	scope := "Whole system"

	if len(args) > 0 {
		promptName = args[0]
	}
	if len(args) > 1 {
		scope = args[1]
	}

	return promptName, scope
}

func setupSharedFlags(cmd *cobra.Command, cfg *config.Config) {
	flags := cmd.Flags()
	flags.StringVarP(&cfg.ConfigFile, "config", "c", "", "Config file to source")
	flags.IntVarP(&cfg.MaxIterations, "max-iterations", "m", 0, "Maximum iterations (default: 25)")
	flags.StringVarP(&cfg.PromptFile, "prompt-file", "p", "", "Prompt file path (use '-' to read from stdin)")
	flags.StringVarP(&cfg.SpecsDir, "specs-dir", "s", "", "Specs directory (default: specs)")
	flags.StringVarP(&cfg.SpecsIndexFile, "specs-index", "i", "", "Specs index file (default: README.md)")
	flags.BoolVar(&cfg.NoSpecsIndex, "no-specs-index", false, "Disable specs index file")
	flags.StringVarP(&cfg.ImplementationPlanName, "implementation-plan-name", "n", "", "Implementation plan file name")
	flags.StringVarP(&cfg.LogFile, "log-file", "l", "", "Log file path")
	flags.BoolVar(&cfg.NoLog, "no-log", false, "Disable logs")
	flags.BoolVar(&cfg.LogTruncate, "log-truncate", false, "Truncate log file before writing")
	flags.StringVar(&cfg.CustomPrompt, "prompt", "", "Inline custom prompt (overrides prompt files)")
	flags.StringVarP(&cfg.AgentName, "agent", "a", "", "AI agent to use: omp, opencode, claude, cursor"+
		" (default: opencode)")
	flags.StringVar(&cfg.Model, "model", "", "AI model to use (e.g., claude-sonnet-4, gpt-4)")
	flags.StringVar(&cfg.AgentMode, "agent-mode", "", "Agent mode/sub-agent to use (e.g., reviewer, planner)")
	flags.StringArray("env", nil, "Set/override an agent environment variable (KEY=VALUE). Repeatable")
}
