package agent

import (
	"io"
	"os"
	"strings"
)

// CopilotAgent implements the Agent interface for the GitHub Copilot CLI.
type CopilotAgent struct {
	Model     string
	AgentMode string
	Env       []string
}

// Execute runs copilot with the given prompt.
// Copilot CLI uses: copilot -p <prompt> [--model <model>] [--sandbox <enable|disable>]
// [--allow-all|--yolo] [--agent <name>] [--continue|--resume <session-id>]
// Configuration via environment variables:
//   - COPILOT_SANDBOX: sandbox policy (default: enable)
//   - COPILOT_ALLOW_ALL: set to "true" to enable all permissions (default: false)
//   - COPILOT_AGENT: custom agent name (e.g., explore, task, research)
//   - COPILOT_RESUME: set to "true" to resume last session (default: false)
func (a *CopilotAgent) Execute(prompt string, output io.Writer) (string, error) {
	args := []string{"-p", prompt}

	// Model selection (if specified, otherwise uses Copilot's auto-selection)
	if a.Model != "" {
		args = append(args, "--model", a.Model)
	}

	// Sandbox policy (default: enable, configurable via COPILOT_SANDBOX)
	sandbox := getEnvOrDefault("COPILOT_SANDBOX", "enable")
	args = append(args, "--sandbox", sandbox)

	// Allow all permissions (default: false, configurable via COPILOT_ALLOW_ALL)
	allowAll := getEnvOrDefault("COPILOT_ALLOW_ALL", "false")
	if strings.ToLower(allowAll) == "true" {
		// Use --allow-all flag for enabling all permissions
		args = append(args, "--allow-all")
	}

	// Custom agent selection (configurable via COPILOT_AGENT or AgentMode field)
	customAgent := os.Getenv("COPILOT_AGENT")
	if customAgent == "" {
		customAgent = a.AgentMode
	}
	if customAgent != "" {
		args = append(args, "--agent", customAgent)
	}

	// Session resumption (configurable via COPILOT_RESUME)
	resume := getEnvOrDefault("COPILOT_RESUME", "false")
	if strings.ToLower(resume) == "true" {
		args = append(args, "--continue")
	}

	return executeAgentCommand("copilot", args, a.Env, output, "copilot")
}

// Name returns the name of the agent.
func (a *CopilotAgent) Name() string {
	return "copilot"
}

// IsAvailable checks if copilot is available in PATH.
func (a *CopilotAgent) IsAvailable() bool {
	return isAgentAvailable("copilot")
}