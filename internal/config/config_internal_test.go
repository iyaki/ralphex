//nolint:dupl,funlen
//nolint:funlen
package config

import (
	"testing"

	"github.com/BurntSushi/toml"
)

func TestDefaultPromptsDir(t *testing.T) {
	result := defaultPromptsDir()
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestDefaultLogFile(t *testing.T) {
	result := defaultLogFile()
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestCloneStringMap(t *testing.T) {
	t.Run("nil or empty input returns nil", func(t *testing.T) {
		var nilMap map[string]string
		result := cloneStringMap(nilMap)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}

		emptyMap := map[string]string{}
		result = cloneStringMap(emptyMap)
		if result != nil {
			t.Errorf("expected nil for empty map, got %v", result)
		}
	})

	t.Run("map with values returns independent copy", func(t *testing.T) {
		input := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		result := cloneStringMap(input)

		if len(result) != len(input) {
			t.Errorf("expected length %d, got %d", len(input), len(result))
		}

		for k, v := range input {
			if result[k] != v {
				t.Errorf("key %q: expected %q, got %q", k, v, result[k])
			}
		}

		input["key1"] = "modified"
		if result["key1"] == "modified" {
			t.Error("result was not an independent copy")
		}
		if result["key1"] != "value1" {
			t.Errorf("expected result[\"key1\"] to still be \"value1\", got %q", result["key1"])
		}
	})
}

func TestMergeScalars(t *testing.T) {
	t.Run("empty metadata leaves base unchanged", func(t *testing.T) {
		base := &Config{
			MaxIterations: 10,
			Model:         "base-model",
		}
		overlay := &Config{
			MaxIterations: 25,
			Model:         "overlay-model",
		}
		meta := toml.MetaData{}

		mergeScalars(base, overlay, meta)

		if base.MaxIterations != 10 {
			t.Errorf("expected MaxIterations unchanged, got %d", base.MaxIterations)
		}
		if base.Model != "base-model" {
			t.Errorf("expected Model unchanged, got %q", base.Model)
		}
	})
}

func TestMergeFileAndSpecScalars(t *testing.T) {
	t.Run("prompt-file defined", func(t *testing.T) {
		base := &Config{PromptFile: "base.md"}
		overlay := &Config{PromptFile: "overlay.md"}
		meta := tomlMetaWithKeys(t, "prompt-file")

		mergeFileAndSpecScalars(base, overlay, meta)

		if base.PromptFile != "overlay.md" {
			t.Errorf("expected PromptFile to be \"overlay.md\", got %q", base.PromptFile)
		}
	})

	t.Run("specs-dir defined", func(t *testing.T) {
		base := &Config{SpecsDir: "base-specs"}
		overlay := &Config{SpecsDir: "overlay-specs"}
		meta := tomlMetaWithKeys(t, "specs-dir")

		mergeFileAndSpecScalars(base, overlay, meta)

		if base.SpecsDir != "overlay-specs" {
			t.Errorf("expected SpecsDir to be \"overlay-specs\", got %q", base.SpecsDir)
		}
	})

	t.Run("specs-index-file defined", func(t *testing.T) {
		base := &Config{SpecsIndexFile: "base.json"}
		overlay := &Config{SpecsIndexFile: "overlay.json"}
		meta := tomlMetaWithKeys(t, "specs-index-file")

		mergeFileAndSpecScalars(base, overlay, meta)

		if base.SpecsIndexFile != "overlay.json" {
			t.Errorf("expected SpecsIndexFile to be \"overlay.json\", got %q", base.SpecsIndexFile)
		}
	})

	t.Run("implementation-plan-name defined", func(t *testing.T) {
		base := &Config{ImplementationPlanName: "base-plan.md"}
		overlay := &Config{ImplementationPlanName: "overlay-plan.md"}
		meta := tomlMetaWithKeys(t, "implementation-plan-name")

		mergeFileAndSpecScalars(base, overlay, meta)

		if base.ImplementationPlanName != "overlay-plan.md" {
			t.Errorf("expected ImplementationPlanName to be \"overlay-plan.md\", got %q", base.ImplementationPlanName)
		}
	})
}

func TestMergePromptAndLogScalars(t *testing.T) {
	t.Run("log-file defined", func(t *testing.T) {
		base := &Config{LogFile: "base.log"}
		overlay := &Config{LogFile: "overlay.log"}
		meta := tomlMetaWithKeys(t, "log-file")

		mergePromptAndLogScalars(base, overlay, meta)

		if base.LogFile != "overlay.log" {
			t.Errorf("expected LogFile to be \"overlay.log\", got %q", base.LogFile)
		}
	})

	t.Run("custom-prompt defined", func(t *testing.T) {
		base := &Config{CustomPrompt: "base prompt"}
		overlay := &Config{CustomPrompt: "overlay prompt"}
		meta := tomlMetaWithKeys(t, "custom-prompt")

		mergePromptAndLogScalars(base, overlay, meta)

		if base.CustomPrompt != "overlay prompt" {
			t.Errorf("expected CustomPrompt to be \"overlay prompt\", got %q", base.CustomPrompt)
		}
	})

	t.Run("prompts-dir defined", func(t *testing.T) {
		base := &Config{PromptsDir: "base-prompts"}
		overlay := &Config{PromptsDir: "overlay-prompts"}
		meta := tomlMetaWithKeys(t, "prompts-dir")

		mergePromptAndLogScalars(base, overlay, meta)

		if base.PromptsDir != "overlay-prompts" {
			t.Errorf("expected PromptsDir to be \"overlay-prompts\", got %q", base.PromptsDir)
		}
	})
}

func TestMergeAgentScalars(t *testing.T) {
	t.Run("agent defined", func(t *testing.T) {
		base := &Config{AgentName: "base-agent"}
		overlay := &Config{AgentName: "overlay-agent"}
		meta := tomlMetaWithKeys(t, "agent")

		mergeAgentScalars(base, overlay, meta)

		if base.AgentName != "overlay-agent" {
			t.Errorf("expected AgentName to be \"overlay-agent\", got %q", base.AgentName)
		}
	})

	t.Run("model defined", func(t *testing.T) {
		base := &Config{Model: "base-model"}
		overlay := &Config{Model: "overlay-model"}
		meta := tomlMetaWithKeys(t, "model")

		mergeAgentScalars(base, overlay, meta)

		if base.Model != "overlay-model" {
			t.Errorf("expected Model to be \"overlay-model\", got %q", base.Model)
		}
	})

	t.Run("agent-mode defined", func(t *testing.T) {
		base := &Config{AgentMode: "base-mode"}
		overlay := &Config{AgentMode: "overlay-mode"}
		meta := tomlMetaWithKeys(t, "agent-mode")

		mergeAgentScalars(base, overlay, meta)

		if base.AgentMode != "overlay-mode" {
			t.Errorf("expected AgentMode to be \"overlay-mode\", got %q", base.AgentMode)
		}
	})
}

//nolint:funlen
func TestMergePromptOverrides(t *testing.T) {
	t.Run("overlay has no overrides", func(t *testing.T) {
		base := &Config{
			PromptOverrides: map[string]PromptConfigOverride{
				"key1": {Model: "base-model"},
			},
		}
		overlay := &Config{}
		meta := toml.MetaData{}

		mergePromptOverrides(base, overlay, meta)

		if len(base.PromptOverrides) != 1 {
			t.Errorf("expected 1 override, got %d", len(base.PromptOverrides))
		}
	})

	t.Run("base has no overrides, overlay has one", func(t *testing.T) {
		base := &Config{}
		overlay := &Config{
			PromptOverrides: map[string]PromptConfigOverride{
				"key1": {Model: "overlay-model"},
			},
		}
		meta := toml.MetaData{}

		mergePromptOverrides(base, overlay, meta)

		if len(base.PromptOverrides) != 1 {
			t.Errorf("expected 1 override, got %d", len(base.PromptOverrides))
		}
		if base.PromptOverrides["key1"].Model != "overlay-model" {
			t.Errorf("expected Model to be \"overlay-model\", got %q", base.PromptOverrides["key1"].Model)
		}
	})

	t.Run("both have same key, overlay defines model field", func(t *testing.T) {
		base := &Config{
			PromptOverrides: map[string]PromptConfigOverride{
				"key1": {Model: "base-model", AgentMode: "base-mode"},
			},
		}
		overlay := &Config{
			PromptOverrides: map[string]PromptConfigOverride{
				"key1": {Model: "overlay-model"},
			},
		}
		meta := tomlMetaWithPromptOverrideKey(t, "key1", "model")

		mergePromptOverrides(base, overlay, meta)

		if base.PromptOverrides["key1"].Model != "overlay-model" {
			t.Errorf("expected Model to be \"overlay-model\", got %q", base.PromptOverrides["key1"].Model)
		}
		if base.PromptOverrides["key1"].AgentMode != "base-mode" {
			t.Errorf("expected AgentMode to remain \"base-mode\", got %q", base.PromptOverrides["key1"].AgentMode)
		}
	})

	t.Run("both have same key, overlay defines agent-mode field", func(t *testing.T) {
		base := &Config{
			PromptOverrides: map[string]PromptConfigOverride{
				"key1": {Model: "base-model", AgentMode: "base-mode"},
			},
		}
		overlay := &Config{
			PromptOverrides: map[string]PromptConfigOverride{
				"key1": {AgentMode: "overlay-mode"},
			},
		}
		meta := tomlMetaWithPromptOverrideKey(t, "key1", "agent-mode")

		mergePromptOverrides(base, overlay, meta)

		if base.PromptOverrides["key1"].AgentMode != "overlay-mode" {
			t.Errorf("expected AgentMode to be \"overlay-mode\", got %q", base.PromptOverrides["key1"].AgentMode)
		}
		if base.PromptOverrides["key1"].Model != "base-model" {
			t.Errorf("expected Model to remain \"base-model\", got %q", base.PromptOverrides["key1"].Model)
		}
	})
}

func TestMergeEnv(t *testing.T) {
	t.Run("overlay has no env", func(t *testing.T) {
		base := &Config{
			Env: map[string]string{"BASE_KEY": "base_value"},
		}
		overlay := &Config{}

		mergeEnv(base, overlay)

		if len(base.Env) != 1 || base.Env["BASE_KEY"] != "base_value" {
			t.Errorf("expected base env unchanged, got %v", base.Env)
		}
	})

	t.Run("base nil, overlay has env", func(t *testing.T) {
		base := &Config{}
		overlay := &Config{
			Env: map[string]string{"OVERLAY_KEY": "overlay_value"},
		}

		mergeEnv(base, overlay)

		if base.Env["OVERLAY_KEY"] != "overlay_value" {
			t.Errorf("expected OVERLAY_KEY to be \"overlay_value\", got %q", base.Env["OVERLAY_KEY"])
		}
	})

	t.Run("both have env, same key", func(t *testing.T) {
		base := &Config{
			Env: map[string]string{"SHARED_KEY": "base_value"},
		}
		overlay := &Config{
			Env: map[string]string{"SHARED_KEY": "overlay_value"},
		}

		mergeEnv(base, overlay)

		if base.Env["SHARED_KEY"] != "overlay_value" {
			t.Errorf("expected SHARED_KEY to be \"overlay_value\", got %q", base.Env["SHARED_KEY"])
		}
	})
}

// Helper to create toml.MetaData with specific keys defined at top level.
func tomlMetaWithKeys(t *testing.T, keys ...string) toml.MetaData {
	t.Helper()
	content := ""
	for _, key := range keys {
		content += key + " = \"dummy\"\n"
	}

	var cfg map[string]interface{}
	meta, err := toml.Decode(content, &cfg)
	if err != nil {
		t.Fatalf("failed to decode TOML: %v", err)
	}

	return meta
}

// Helper to create toml.MetaData with a specific prompt-overrides field defined.
func tomlMetaWithPromptOverrideKey(t *testing.T, key, field string) toml.MetaData {
	t.Helper()
	// Create TOML with nested table structure
	content := "[prompt-overrides." + key + "]\n" + field + " = \"dummy\"\n"

	var cfg map[string]interface{}
	meta, err := toml.Decode(content, &cfg)
	if err != nil {
		t.Fatalf("failed to decode TOML: %v", err)
	}

	return meta
}

func TestResolveLogTruncate(t *testing.T) {
	tests := []struct {
		name      string
		flagValue bool
		envValue  string
		fileValue bool
		expected  bool
	}{
		{"flag true wins", true, "", false, true},
		{"flag true wins over env", true, "false", false, true},
		{"env false inverts to true", false, "false", false, true},
		{"env true inverts to false", false, "true", false, false},
		{"env invalid falls through to file", false, "invalid", true, true},
		{"no flag or env uses file", false, "", true, true},
		{"no flag or env uses file false", false, "", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveLogTruncate(tc.flagValue, tc.envValue, tc.fileValue)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestMergeScalarsAllFields(t *testing.T) {
	base := &Config{}
	overlay := &Config{
		MaxIterations:  99,
		NoSpecsIndex:   true,
		LogTruncate:    true,
		AgentName:      "omp",
		Model:          "m4",
		SpecsDir:       "specs",
		SpecsIndexFile: "README.md",
		LogFile:        "ralph.log",
	}
	meta := tomlMetaWithKeys(t, "max-iterations", "no-specs-index", "log-truncate",
		"agent", "model", "specs-dir", "specs-index-file", "log-file")

	mergeScalars(base, overlay, meta)

	if base.MaxIterations != 99 {
		t.Errorf("expected MaxIterations 99, got %d", base.MaxIterations)
	}
	if !base.NoSpecsIndex {
		t.Error("expected NoSpecsIndex true")
	}
	if !base.LogTruncate {
		t.Error("expected LogTruncate true")
	}
	if base.AgentName != "omp" {
		t.Errorf("expected AgentName 'omp', got %q", base.AgentName)
	}
}

func TestMergeScalarsPartialOverlay(t *testing.T) {
	base := &Config{
		MaxIterations: 25,
		AgentName:     "claude",
		Model:         "sonnet",
	}
	overlay := &Config{
		MaxIterations: 50,
	}
	meta := tomlMetaWithKeys(t, "max-iterations")

	mergeScalars(base, overlay, meta)

	if base.MaxIterations != 50 {
		t.Errorf("expected MaxIterations 50, got %d", base.MaxIterations)
	}
	if base.AgentName != "claude" {
		t.Errorf("expected AgentName to remain 'claude', got %q", base.AgentName)
	}
}
