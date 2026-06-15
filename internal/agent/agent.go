// Package agent provides integrations for supported AI agent CLIs.
package agent

import (
	"fmt"
	"io"
)

// Agent represents an AI agent CLI that can execute prompts.
type Agent interface {
	// Execute runs the agent with the given prompt and returns the output.
	Execute(prompt string, output io.Writer) (string, error)

	// Name returns the name of the agent.
	Name() string

	// IsAvailable checks if the agent CLI is available on the system.
	IsAvailable() bool
}

// GetAgent returns the appropriate agent based on configuration.
func GetAgent(agentName, model, agentMode string, env []string) (Agent, error) {
	effectiveEnv := cloneStringSlice(env)

	switch agentName {
	case "omp":
		return &OmpAgent{Model: model, AgentMode: agentMode, Env: effectiveEnv}, nil
	case "claude":
		return &ClaudeAgent{Model: model, AgentMode: agentMode, Env: effectiveEnv}, nil
	case "cursor":
		return &CursorAgent{Model: model, AgentMode: agentMode, Env: effectiveEnv}, nil
	case "opencode":
		return &OpencodeAgent{Model: model, AgentMode: agentMode, Env: effectiveEnv}, nil
	default:
		return nil, fmt.Errorf("unknown agent %q (supported: omp, opencode, claude, cursor)", agentName)
	}
}
