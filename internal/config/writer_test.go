package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/iyaki/ralphex/internal/config"
)

func createTestConfig() *config.Config {
	return &config.Config{
		MaxIterations:          50,
		SpecsDir:               "custom-specs",
		SpecsIndexFile:         "INDEX.md",
		ImplementationPlanName: "PLAN.md",
		LogFile:                "custom.log",
		LogTruncate:            true,
		CustomPrompt:           "You are a helpful assistant.",
		PromptsDir:             "/tmp/prompts",
		AgentName:              "claude",
		Model:                  "claude-3-opus",
		AgentMode:              "planner",
	}
}

func verifyCoreSettings(t *testing.T, expected, actual *config.Config) {
	t.Helper()
	if actual.MaxIterations != expected.MaxIterations {
		t.Errorf("Expected MaxIterations %d, got %d", expected.MaxIterations, actual.MaxIterations)
	}
	if actual.SpecsDir != expected.SpecsDir {
		t.Errorf("Expected SpecsDir %s, got %s", expected.SpecsDir, actual.SpecsDir)
	}
	if actual.SpecsIndexFile != expected.SpecsIndexFile {
		t.Errorf("Expected SpecsIndexFile %s, got %s", expected.SpecsIndexFile, actual.SpecsIndexFile)
	}
	if actual.ImplementationPlanName != expected.ImplementationPlanName {
		t.Errorf("Expected ImplementationPlanName %s, got %s", expected.ImplementationPlanName, actual.ImplementationPlanName)
	}
}

func verifyLogSettings(t *testing.T, expected, actual *config.Config) {
	t.Helper()
	if actual.LogFile != expected.LogFile {
		t.Errorf("Expected LogFile %s, got %s", expected.LogFile, actual.LogFile)
	}
	if actual.LogTruncate != expected.LogTruncate {
		t.Errorf("Expected LogTruncate %v, got %v", expected.LogTruncate, actual.LogTruncate)
	}
}

func verifyAgentSettings(t *testing.T, expected, actual *config.Config) {
	t.Helper()
	if actual.CustomPrompt != expected.CustomPrompt {
		t.Errorf("Expected CustomPrompt %s, got %s", expected.CustomPrompt, actual.CustomPrompt)
	}
	if actual.PromptsDir != expected.PromptsDir {
		t.Errorf("Expected PromptsDir %s, got %s", expected.PromptsDir, actual.PromptsDir)
	}
	if actual.AgentName != expected.AgentName {
		t.Errorf("Expected AgentName %s, got %s", expected.AgentName, actual.AgentName)
	}
	if actual.Model != expected.Model {
		t.Errorf("Expected Model %s, got %s", expected.Model, actual.Model)
	}
	if actual.AgentMode != expected.AgentMode {
		t.Errorf("Expected AgentMode %s, got %s", expected.AgentMode, actual.AgentMode)
	}
}

func TestWriteConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "ralph-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Define a sample config
	cfg := createTestConfig()

	// Define the output path
	outputPath := filepath.Join(tmpDir, "ralph.toml")

	// Call WriteConfig
	err = config.WriteConfig(outputPath, cfg)
	if err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", outputPath)
	}

	// Read the file back and verify content
	var loadedCfg config.Config
	if _, err := toml.DecodeFile(outputPath, &loadedCfg); err != nil {
		t.Fatalf("Failed to decode generated config: %v", err)
	}

	// Verify fields
	verifyCoreSettings(t, cfg, &loadedCfg)
	verifyLogSettings(t, cfg, &loadedCfg)
	verifyAgentSettings(t, cfg, &loadedCfg)
}

func TestWriteConfig_Atomic(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "ralph-atomic-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	outputPath := filepath.Join(tmpDir, "ralph.toml")

	// Create an initial file
	initialContent := []byte("invalid-toml")
	const filePerm = 0644
	if err := os.WriteFile(outputPath, initialContent, filePerm); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	cfg := &config.Config{
		MaxIterations: 10,
	}

	// Overwrite it
	if err := config.WriteConfig(outputPath, cfg); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Verify it's valid TOML now
	var loadedCfg config.Config
	if _, err := toml.DecodeFile(outputPath, &loadedCfg); err != nil {
		t.Fatalf("Failed to decode overwritten config: %v", err)
	}

	if loadedCfg.MaxIterations != 10 {
		t.Errorf("Expected MaxIterations 10, got %d", loadedCfg.MaxIterations)
	}
}

func TestWriteConfig_SuccessAndVerify(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.toml")
	cfg := &config.Config{
		AgentName:     "claude",
		Model:         "sonnet",
		MaxIterations: 75,
		LogTruncate:   true,
	}

	err := config.WriteConfig(path, cfg)
	if err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	// Read back and verify
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "claude") {
		t.Error("expected agent in output")
	}
	if !strings.Contains(contentStr, "max-iterations = 75") {
		t.Error("expected max-iterations in output")
	}
}

func TestWriteConfig_EmptyStringsShouldBeOmitted(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.toml")
	cfg := &config.Config{
		AgentName:              "opencode",
		Model:                  "gpt-4",
		AgentMode:              "reviewer",
		MaxIterations:          25,
		SpecsDir:               "specs",
		SpecsIndexFile:         "README.md",
		ImplementationPlanName: "IMPLEMENTATION_PLAN.md",
		PromptsDir:             ".ralph/prompts",
		LogFile:                "ralph.log",
		LogTruncate:            false,
	}

	err := config.WriteConfig(path, cfg)
	if err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	contentStr := string(content)
	t.Logf("Generated TOML:\n%s", contentStr)

	// Should NOT contain empty string values
	if strings.Contains(contentStr, `config-file = ""`) {
		t.Error("Config file should not write empty config-file field")
	}
	if strings.Contains(contentStr, `prompt-file = ""`) {
		t.Error("Config file should not write empty prompt-file field")
	}
	if strings.Contains(contentStr, `custom-prompt = ""`) {
		t.Error("Config file should not write empty custom-prompt field")
	}
}

func TestWriteConfig_CannotCreateTempFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.toml")
	cfg := &config.Config{AgentName: "test"}

	// Make directory read-only to force temp file creation to fail
	oldMode := uint32(0755)
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, os.FileMode(oldMode)); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, os.FileMode(0755)) })

	// Try to write with a non-writable directory
	// Create a directory that will fail temp file creation
	badDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(badDir, 0500); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	badPath := filepath.Join(badDir, "test.toml")

	err := config.WriteConfig(badPath, cfg)
	if err == nil {
		t.Fatal("expected error when temp file creation fails")
	}
}
