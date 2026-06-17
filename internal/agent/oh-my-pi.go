package agent

import (
	"io"
)

// OmpAgent implements the Agent interface for the oh-my-pi (omp) CLI.
type OmpAgent struct {
	Model     string
	AgentMode string
	Env       []string
}

// Execute runs omp with the given prompt.
// omp CLI uses: omp launch --print [--model <model>] <prompt>.
func (a *OmpAgent) Execute(prompt string, output io.Writer) (string, error) {
	args := []string{"--print", "--no-title"}
	if a.Model != "" {
		args = append(args, "--model", a.Model)
	}
	args = append(args, prompt)

	return executeAgentCommand("omp", args, a.Env, output, "omp")
}

// Name returns the name of the agent.
func (a *OmpAgent) Name() string {
	return "omp"
}

// IsAvailable checks if omp is available in PATH.
func (a *OmpAgent) IsAvailable() bool {
	return isAgentAvailable("omp")
}
