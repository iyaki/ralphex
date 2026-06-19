package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iyaki/ralphex/internal/cli"
)

func TestNewPromptsCommandBasicProperties(t *testing.T) {
	cmd := cli.NewPromptsCommand()
	if cmd.Use != "prompts" {
		t.Fatalf("unexpected use string: %q", cmd.Use)
	}
	if cmd.Short != "List and view available prompts" {
		t.Fatalf("unexpected short description: %q", cmd.Short)
	}

	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help should execute successfully: %v", err)
	}
}

func TestNewPromptsListCommandBasicProperties(t *testing.T) {
	cmd := cli.NewPromptsListCommand()
	if cmd.Use != "list" {
		t.Fatalf("unexpected use string: %q", cmd.Use)
	}
	if cmd.Short != "List available prompts" {
		t.Fatalf("unexpected short description: %q", cmd.Short)
	}

	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help should execute successfully: %v", err)
	}
}

func TestNewPromptsShowCommandBasicProperties(t *testing.T) {
	cmd := cli.NewPromptsShowCommand()
	if cmd.Use != "show <name>" {
		t.Fatalf("unexpected use string: %q", cmd.Use)
	}
	if cmd.Short != "Show full prompt content" {
		t.Fatalf("unexpected short description: %q", cmd.Short)
	}

	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help should execute successfully: %v", err)
	}
}

func TestPromptsListShowsBuiltInPrompts(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cmd := cli.NewPromptsListCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected execute success, got: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Built-in Prompts:") {
		t.Errorf("expected output to contain 'Built-in Prompts:', got %q", output)
	}
	if !strings.Contains(output, "build") {
		t.Errorf("expected output to mention 'build' prompt, got %q", output)
	}
	if !strings.Contains(output, "plan") {
		t.Errorf("expected output to mention 'plan' prompt, got %q", output)
	}
	if !strings.Contains(output, "Use 'ralph run <prompt-name>' to execute a prompt.") {
		t.Errorf("expected usage hint, got %q", output)
	}
}

func TestPromptsListShowsCustomPrompts(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create a custom prompts directory in HOME/.ralph
	promptsDir := filepath.Join(homeDir, ".ralph")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	customPrompt := filepath.Join(promptsDir, "review.md")
	content := `# Code Review Prompt

This is a custom code review prompt.
It checks for security issues and performance.
`
	if err := os.WriteFile(customPrompt, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write custom prompt: %v", err)
	}

	cmd := cli.NewPromptsListCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected execute success, got: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Custom Prompts:") {
		t.Errorf("expected output to contain 'Custom Prompts:', got %q", output)
	}
	if !strings.Contains(output, "review") {
		t.Errorf("expected output to mention 'review' prompt, got %q", output)
	}
}
func TestPromptsShowBuild(t *testing.T) {
	testPromptsShow(t, "build", "Agent Instructions (Build Mode)")
}

func TestPromptsShowPlan(t *testing.T) {
	testPromptsShow(t, "plan", "Agent Instructions (Planning Mode)")
}

func testPromptsShow(t *testing.T, promptName, expectedHeader string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cmd := cli.NewPromptsShowCommand()
	cmd.SetArgs([]string{promptName})
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected execute success, got: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, expectedHeader) {
		t.Errorf("expected output to contain %q, got %q", expectedHeader, output)
	}
	if !strings.Contains(output, "Study `specs/*`") {
		t.Errorf("expected output to mention studying specs, got %q", output)
	}
	if !strings.Contains(output, "IMPLEMENTATION_PLAN.md") {
		t.Errorf("expected output to mention implementation plan, got %q", output)
	}
}


func TestPromptsShowNonExistent(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cmd := cli.NewPromptsShowCommand()
	cmd.SetArgs([]string{"nonexistent"})
	var out bytes.Buffer
	cmd.SetOut(&out)

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent prompt, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

func TestPromptsShowCustomPrompt(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create a custom prompts directory in HOME/.ralph
	promptsDir := filepath.Join(homeDir, ".ralph")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	customPrompt := filepath.Join(promptsDir, "custom.md")
	content := `# Custom Prompt

This is a custom prompt for testing.
`
	if err := os.WriteFile(customPrompt, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write custom prompt: %v", err)
	}

	cmd := cli.NewPromptsShowCommand()
	cmd.SetArgs([]string{"custom"})
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected execute success, got: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Custom Prompt") {
		t.Errorf("expected output to contain custom prompt title, got %q", output)
	}
	if !strings.Contains(output, "This is a custom prompt for testing.") {
		t.Errorf("expected output to contain custom prompt body, got %q", output)
	}
}

func TestPromptsShowCustomPromptWithFrontmatter(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Create a custom prompts directory in HOME/.ralph
	promptsDir := filepath.Join(homeDir, ".ralph")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	customPrompt := filepath.Join(promptsDir, "with-fm.md")
	content := `---
model: default
agent-mode: task
---
# Prompt with Frontmatter

This content should be displayed without frontmatter.
`
	if err := os.WriteFile(customPrompt, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write custom prompt: %v", err)
	}

	cmd := cli.NewPromptsShowCommand()
	cmd.SetArgs([]string{"with-fm"})
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected execute success, got: %v", err)
	}

	output := out.String()
	// Frontmatter should be stripped
	if strings.Contains(output, "model: default") {
		t.Errorf("expected frontmatter to be stripped, but found it in output: %q", output)
	}
	if strings.Contains(output, "agent-mode: task") {
		t.Errorf("expected frontmatter to be stripped, but found it in output: %q", output)
	}
	if !strings.Contains(output, "Prompt with Frontmatter") {
		t.Errorf("expected output to contain title, got %q", output)
	}
	if !strings.Contains(output, "This content should be displayed without frontmatter.") {
		t.Errorf("expected output to contain body, got %q", output)
	}
}
