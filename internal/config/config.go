// Package config handles loading and resolving Ralph configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

const (
	defaultMaxIterations          = 25
	defaultSpecsDir               = "specs"
	defaultSpecsIndexFile         = "README.md"
	defaultImplementationPlanName = "IMPLEMENTATION_PLAN.md"
	defaultAgentName              = "opencode"
)

type envValues struct {
	maxIterations          string
	specsDir               string
	specsIndexFile         string
	implementationPlanName string
	customPrompt           string
	logFile                string
	logAppend              string
	promptsDir             string
	agentName              string
	model                  string
	agentMode              string
}

const (
	defaultLogTruncate = false
)

// Config holds all Ralph configuration.
type Config struct {
	ConfigFile             string                          `toml:"config-file,omitempty"`
	MaxIterations          int                             `toml:"max-iterations"`
	PromptFile             string                          `toml:"prompt-file,omitempty"`
	SpecsDir               string                          `toml:"specs-dir"`
	SpecsIndexFile         string                          `toml:"specs-index-file"`
	NoSpecsIndex           bool                            `toml:"no-specs-index"`
	ImplementationPlanName string                          `toml:"implementation-plan-name"`
	LogFile                string                          `toml:"log-file,omitempty"`
	LogTruncate            bool                            `toml:"log-truncate,omitempty"`
	CustomPrompt           string                          `toml:"custom-prompt,omitempty"`
	PromptsDir             string                          `toml:"prompts-dir"`
	AgentName              string                          `toml:"agent"`
	Model                  string                          `toml:"model,omitempty"`
	AgentMode              string                          `toml:"agent-mode,omitempty"`
	Env                    map[string]string               `toml:"env,omitempty"`
	PromptOverrides        map[string]PromptConfigOverride `toml:"prompt-overrides,omitempty"`

	configLoaded bool
}

// PromptConfigOverride defines per-prompt configuration overrides.
type PromptConfigOverride struct {
	Model     string `toml:"model"`
	AgentMode string `toml:"agent-mode"`
}

// LoadConfig loads configuration with proper precedence: flags > env vars > config file > defaults.
func (c *Config) LoadConfig() error {
	configFromFile, configPath, err := c.resolveFileConfig()
	if err != nil {
		return err
	}

	if err := c.applyLocalOverlay(configFromFile, configPath); err != nil {
		return err
	}

	env := readEnv()
	c.applyConfigValues(configFromFile, env)
	c.configLoaded = true

	return nil
}

func (c *Config) applyLocalOverlay(configFromFile *Config, configPath string) error {
	if configPath == "" {
		return nil
	}

	overlayPath := filepath.Join(filepath.Dir(configPath), "ralph-local.toml")
	if _, err := os.Stat(overlayPath); err != nil {
		return nil
	}

	overlayConfig := &Config{}
	meta, err := c.loadConfigFile(overlayPath, overlayConfig)
	if err != nil {
		return fmt.Errorf("failed to load overlay config file %s: %w", overlayPath, err)
	}
	if err := validateConfigFileKeys(meta, overlayPath); err != nil {
		return err
	}

	mergeConfig(configFromFile, overlayConfig, meta)

	return nil
}

func (c *Config) resolveFileConfig() (*Config, string, error) {
	configFromFile := &Config{
		LogTruncate: defaultLogTruncate,
	}

	// Priority 1: Config file path from flag (c.ConfigFile is already set by flag parsing)
	configPath := c.ConfigFile

	// Priority 2: Config file path from environment variable
	if configPath == "" {
		configPath = os.Getenv("RALPH_CONFIG")
	}

	if configPath == "" {
		foundPath, err := loadDefaultConfig(c, configFromFile)
		if err != nil {
			return nil, "", err
		}

		return configFromFile, foundPath, nil
	}

	resolvedPath := resolveConfigPath(configPath)
	meta, err := c.loadConfigFile(resolvedPath, configFromFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load config file %s: %w", resolvedPath, err)
	}

	if err := validateConfigFileKeys(meta, resolvedPath); err != nil {
		return nil, "", err
	}

	return configFromFile, resolvedPath, nil
}

func resolveConfigPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	cwd, _ := os.Getwd()

	return filepath.Join(cwd, path)
}

func loadDefaultConfig(c *Config, target *Config) (string, error) {
	cwd, _ := os.Getwd()

	path := filepath.Join(cwd, "ralph.toml")
	if _, err := os.Stat(path); err == nil {
		meta, err := c.loadConfigFile(path, target)
		if err != nil {
			return "", fmt.Errorf("failed to load config file %s: %w", path, err)
		}
		if err := validateConfigFileKeys(meta, path); err != nil {
			return "", err
		}

		return path, nil
	}

	return "", nil
}

func validateConfigFileKeys(meta toml.MetaData, path string) error {
	if meta.IsDefined("config-file") {
		return fmt.Errorf("unsupported config key 'config-file' in %s", path)
	}

	return nil
}

func readEnv() envValues {
	return envValues{
		maxIterations:          os.Getenv("RALPH_MAX_ITERATIONS"),
		specsDir:               os.Getenv("RALPH_SPECS_DIR"),
		specsIndexFile:         os.Getenv("RALPH_SPECS_INDEX_FILE"),
		implementationPlanName: os.Getenv("RALPH_IMPLEMENTATION_PLAN_NAME"),
		customPrompt:           os.Getenv("RALPH_CUSTOM_PROMPT"),
		logFile:                os.Getenv("RALPH_LOG_FILE"),
		logAppend:              os.Getenv("RALPH_LOG_APPEND"),
		promptsDir:             os.Getenv("RALPH_PROMPTS_DIR"),
		agentName:              os.Getenv("RALPH_AGENT"),
		model:                  os.Getenv("RALPH_MODEL"),
		agentMode:              os.Getenv("RALPH_AGENT_MODE"),
	}
}

func (c *Config) applyConfigValues(fileCfg *Config, env envValues) {
	c.MaxIterations = resolveInt(c.MaxIterations, env.maxIterations, fileCfg.MaxIterations, defaultMaxIterations)
	c.PromptFile = resolveString(c.PromptFile, "", fileCfg.PromptFile, "")
	c.SpecsDir = resolveString(c.SpecsDir, env.specsDir, fileCfg.SpecsDir, defaultSpecsDir)
	c.SpecsIndexFile = resolveString(c.SpecsIndexFile, env.specsIndexFile, fileCfg.SpecsIndexFile, defaultSpecsIndexFile)
	c.NoSpecsIndex = resolveBool(c.NoSpecsIndex, fileCfg.NoSpecsIndex)
	c.ImplementationPlanName = resolveString(
		c.ImplementationPlanName,
		env.implementationPlanName,
		fileCfg.ImplementationPlanName,
		defaultImplementationPlanName,
	)
	c.CustomPrompt = resolveString(c.CustomPrompt, env.customPrompt, fileCfg.CustomPrompt, "")
	c.PromptsDir = resolveString(c.PromptsDir, env.promptsDir, fileCfg.PromptsDir, defaultPromptsDir())
	c.LogFile = resolveString(c.LogFile, env.logFile, fileCfg.LogFile, defaultLogFile())
	c.LogTruncate = resolveLogTruncate(c.LogTruncate, env.logAppend, fileCfg.LogTruncate)
	c.AgentName = resolveString(c.AgentName, env.agentName, fileCfg.AgentName, defaultAgentName)
	c.Model = resolveString(c.Model, env.model, fileCfg.Model, "")
	c.AgentMode = resolveString(c.AgentMode, env.agentMode, fileCfg.AgentMode, "")
	c.Env = cloneStringMap(fileCfg.Env)

	// Prompt overrides only come from the config file.
	if len(fileCfg.PromptOverrides) > 0 {
		c.PromptOverrides = fileCfg.PromptOverrides
	}
}

func resolveInt(flagValue int, envValue string, fileValue int, defaultValue int) int {
	if flagValue != 0 {
		return flagValue
	}

	if envValue != "" {
		if parsed, err := strconv.Atoi(envValue); err == nil {
			return parsed
		}
	}

	if fileValue != 0 {
		return fileValue
	}

	return defaultValue
}

func resolveString(flagValue, envValue, fileValue, defaultValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if envValue != "" {
		return envValue
	}

	if fileValue != "" {
		return fileValue
	}

	return defaultValue
}

func resolveBool(flagValue bool, fileValue bool) bool {
	if flagValue {
		return true
	}

	return fileValue
}

func resolveLogTruncate(flagValue bool, envValue string, fileValue bool) bool {
	if flagValue {
		return true
	}

	if envValue != "" {
		if parsed, err := strconv.ParseBool(envValue); err == nil {
			return !parsed
		}
	}

	return fileValue
}

func defaultPromptsDir() string {
	return filepath.Join(os.Getenv("HOME"), ".ralph")
}

func defaultLogFile() string {
	return ""
}

// loadConfigFile reads a TOML config file and populates the given Config.
func (c *Config) loadConfigFile(path string, target *Config) (toml.MetaData, error) {
	return toml.DecodeFile(path, target)
}

func mergeConfig(base *Config, overlay *Config, meta toml.MetaData) {
	mergeScalars(base, overlay, meta)
	mergePromptOverrides(base, overlay, meta)
	mergeEnv(base, overlay)
}

func mergeScalars(base *Config, overlay *Config, meta toml.MetaData) {
	if meta.IsDefined("max-iterations") {
		base.MaxIterations = overlay.MaxIterations
	}
	if meta.IsDefined("no-specs-index") {
		base.NoSpecsIndex = overlay.NoSpecsIndex
	}
	if meta.IsDefined("log-truncate") {
		base.LogTruncate = overlay.LogTruncate
	}

	mergeStringScalars(base, overlay, meta)
}

func mergeStringScalars(base *Config, overlay *Config, meta toml.MetaData) {
	mergeFileAndSpecScalars(base, overlay, meta)
	mergePromptAndLogScalars(base, overlay, meta)
	mergeAgentScalars(base, overlay, meta)
}

func mergeFileAndSpecScalars(base *Config, overlay *Config, meta toml.MetaData) {
	if meta.IsDefined("prompt-file") {
		base.PromptFile = overlay.PromptFile
	}
	if meta.IsDefined("specs-dir") {
		base.SpecsDir = overlay.SpecsDir
	}
	if meta.IsDefined("specs-index-file") {
		base.SpecsIndexFile = overlay.SpecsIndexFile
	}
	if meta.IsDefined("implementation-plan-name") {
		base.ImplementationPlanName = overlay.ImplementationPlanName
	}
}

func mergePromptAndLogScalars(base *Config, overlay *Config, meta toml.MetaData) {
	if meta.IsDefined("log-file") {
		base.LogFile = overlay.LogFile
	}
	if meta.IsDefined("custom-prompt") {
		base.CustomPrompt = overlay.CustomPrompt
	}
	if meta.IsDefined("prompts-dir") {
		base.PromptsDir = overlay.PromptsDir
	}
}

func mergeAgentScalars(base *Config, overlay *Config, meta toml.MetaData) {
	if meta.IsDefined("agent") {
		base.AgentName = overlay.AgentName
	}
	if meta.IsDefined("model") {
		base.Model = overlay.Model
	}
	if meta.IsDefined("agent-mode") {
		base.AgentMode = overlay.AgentMode
	}
}

func mergePromptOverrides(base *Config, overlay *Config, meta toml.MetaData) {
	if len(overlay.PromptOverrides) == 0 {
		return
	}

	if base.PromptOverrides == nil {
		base.PromptOverrides = make(map[string]PromptConfigOverride)
	}

	for k, v := range overlay.PromptOverrides {
		if baseVal, ok := base.PromptOverrides[k]; ok {
			// To correctly merge structs inside the map, we need to check if fields are defined in TOML.
			// However, toml.MetaData keys are flat.
			// We can check specific keys.
			baseKey := []string{"prompt-overrides", k}

			if meta.IsDefined(append(baseKey, "model")...) {
				baseVal.Model = v.Model
			}
			if meta.IsDefined(append(baseKey, "agent-mode")...) {
				baseVal.AgentMode = v.AgentMode
			}
			base.PromptOverrides[k] = baseVal
		} else {
			base.PromptOverrides[k] = v
		}
	}
}

func mergeEnv(base *Config, overlay *Config) {
	if len(overlay.Env) == 0 {
		return
	}

	if base.Env == nil {
		base.Env = make(map[string]string)
	}

	for k, v := range overlay.Env {
		base.Env[k] = v
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(source))
	for k, v := range source {
		cloned[k] = v
	}

	return cloned
}

// LoadDefaultConfigForTest exports loadDefaultConfig for testing.
func LoadDefaultConfigForTest(c, target *Config) (string, error) {
	return loadDefaultConfig(c, target)
}

// ValidateConfigFileKeysForTest exports validateConfigFileKeys for testing.
func ValidateConfigFileKeysForTest(meta toml.MetaData, path string) error {
	return validateConfigFileKeys(meta, path)
}
