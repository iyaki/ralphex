package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/iyaki/ralphex/internal/config"
)

func TestReadBoolFlagOverrideReturnsZeroValueWhenUnchanged(t *testing.T) {
	cmd := NewRunCommand()

	override, err := readBoolFlagOverride(cmd, "log-truncate")
	if err != nil {
		t.Fatalf("expected no error for unchanged flag, got %v", err)
	}
	if override.changed {
		t.Fatalf("expected unchanged override, got %+v", override)
	}
}

func TestReadBoolFlagOverrideTracksExplicitTrue(t *testing.T) {
	cmd := NewRunCommand()
	if err := cmd.ParseFlags([]string{"--log-truncate"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	override, err := readBoolFlagOverride(cmd, "log-truncate")
	if err != nil {
		t.Fatalf("expected no error reading override, got %v", err)
	}
	if !override.changed {
		t.Fatalf("expected changed override, got %+v", override)
	}
	if !override.value {
		t.Fatalf("expected override value true, got %+v", override)
	}
}

func TestApplyBoolFlagOverridesAppliesOnlyChangedFlags(t *testing.T) {
	tests := []struct {
		name                string
		initialLogTruncate  bool
		logTruncateOverride boolFlagOverride
		expectedLogTruncate bool
	}{
		{
			name:               "changed",
			initialLogTruncate: false,
			logTruncateOverride: boolFlagOverride{
				changed: true,
				value:   true,
			},
			expectedLogTruncate: true,
		},
		{
			name:                "unchanged",
			initialLogTruncate:  false,
			expectedLogTruncate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				LogTruncate: tt.initialLogTruncate,
			}

			applyBoolFlagOverrides(cfg, tt.logTruncateOverride)

			if cfg.LogTruncate != tt.expectedLogTruncate {
				t.Fatalf("expected LogTruncate=%v, got %v", tt.expectedLogTruncate, cfg.LogTruncate)
			}
		})
	}
}

func TestReadEnvFlagOverridesReturnsEmptyWhenUnchanged(t *testing.T) {
	cmd := NewRunCommand()

	overrides, err := readEnvFlagOverrides(cmd)
	if err != nil {
		t.Fatalf("expected no error for unchanged --env flag, got %v", err)
	}
	if len(overrides) != 0 {
		t.Fatalf("expected empty overrides for unchanged --env flag, got %+v", overrides)
	}
}

func TestReadEnvFlagOverridesParsesSplitOnFirstEquals(t *testing.T) {
	cmd := NewRunCommand()
	if err := cmd.ParseFlags([]string{"--env", "FOO=bar", "--env", "EMPTY=", "--env", "COMPLEX=a=b=c"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	overrides, err := readEnvFlagOverrides(cmd)
	if err != nil {
		t.Fatalf("expected no error reading --env overrides, got %v", err)
	}

	expected := map[string]string{
		"FOO":     "bar",
		"EMPTY":   "",
		"COMPLEX": "a=b=c",
	}

	if !reflect.DeepEqual(overrides, expected) {
		t.Fatalf("expected parsed --env overrides %+v, got %+v", expected, overrides)
	}
}

func TestReadEnvFlagOverridesRejectsEntryWithoutEquals(t *testing.T) {
	cmd := NewRunCommand()
	if err := cmd.ParseFlags([]string{"--env", "NOT_VALID"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	_, err := readEnvFlagOverrides(cmd)
	if err == nil {
		t.Fatal("expected error for --env entry without '='")
	}
	if !strings.Contains(err.Error(), "expected KEY=VALUE") {
		t.Fatalf("expected KEY=VALUE validation error, got %v", err)
	}
}

func TestReadEnvFlagOverridesRejectsInvalidKey(t *testing.T) {
	cmd := NewRunCommand()
	if err := cmd.ParseFlags([]string{"--env", "1INVALID=value"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	_, err := readEnvFlagOverrides(cmd)
	if err == nil {
		t.Fatal("expected error for invalid --env key")
	}
	if !strings.Contains(err.Error(), "invalid --env key") {
		t.Fatalf("expected invalid key error, got %v", err)
	}
}

func TestReadEnvFlagOverridesLastValueWins(t *testing.T) {
	cmd := NewRunCommand()
	if err := cmd.ParseFlags([]string{"--env", "FOO=one", "--env", "FOO=two"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	overrides, err := readEnvFlagOverrides(cmd)
	if err != nil {
		t.Fatalf("expected no error reading duplicate --env keys, got %v", err)
	}
	if got := overrides["FOO"]; got != "two" {
		t.Fatalf("expected last --env value to win, got %q", got)
	}
}

func TestApplyEnvFlagOverridesCreatesEnvMapWhenMissing(t *testing.T) {
	cfg := &config.Config{}

	applyEnvFlagOverrides(cfg, map[string]string{
		"FOO": "bar",
	})

	if cfg.Env == nil {
		t.Fatal("expected Env map to be initialized")
	}
	if got := cfg.Env["FOO"]; got != "bar" {
		t.Fatalf("expected Env[FOO]=bar, got %q", got)
	}
}

func TestApplyEnvFlagOverridesOverwritesExistingKeys(t *testing.T) {
	cfg := &config.Config{
		Env: map[string]string{
			"FOO": "from-config",
			"BAR": "keep",
		},
	}

	applyEnvFlagOverrides(cfg, map[string]string{
		"FOO": "from-flag",
	})

	expected := map[string]string{
		"FOO": "from-flag",
		"BAR": "keep",
	}

	if !reflect.DeepEqual(cfg.Env, expected) {
		t.Fatalf("expected merged env map %+v, got %+v", expected, cfg.Env)
	}
}
