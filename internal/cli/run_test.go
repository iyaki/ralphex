package cli_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/iyaki/ralphex/internal/cli"
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
