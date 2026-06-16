package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/iyaki/ralphex/internal/config"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()
	t.Setenv("RALPH_MAX_ITERATIONS", "")
	t.Setenv("RALPH_SPECS_DIR", "")
	t.Setenv("RALPH_SPECS_INDEX_FILE", "")
	t.Setenv("RALPH_IMPLEMENTATION_PLAN_NAME", "")
	t.Setenv("RALPH_CUSTOM_PROMPT", "")
	t.Setenv("RALPH_LOG_FILE", "")
	t.Setenv("RALPH_LOG_ENABLED", "")
	t.Setenv("RALPH_LOG_APPEND", "")
	t.Setenv("RALPH_PROMPTS_DIR", "")
	t.Setenv("RALPH_AGENT", "")
	t.Setenv("RALPH_MODEL", "")
	t.Setenv("RALPH_AGENT_MODE", "")
}

func TestLoadConfigDefaults(t *testing.T) {
	clearConfigEnv(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	newDir := t.TempDir()
	if err := os.Chdir(newDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	c := &config.Config{}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertDefaultCoreFields(t, c, home)
	assertDefaultLoggingFields(t, c)
}

func assertDefaultCoreFields(t *testing.T, c *config.Config, home string) {
	t.Helper()

	if c.MaxIterations != 25 {
		t.Fatalf("expected default max iterations 25, got %d", c.MaxIterations)
	}
	if c.SpecsDir != "specs" {
		t.Fatalf("expected default specs dir, got %q", c.SpecsDir)
	}
	if c.SpecsIndexFile != "README.md" {
		t.Fatalf("expected default specs index file, got %q", c.SpecsIndexFile)
	}
	if c.ImplementationPlanName != "IMPLEMENTATION_PLAN.md" {
		t.Fatalf("expected default implementation plan name, got %q", c.ImplementationPlanName)
	}
	if c.AgentName != "opencode" {
		t.Fatalf("expected default agent, got %q", c.AgentName)
	}
	if c.PromptsDir != filepath.Join(home, ".ralph") {
		t.Fatalf("expected default prompts dir in HOME, got %q", c.PromptsDir)
	}
}

func assertDefaultLoggingFields(t *testing.T, c *config.Config) {
	t.Helper()

	if c.LogFile != "" {
		t.Fatalf("expected default LogFile=\"\", got %q", c.LogFile)
	}
	if c.LogTruncate {
		t.Fatalf("expected default LogTruncate=false, got %v", c.LogTruncate)
	}
}

func TestLoadConfigPrecedence(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	content := `max-iterations = 7
specs-dir = "file-specs"
custom-prompt = "from-file"
agent = "cursor"
model = "file-model"
agent-mode = "file-mode"
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	t.Setenv("RALPH_MAX_ITERATIONS", "9")
	t.Setenv("RALPH_SPECS_DIR", "env-specs")
	t.Setenv("RALPH_CUSTOM_PROMPT", "from-env")
	t.Setenv("RALPH_AGENT", "claude")
	t.Setenv("RALPH_MODEL", "env-model")
	t.Setenv("RALPH_AGENT_MODE", "env-mode")

	c := &config.Config{
		ConfigFile:    configFile,
		MaxIterations: 13,
		SpecsDir:      "flag-specs",
		AgentName:     "opencode",
	}

	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.MaxIterations != 13 {
		t.Fatalf("expected flag value for max iterations, got %d", c.MaxIterations)
	}
	if c.SpecsDir != "flag-specs" {
		t.Fatalf("expected flag value for specs dir, got %q", c.SpecsDir)
	}
	if c.CustomPrompt != "from-env" {
		t.Fatalf("expected env override for custom prompt, got %q", c.CustomPrompt)
	}
	if c.AgentName != "opencode" {
		t.Fatalf("expected flag override for agent, got %q", c.AgentName)
	}
	if c.Model != "env-model" {
		t.Fatalf("expected env override for model, got %q", c.Model)
	}
	if c.AgentMode != "env-mode" {
		t.Fatalf("expected env override for agent mode, got %q", c.AgentMode)
	}
}

func TestLoadConfigPromptFileFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	if err := os.WriteFile(configFile, []byte(`prompt-file = "prompt.md"`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	c := &config.Config{ConfigFile: configFile}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.PromptFile != "prompt.md" {
		t.Fatalf("expected prompt file from config file, got %q", c.PromptFile)
	}
}

func TestLoadConfigPromptFileFlagWinsOverConfigFile(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	if err := os.WriteFile(configFile, []byte(`prompt-file = "from-config.md"`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	c := &config.Config{ConfigFile: configFile, PromptFile: "from-flag.md"}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.PromptFile != "from-flag.md" {
		t.Fatalf("expected prompt file from flag, got %q", c.PromptFile)
	}
}

func TestLoadConfigNoSpecsIndexFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	if err := os.WriteFile(configFile, []byte(`no-specs-index = true`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	c := &config.Config{ConfigFile: configFile}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !c.NoSpecsIndex {
		t.Fatalf("expected no-specs-index=true from config file")
	}
}

func TestLoadConfigNoSpecsIndexFlagWinsOverConfigFile(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	if err := os.WriteFile(configFile, []byte(`no-specs-index = false`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	c := &config.Config{ConfigFile: configFile, NoSpecsIndex: true}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !c.NoSpecsIndex {
		t.Fatalf("expected no-specs-index=true from flag")
	}
}

func TestLoadConfigRejectsConfigFileKey(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	if err := os.WriteFile(configFile, []byte(`config-file = "./other.toml"`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	c := &config.Config{ConfigFile: configFile}
	err := c.LoadConfig()
	if err == nil {
		t.Fatal("expected error for unsupported config-file key")
	}

	if !strings.Contains(err.Error(), "unsupported config key 'config-file'") {
		t.Fatalf("expected unsupported key error, got %v", err)
	}
}

func TestLoadConfigRejectsConfigFileKeyInDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := os.WriteFile(filepath.Join(dir, "ralph.toml"), []byte(`config-file = "./other.toml"`), 0644); err != nil {
		t.Fatalf("failed to write default config: %v", err)
	}

	c := &config.Config{}
	err = c.LoadConfig()
	if err == nil {
		t.Fatal("expected error for unsupported config-file key in default config")
	}

	if !strings.Contains(err.Error(), "unsupported config key 'config-file'") {
		t.Fatalf("expected unsupported key error, got %v", err)
	}
}

func TestLoadConfigMissingConfigFile(t *testing.T) {
	c := &config.Config{ConfigFile: filepath.Join(t.TempDir(), "does-not-exist.toml")}
	if err := c.LoadConfig(); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoadConfigDefaultFileDiscovery(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	t.Setenv("HOME", t.TempDir())

	content := `max-iterations = 44
specs-dir = "from-default-file"
`
	if err := os.WriteFile(filepath.Join(dir, "ralph.toml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write default config: %v", err)
	}

	c := &config.Config{}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.MaxIterations != 44 {
		t.Fatalf("expected max iterations from default file, got %d", c.MaxIterations)
	}
	if c.SpecsDir != "from-default-file" {
		t.Fatalf("expected specs dir from default file, got %q", c.SpecsDir)
	}
}

func TestLoadConfigEnvironmentValues(t *testing.T) {
	clearConfigEnv(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RALPH_MAX_ITERATIONS", "31")
	t.Setenv("RALPH_SPECS_DIR", "env-specs")
	t.Setenv("RALPH_SPECS_INDEX_FILE", "index.md")
	t.Setenv("RALPH_IMPLEMENTATION_PLAN_NAME", "IMPL.md")
	t.Setenv("RALPH_CUSTOM_PROMPT", "prompt-from-env")
	t.Setenv("RALPH_LOG_FILE", filepath.Join(t.TempDir(), "x.log"))
	t.Setenv("RALPH_LOG_ENABLED", "0")
	t.Setenv("RALPH_LOG_APPEND", "0")
	t.Setenv("RALPH_PROMPTS_DIR", "env-prompts")
	t.Setenv("RALPH_AGENT", "claude")
	t.Setenv("RALPH_MODEL", "m-env")
	t.Setenv("RALPH_AGENT_MODE", "planner")

	c := &config.Config{}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCoreEnvFields(t, c)
	assertPromptEnvFields(t, c)
	assertLogEnvFields(t, c)
	assertAgentEnvFields(t, c)
}

func assertCoreEnvFields(t *testing.T, c *config.Config) {
	t.Helper()
	if c.MaxIterations != 31 || c.SpecsDir != "env-specs" || c.SpecsIndexFile != "index.md" {
		t.Fatalf("expected env-derived core fields, got %+v", c)
	}
}

func assertPromptEnvFields(t *testing.T, c *config.Config) {
	t.Helper()
	if c.ImplementationPlanName != "IMPL.md" || c.CustomPrompt != "prompt-from-env" {
		t.Fatalf("expected env-derived prompt fields, got %+v", c)
	}
}

func assertLogEnvFields(t *testing.T, c *config.Config) {
	t.Helper()
	if c.LogFile == "" {
		t.Fatalf("expected LogFile from env, got empty")
	}
	if !c.LogTruncate {
		t.Fatalf("expected LogTruncate=true from env RALPH_LOG_APPEND=0, got %v", c.LogTruncate)
	}
}

func assertAgentEnvFields(t *testing.T, c *config.Config) {
	t.Helper()
	if c.PromptsDir != "env-prompts" || c.AgentName != "claude" || c.Model != "m-env" || c.AgentMode != "planner" {
		t.Fatalf("expected env-derived agent fields, got %+v", c)
	}
}

func TestLoadConfigRelativeConfigPath(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := os.WriteFile("myconfig.toml", []byte("max-iterations = 52\n"), 0644); err != nil {
		t.Fatalf("failed to write relative config: %v", err)
	}

	c := &config.Config{ConfigFile: "myconfig.toml"}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.MaxIterations != 52 {
		t.Fatalf("expected max-iterations from relative config path, got %d", c.MaxIterations)
	}
}

func TestLoadConfigIgnoresLegacyFileNames(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	t.Setenv("HOME", t.TempDir())

	if err := os.WriteFile(filepath.Join(dir, ".ralphrc.toml"), []byte("specs-dir = \"legacy\"\n"), 0644); err != nil {
		t.Fatalf("failed to write legacy config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".ralphrc"), []byte("specs-dir = \"legacy-no-ext\"\n"), 0644); err != nil {
		t.Fatalf("failed to write legacy no-extension config: %v", err)
	}

	c := &config.Config{}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.SpecsDir != "specs" {
		t.Fatalf("expected default specs-dir when only legacy config exists, got %q", c.SpecsDir)
	}
}

func TestLoadConfigPromptOverrides(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	content := `
[prompt-overrides.my-prompt]
model = "gpt-4"
agent-mode = "planner"

[prompt-overrides.another-prompt]
model = "claude-3-opus"
agent-mode = "code"
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	c := &config.Config{
		ConfigFile: configFile,
	}

	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.PromptOverrides) != 2 {
		t.Fatalf("expected 2 prompt overrides, got %d", len(c.PromptOverrides))
	}

	p1, ok := c.PromptOverrides["my-prompt"]
	if !ok {
		t.Fatal("expected override for 'my-prompt'")
	}
	if p1.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %q", p1.Model)
	}
	if p1.AgentMode != "planner" {
		t.Errorf("expected agent-mode 'planner', got %q", p1.AgentMode)
	}

	p2, ok := c.PromptOverrides["another-prompt"]
	if !ok {
		t.Fatal("expected override for 'another-prompt'")
	}
	if p2.Model != "claude-3-opus" {
		t.Errorf("expected model 'claude-3-opus', got %q", p2.Model)
	}
	if p2.AgentMode != "code" {
		t.Errorf("expected agent-mode 'code', got %q", p2.AgentMode)
	}
}

func TestLoadConfigEnvTable(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	content := `
[env]
OPENAI_API_KEY = "from-file"
HTTP_PROXY = "http://127.0.0.1:8080"
EMPTY_VALUE = ""
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	c := &config.Config{ConfigFile: configFile}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]string{
		"OPENAI_API_KEY": "from-file",
		"HTTP_PROXY":     "http://127.0.0.1:8080",
		"EMPTY_VALUE":    "",
	}

	if !reflect.DeepEqual(c.Env, expected) {
		t.Fatalf("expected env map %+v, got %+v", expected, c.Env)
	}
}

func TestLoadConfigEnvTablePreservesComplexValues(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	content := `
[env]
DATABASE_URL = "postgres://user:pass@127.0.0.1:5432/app?sslmode=disable&x=a=b"
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	c := &config.Config{ConfigFile: configFile}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := c.Env["DATABASE_URL"]; got != "postgres://user:pass@127.0.0.1:5432/app?sslmode=disable&x=a=b" {
		t.Fatalf("expected DATABASE_URL to preserve complex value, got %q", got)
	}
}

func TestLoadConfigWithoutEnvTableLeavesEnvNil(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "ralph.toml")
	if err := os.WriteFile(configFile, []byte(`model = "gpt-4"`), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	c := &config.Config{ConfigFile: configFile}
	if err := c.LoadConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Env != nil {
		t.Fatalf("expected Env to be nil when [env] is not defined, got %+v", c.Env)
	}
}
