package cli_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/iyaki/ralphex/internal/cli"
	"github.com/iyaki/ralphex/internal/config"
)

func TestNewRunCommandBasicProperties(t *testing.T) {
	cmd := cli.NewRunCommand()
	if !strings.Contains(cmd.Use, "run") {
		t.Fatalf("unexpected use string: %q", cmd.Use)
	}
	if cmd.Flags().Lookup("max-iterations") == nil {
		t.Fatal("expected max-iterations flag to exist")
	}

	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help should execute successfully: %v", err)
	}
}

func TestRunCommandExecuteDebugHappyPath(t *testing.T) {
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

	t.Setenv("HOME", t.TempDir())
	t.Setenv("DEBUG", "1")

	binDir := t.TempDir()
	writeExecutable(t, binDir, "opencode", "#!/bin/sh\necho \"ok\"\n")
	t.Setenv("PATH", binDir)

	cmd := cli.NewRunCommand()
	cmd.SetArgs([]string{"build"})

	// Capture output
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected execute success in debug mode, got: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "[build]") {
		t.Errorf("expected output to contain [build], got %q", output)
	}
}

func TestRunCommandExecuteInitAsPrompt(t *testing.T) {
	// This tests that `ralph run init` treats "init" as a prompt name, not the init command
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

	t.Setenv("HOME", t.TempDir())
	t.Setenv("DEBUG", "1")

	binDir := t.TempDir()
	writeExecutable(t, binDir, "opencode", "#!/bin/sh\necho \"ok\"\n")
	t.Setenv("PATH", binDir)

	cmd := cli.NewRunCommand()
	cmd.SetArgs([]string{"init"}) // "init" as prompt name

	// Capture output
	var out bytes.Buffer
	cmd.SetOut(&out)

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing 'init' prompt")
	}
	if !strings.Contains(err.Error(), "prompt file not found for 'init'") {
		t.Fatalf("expected prompt not found error, got: %v", err)
	}
}

func TestReadBoolFlagOverride(t *testing.T) {
	cmd := cli.NewRunCommand()
	cmd.Flags().Bool("test-flag", false, "")

	// Flag not changed
	override, err := cli.ReadBoolFlagOverride(cmd, "test-flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if override.Changed {
		t.Fatal("expected Changed to be false")
	}

	// Flag changed to true
	cmd.SetArgs([]string{"--test-flag=true"})
	if err := cmd.ParseFlags([]string{"--test-flag=true"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	override, err = cli.ReadBoolFlagOverride(cmd, "test-flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !override.Changed || !override.Value {
		t.Fatal("expected Changed=true and Value=true")
	}
}

func TestReadEnvFlagOverrides(t *testing.T) {
	cmd := cli.NewRunCommand()

	// No env flags
	overrides, err := cli.ReadEnvFlagOverrides(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overrides != nil {
		t.Fatal("expected nil overrides")
	}

	// With env flags
	cmd.SetArgs([]string{"--env", "KEY1=value1", "--env", "KEY2=value2"})
	if err := cmd.ParseFlags([]string{"--env", "KEY1=value1", "--env", "KEY2=value2"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	overrides, err = cli.ReadEnvFlagOverrides(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overrides["KEY1"] != "value1" || overrides["KEY2"] != "value2" {
		t.Fatalf("expected KEY1=value1, KEY2=value2, got %v", overrides)
	}
}

func TestReadEnvFlagOverridesInvalidEntry(t *testing.T) {
	cmd := cli.NewRunCommand()
	cmd.SetArgs([]string{"--env", "invalid-no-value"})
	if err := cmd.ParseFlags([]string{"--env", "invalid-no-value"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	_, err := cli.ReadEnvFlagOverridesForTest(cmd)
	if err == nil {
		t.Fatal("expected error for invalid env entry")
	}
	if !strings.Contains(err.Error(), "expected KEY=VALUE") {
		t.Fatalf("expected KEY=VALUE error, got %v", err)
	}
}

func TestReadEnvFlagOverridesInvalidKey(t *testing.T) {
	cmd := cli.NewRunCommand()
	cmd.SetArgs([]string{"--env", "invalid-key=value"})
	if err := cmd.ParseFlags([]string{"--env", "invalid-key=value"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	_, err := cli.ReadEnvFlagOverridesForTest(cmd)
	if err == nil {
		t.Fatal("expected error for invalid env key")
	}
	if !strings.Contains(err.Error(), "invalid --env key") {
		t.Fatalf("expected invalid key error, got %v", err)
	}
}

func TestRunLoopReachesMaxIterations(t *testing.T) {
	tmp := t.TempDir()
	binDir := t.TempDir()
	writeExecutable(t, binDir, "opencode", "#!/bin/sh\necho \"no completion signal\"\n")
	t.Setenv("PATH", binDir)
	t.Setenv("HOME", tmp)

	cfg := &config.Config{
		AgentName:     "opencode",
		Model:         "test-model",
		MaxIterations: 2,
		Env:           map[string]string{},
	}

	var out bytes.Buffer
	err := cli.RunLoop(cfg, "test-prompt", "test", &out)
	if err == nil || !strings.Contains(err.Error(), "max iterations reached") {
		t.Fatalf("expected max iterations error, got %v", err)
	}
}

func TestRunLoopDetectsCompletionSignal(t *testing.T) {
	tmp := t.TempDir()
	binDir := t.TempDir()
	writeExecutable(t, binDir, "opencode", "#!/bin/sh\necho \"<promise>COMPLETE</promise>\"\n")
	t.Setenv("PATH", binDir)
	t.Setenv("HOME", tmp)

	cfg := &config.Config{
		AgentName:     "opencode",
		Model:         "test-model",
		MaxIterations: 5,
		Env:           map[string]string{},
	}

	var out bytes.Buffer
	err := cli.RunLoop(cfg, "test-prompt", "test", &out)
	if err != nil {
		t.Fatalf("expected success with completion signal, got %v", err)
	}
}

func TestRunLoopAgentNotAvailable(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("PATH", tmp) // No agents available

	cfg := &config.Config{
		AgentName:     "opencode",
		Model:         "test-model",
		MaxIterations: 1,
		Env:           map[string]string{},
	}

	var out bytes.Buffer
	err := cli.RunLoop(cfg, "test-prompt", "test", &out)
	// Should complete but warn about agent not found
	output := out.String()
	if !strings.Contains(output, "Warning: opencode agent not found") {
		t.Fatalf("expected agent not found warning, got %q", output)
	}
	_ = err // Error expected due to max iterations
}

func TestHasCompletionSignal(t *testing.T) {
	tests := []struct {
		name     string
		result   string
		expected bool
	}{
		{"contains signal", "some text\n<promise>COMPLETE</promise>\nmore", true},
		{"no signal", "some text\nmore text", false},
		{"signal with spaces", "  <promise>COMPLETE</promise>  ", true},
		{"partial signal", "<promise>COMPLETE", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cli.HasCompletionSignal(tc.result, "<promise>COMPLETE</promise>")
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}
