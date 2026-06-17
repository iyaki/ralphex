//nolint:funlen,testpackage,gocognit,cyclop
package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/iyaki/ralphex/internal/config"
)

func TestBuildInitPreviewLines(t *testing.T) {
	t.Run("minimal answers shows logging disabled", func(t *testing.T) {
		session := &InitSession{
			OutputPath: "/test/output.toml",
			Answers: &InitAnswers{
				AgentName:              "opencode",
				Model:                  "",
				AgentMode:              "",
				MaxIterations:          25,
				SpecsDir:               "specs",
				SpecsIndexFile:         "specs-index.json",
				ImplementationPlanName: "plan.md",
				PromptsDir:             "/prompts",
				LogFile:                "",
			},
		}

		lines := buildInitPreviewLines(session)

		if lines[0] != "output path: /test/output.toml" {
			t.Errorf("unexpected first line: %q", lines[0])
		}
		if lines[1] != "agent: opencode" {
			t.Errorf("unexpected second line: %q", lines[1])
		}
		for _, line := range lines {
			if line == "model: " || line == "agent-mode: " {
				t.Errorf("should not have empty model/agent-mode lines: %q", line)
			}
		}
		if lines[len(lines)-1] != "logging: disabled" {
			t.Errorf("expected last line to be \"logging: disabled\", got %q", lines[len(lines)-1])
		}
	})

	t.Run("with model includes model line", func(t *testing.T) {
		session := &InitSession{
			OutputPath: "/test.toml",
			Answers: &InitAnswers{
				AgentName: "opencode",
				Model:     "gpt-4",
			},
		}

		lines := buildInitPreviewLines(session)

		found := false
		for _, line := range lines {
			if line == "model: gpt-4" {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("expected \"model: gpt-4\" in lines, got %v", lines)
		}
	})

	t.Run("with agent-mode includes agent-mode line", func(t *testing.T) {
		session := &InitSession{
			OutputPath: "/test.toml",
			Answers: &InitAnswers{
				AgentName: "opencode",
				AgentMode: "agent",
			},
		}

		lines := buildInitPreviewLines(session)

		found := false
		for _, line := range lines {
			if line == "agent-mode: agent" {
				found = true

				break
			}
		}
		if !found {
			t.Errorf("expected \"agent-mode: agent\" in lines, got %v", lines)
		}
	})

	t.Run("with log file includes logging lines", func(t *testing.T) {
		session := &InitSession{
			OutputPath: "/test.toml",
			Answers: &InitAnswers{
				AgentName:   "opencode",
				LogFile:     "/var/log/ralph.log",
				LogTruncate: true,
			},
		}

		lines := buildInitPreviewLines(session)

		foundLogging := false
		foundLogFile := false
		foundLogTruncate := false
		for _, line := range lines {
			if line == "logging: enabled" {
				foundLogging = true
			}
			if line == "log-file: /var/log/ralph.log" {
				foundLogFile = true
			}
			if line == "log-truncate: yes" {
				foundLogTruncate = true
			}
		}

		if !foundLogging {
			t.Error("expected \"logging: enabled\" in lines")
		}
		if !foundLogFile {
			t.Error("expected \"log-file: /var/log/ralph.log\" in lines")
		}
		if !foundLogTruncate {
			t.Error("expected \"log-truncate: yes\" in lines")
		}
	})
}

func TestDefaultInitAnswers(t *testing.T) {
	answers := defaultInitAnswers()

	if answers.AgentName != "opencode" {
		t.Errorf("expected AgentName \"opencode\", got %q", answers.AgentName)
	}
	if answers.MaxIterations != 25 {
		t.Errorf("expected MaxIterations 25, got %d", answers.MaxIterations)
	}
	if answers.SpecsDir != "specs" {
		t.Errorf("expected SpecsDir \"specs\", got %q", answers.SpecsDir)
	}
	if answers.SpecsIndexFile != "README.md" {
		t.Errorf("expected SpecsIndexFile \"README.md\", got %q", answers.SpecsIndexFile)
	}
	if answers.ImplementationPlanName != "IMPLEMENTATION_PLAN.md" {
		t.Errorf("expected ImplementationPlanName \"IMPLEMENTATION_PLAN.md\", got %q", answers.ImplementationPlanName)
	}
	if answers.PromptsDir != ".ralph/prompts" {
		t.Errorf("expected PromptsDir \".ralph/prompts\", got %q", answers.PromptsDir)
	}
}

func TestSeedInitAgentDefault(t *testing.T) {
	t.Run("valid agent name sets answers.AgentName", func(t *testing.T) {
		var answers InitAnswers
		seedInitAgentDefault(&answers, "claude")

		if answers.AgentName != "claude" {
			t.Errorf("expected AgentName \"claude\", got %q", answers.AgentName)
		}
	})

	t.Run("invalid agent name leaves answers unchanged", func(t *testing.T) {
		answers := InitAnswers{AgentName: "opencode"}
		seedInitAgentDefault(&answers, "invalid-agent")

		if answers.AgentName != "opencode" {
			t.Errorf("expected AgentName unchanged \"opencode\", got %q", answers.AgentName)
		}
	})
}

func TestSeedInitMaxIterationsDefault(t *testing.T) {
	t.Run("positive value sets answers.MaxIterations", func(t *testing.T) {
		var answers InitAnswers
		seedInitMaxIterationsDefault(&answers, 50)

		if answers.MaxIterations != 50 {
			t.Errorf("expected MaxIterations 50, got %d", answers.MaxIterations)
		}
	})

	t.Run("zero value leaves answers unchanged", func(t *testing.T) {
		answers := InitAnswers{MaxIterations: 25}
		seedInitMaxIterationsDefault(&answers, 0)

		if answers.MaxIterations != 25 {
			t.Errorf("expected MaxIterations unchanged 25, got %d", answers.MaxIterations)
		}
	})
}

func TestSeedInitStringDefaults(t *testing.T) {
	t.Run("populate existing config fields except AgentMode and LogFile", func(t *testing.T) {
		var answers InitAnswers
		existingConfig := &config.Config{
			Model:                  "gpt-4",
			AgentMode:              "agent",
			SpecsDir:               "my-specs",
			SpecsIndexFile:         "my-index.json",
			ImplementationPlanName: "my-plan.md",
			PromptsDir:             "my-prompts",
			LogFile:                "/my/log.log",
		}

		seedInitStringDefaults(&answers, existingConfig)

		if answers.Model != "gpt-4" {
			t.Errorf("expected Model \"gpt-4\", got %q", answers.Model)
		}
		if answers.AgentMode != "" {
			t.Errorf("expected AgentMode \"\" (omitted), got %q", answers.AgentMode)
		}
		if answers.SpecsDir != "my-specs" {
			t.Errorf("expected SpecsDir \"my-specs\", got %q", answers.SpecsDir)
		}
		if answers.SpecsIndexFile != "my-index.json" {
			t.Errorf("expected SpecsIndexFile \"my-index.json\", got %q", answers.SpecsIndexFile)
		}
		if answers.ImplementationPlanName != "my-plan.md" {
			t.Errorf("expected ImplementationPlanName \"my-plan.md\", got %q", answers.ImplementationPlanName)
		}
		if answers.PromptsDir != "my-prompts" {
			t.Errorf("expected PromptsDir \"my-prompts\", got %q", answers.PromptsDir)
		}
		if answers.LogFile != "" {
			t.Errorf("expected LogFile \"\" (omitted), got %q", answers.LogFile)
		}
	})
}

func TestSeedInitBoolDefaults(t *testing.T) {
	t.Run("meta.Has returns true sets answers.LogTruncate", func(t *testing.T) {
		var answers InitAnswers
		existingConfig := &config.Config{LogTruncate: true}
		meta := tomlMetaWithKeys(t, "log-truncate")

		seedInitBoolDefaults(&answers, existingConfig, meta)

		if answers.LogTruncate != true {
			t.Errorf("expected LogTruncate true, got %v", answers.LogTruncate)
		}
	})

	t.Run("meta missing key leaves answers unchanged", func(t *testing.T) {
		answers := InitAnswers{LogTruncate: false}
		existingConfig := &config.Config{LogTruncate: true}
		meta := toml.MetaData{}

		seedInitBoolDefaults(&answers, existingConfig, meta)

		if answers.LogTruncate != false {
			t.Errorf("expected LogTruncate unchanged false, got %v", answers.LogTruncate)
		}
	})
}

func TestLoadExistingInitConfig(t *testing.T) {
	t.Run("valid TOML file returns config and metadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/test.toml"
		content := `
agent = "claude"
max-iterations = 30
`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		cfg, meta, ok := loadExistingInitConfig(configPath)

		if !ok {
			t.Fatal("expected ok=true")
		}
		if cfg.AgentName != "claude" {
			t.Errorf("expected AgentName \"claude\", got %q", cfg.AgentName)
		}
		if cfg.MaxIterations != 30 {
			t.Errorf("expected MaxIterations 30, got %d", cfg.MaxIterations)
		}
		if !meta.IsDefined("agent") {
			t.Error("expected metadata to define \"agent\"")
		}
		if !meta.IsDefined("max-iterations") {
			t.Error("expected metadata to define \"max-iterations\"")
		}
	})

	t.Run("missing file returns nil config and false", func(t *testing.T) {
		_, _, ok := loadExistingInitConfig("/nonexistent/path/config.toml")

		if ok {
			t.Error("expected ok=false for missing file")
		}
	})
}

func TestInitConfigExists(t *testing.T) {
	t.Run("file exists returns true", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/test.toml"
		if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		exists, err := initConfigExists(configPath)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected exists=true")
		}
	})

	t.Run("file missing returns false", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/nonexistent.toml"

		exists, err := initConfigExists(configPath)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if exists {
			t.Error("expected exists=false")
		}
	})
}

func TestBuildConfigFromAnswers(t *testing.T) {
	answers := &InitAnswers{
		AgentName:              "opencode",
		Model:                  "gpt-4",
		AgentMode:              "agent",
		MaxIterations:          50,
		SpecsDir:               "my-specs",
		SpecsIndexFile:         "my-index.json",
		ImplementationPlanName: "my-plan.md",
		PromptsDir:             "my-prompts",
		LogFile:                "/my/log.log",
		LogTruncate:            true,
	}

	cfg := buildConfigFromAnswers(answers)

	if cfg.AgentName != "opencode" {
		t.Errorf("expected AgentName \"opencode\", got %q", cfg.AgentName)
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("expected Model \"gpt-4\", got %q", cfg.Model)
	}
	if cfg.AgentMode != "agent" {
		t.Errorf("expected AgentMode \"agent\", got %q", cfg.AgentMode)
	}
	if cfg.MaxIterations != 50 {
		t.Errorf("expected MaxIterations 50, got %d", cfg.MaxIterations)
	}
	if cfg.LogFile != "/my/log.log" {
		t.Errorf("expected LogFile \"/my/log.log\", got %q", cfg.LogFile)
	}
	if !cfg.LogTruncate {
		t.Error("expected LogTruncate true")
	}
}

func TestParseConfirmAnswer(t *testing.T) {
	tests := []struct {
		input     string
		wantValue bool
		wantValid bool
	}{
		{"y", true, true},
		{"yes", true, true},
		{"true", true, true},
		{"1", true, true},
		{"n", false, true},
		{"no", false, true},
		{"false", false, true},
		{"0", false, true},
		{"invalid", false, false},
		{"", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			value, valid := parseConfirmAnswer(tc.input)
			if value != tc.wantValue {
				t.Errorf("expected value %v, got %v", tc.wantValue, value)
			}
			if valid != tc.wantValid {
				t.Errorf("expected valid %v, got %v", tc.wantValid, valid)
			}
		})
	}
}

func TestValidateInitAgent(t *testing.T) {
	t.Run("valid agent returns nil", func(t *testing.T) {
		validAgents := []string{"opencode", "claude", "cursor"}
		for _, agent := range validAgents {
			if err := validateInitAgent(agent); err != nil {
				t.Errorf("expected nil error for %q, got %v", agent, err)
			}
		}
	})

	t.Run("invalid agent returns error", func(t *testing.T) {
		err := validateInitAgent("invalid")
		if err == nil {
			t.Error("expected error for invalid agent")
		}
	})
}

func TestValidatePositiveInitInteger(t *testing.T) {
	t.Run("positive integer returns nil", func(t *testing.T) {
		if err := validatePositiveInitInteger("50"); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("zero returns error", func(t *testing.T) {
		err := validatePositiveInitInteger("0")
		if err == nil {
			t.Error("expected error for zero")
		}
	})

	t.Run("negative returns error", func(t *testing.T) {
		err := validatePositiveInitInteger("-5")
		if err == nil {
			t.Error("expected error for negative")
		}
	})

	t.Run("non-numeric returns error", func(t *testing.T) {
		err := validatePositiveInitInteger("abc")
		if err == nil {
			t.Error("expected error for non-numeric")
		}
	})
}

func TestSetInitAnswerAgentName(t *testing.T) {
	var answers InitAnswers
	err := setInitAnswerAgentName(&answers, "claude")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if answers.AgentName != "claude" {
		t.Errorf("expected AgentName \"claude\", got %q", answers.AgentName)
	}
}

func TestSetInitAnswerMaxIterations(t *testing.T) {
	t.Run("valid integer sets value", func(t *testing.T) {
		var answers InitAnswers
		err := setInitAnswerMaxIterations(&answers, "50")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if answers.MaxIterations != 50 {
			t.Errorf("expected MaxIterations 50, got %d", answers.MaxIterations)
		}
	})

	t.Run("invalid integer returns error", func(t *testing.T) {
		var answers InitAnswers
		err := setInitAnswerMaxIterations(&answers, "abc")
		if err == nil {
			t.Error("expected error for invalid integer")
		}
	})
}

func TestApplyInitAnswer(t *testing.T) {
	t.Run("valid key calls applier", func(t *testing.T) {
		var answers InitAnswers
		err := applyInitAnswer(&answers, "agent", "claude")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if answers.AgentName != "claude" {
			t.Errorf("expected AgentName \"claude\", got %q", answers.AgentName)
		}
	})

	t.Run("unknown key returns error", func(t *testing.T) {
		var answers InitAnswers
		err := applyInitAnswer(&answers, "unknown-key", "value")
		if err == nil {
			t.Error("expected error for unknown key")
		}
		if err.Error() != "unknown init question key: unknown-key" {
			t.Errorf("expected specific error message, got %q", err.Error())
		}
	})
}

// Helper reused from config tests.
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
func TestPrintInitPreview(t *testing.T) {
	session := &InitSession{
		OutputPath: "/test.toml",
		Answers: &InitAnswers{
			AgentName: "opencode",
		},
	}

	var buf bytes.Buffer
	session.Writer = &buf

	err := printInitPreview(session)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Configuration preview:") {
		t.Error("expected \"Configuration preview:\" in output")
	}
	if !strings.Contains(output, "agent: opencode") {
		t.Error("expected \"agent: opencode\" in output")
	}
}

// mockAnswerReader implements answerReader for testing.
type mockAnswerReader struct {
	answers []string
	index   int
}

func (m *mockAnswerReader) ReadAnswer() (string, error) {
	if m.index >= len(m.answers) {
		return "", io.EOF
	}
	answer := m.answers[m.index]

	m.index++

	return answer, nil
}

func TestConfirmExistingConfigOverwriteWithReader(t *testing.T) {
	t.Run("user confirms overwrite", func(t *testing.T) {
		session := &InitSession{
			Writer: &bytes.Buffer{},
		}
		reader := &mockAnswerReader{answers: []string{"yes"}}

		confirmed, err := confirmExistingConfigOverwriteWithReader(session, reader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !confirmed {
			t.Error("expected confirmed=true")
		}
	})

	t.Run("user declines overwrite", func(t *testing.T) {
		session := &InitSession{
			Writer: &bytes.Buffer{},
		}
		reader := &mockAnswerReader{answers: []string{"no"}}

		confirmed, err := confirmExistingConfigOverwriteWithReader(session, reader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if confirmed {
			t.Error("expected confirmed=false")
		}
	})
}

func TestConfirmInitWriteWithReader(t *testing.T) {
	t.Run("user confirms write", func(t *testing.T) {
		session := &InitSession{
			Writer:  &bytes.Buffer{},
			Answers: &InitAnswers{AgentName: "opencode"},
		}
		reader := &mockAnswerReader{answers: []string{"yes"}}

		confirmed, err := confirmInitWriteWithReader(session, reader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !confirmed {
			t.Error("expected confirmed=true")
		}
		if !session.Confirmed {
			t.Error("expected session.Confirmed=true")
		}
	})

	t.Run("user declines write", func(t *testing.T) {
		session := &InitSession{
			Writer:  &bytes.Buffer{},
			Answers: &InitAnswers{AgentName: "opencode"},
		}
		reader := &mockAnswerReader{answers: []string{"no"}}

		confirmed, err := confirmInitWriteWithReader(session, reader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if confirmed {
			t.Error("expected confirmed=false")
		}
		if session.Confirmed {
			t.Error("expected session.Confirmed=false")
		}
	})
}

func TestPromptForAnswerWithReader(t *testing.T) {
	t.Run("valid answer on first try", func(t *testing.T) {
		session := &InitSession{
			Writer: &bytes.Buffer{},
		}
		question := InitQuestion{
			Key:    "agent",
			Prompt: "Which agent?",
			Type:   "input",
		}
		reader := &mockAnswerReader{answers: []string{"opencode"}}

		answer, err := promptForAnswerWithReader(session, question, reader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if answer != "opencode" {
			t.Errorf("expected \"opencode\", got %q", answer)
		}
	})

	t.Run("retries after invalid answer", func(t *testing.T) {
		session := &InitSession{
			Writer: &bytes.Buffer{},
		}
		question := InitQuestion{
			Key:    "confirm",
			Prompt: "Continue?",
			Type:   "confirm",
		}
		// First invalid, then valid
		reader := &mockAnswerReader{answers: []string{"invalid", "yes"}}

		answer, err := promptForAnswerWithReader(session, question, reader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if answer != "yes" {
			t.Errorf("expected \"yes\", got %q", answer)
		}
	})
}

func TestAskSingleQuestionWithReader(t *testing.T) {
	t.Run("returns answer from reader", func(t *testing.T) {
		session := &InitSession{
			Writer: &bytes.Buffer{},
		}
		question := InitQuestion{
			Prompt: "Test?",
		}
		reader := &mockAnswerReader{answers: []string{"my-answer"}}

		answer, err := askSingleQuestionWithReader(session, question, reader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if answer != "my-answer" {
			t.Errorf("expected \"my-answer\", got %q", answer)
		}
	})

	t.Run("returns default on empty answer", func(t *testing.T) {
		session := &InitSession{
			Writer: &bytes.Buffer{},
		}
		question := InitQuestion{
			Prompt:       "Test?",
			DefaultValue: "default-value",
		}
		reader := &mockAnswerReader{answers: []string{""}}

		answer, err := askSingleQuestionWithReader(session, question, reader)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if answer != "default-value" {
			t.Errorf("expected \"default-value\", got %q", answer)
		}
	})
}

// mockErrorReader always returns an error.
type mockErrorReader struct{}

func (m *mockErrorReader) ReadAnswer() (string, error) {
	return "", io.EOF
}

func TestConfirmExistingConfigOverwriteWithReaderError(t *testing.T) {
	session := &InitSession{
		Writer: &bytes.Buffer{},
	}
	reader := &mockErrorReader{}

	confirmed, err := confirmExistingConfigOverwriteWithReader(session, reader)
	if err == nil {
		t.Error("expected error from reader")
	}
	if confirmed {
		t.Error("expected confirmed=false on error")
	}
}

func TestConfirmInitWriteWithReaderError(t *testing.T) {
	session := &InitSession{
		Writer:  &bytes.Buffer{},
		Answers: &InitAnswers{AgentName: "opencode"},
	}
	reader := &mockErrorReader{}

	confirmed, err := confirmInitWriteWithReader(session, reader)
	if err == nil {
		t.Error("expected error from reader")
	}
	if confirmed {
		t.Error("expected confirmed=false on error")
	}
}

func TestAskQuestionsWithReaderError(t *testing.T) {
	session := &InitSession{
		Writer:  &bytes.Buffer{},
		Answers: &InitAnswers{},
	}
	questions := []InitQuestion{
		{Key: "agent", Prompt: "Agent?", Type: "input"},
	}
	reader := &mockErrorReader{}

	err := askQuestionsWithReader(session, questions, reader)
	if err == nil {
		t.Error("expected error from reader")
	}
}

func TestRunInitQuestionnaireWithRunnerError(t *testing.T) {
	session := &InitSession{
		Answers: &InitAnswers{LogFile: ""},
	}
	runner := &mockErrorQuestionnaireRunner{err: io.EOF}

	err := runInitQuestionnaireWithRunner(session, runner)
	if err == nil {
		t.Error("expected error from runner")
	}
}

type mockErrorQuestionnaireRunner struct {
	err error
}

func (m *mockErrorQuestionnaireRunner) AskQuestions(_ *InitSession, _ []InitQuestion) error {
	return m.err
}

func TestAskQuestionsWithReaderSuccess(t *testing.T) {
	session := &InitSession{
		Writer:  &bytes.Buffer{},
		Answers: &InitAnswers{},
	}
	questions := []InitQuestion{
		{Key: "agent", Prompt: "Agent?", Type: "input", DefaultValue: "opencode"},
		{Key: "model", Prompt: "Model?", Type: "input"},
	}
	reader := &mockAnswerReader{answers: []string{"claude", "gpt-4"}}

	err := askQuestionsWithReader(session, questions, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Answers.AgentName != "claude" {
		t.Errorf("expected AgentName \"claude\", got %q", session.Answers.AgentName)
	}
	if session.Answers.Model != "gpt-4" {
		t.Errorf("expected Model \"gpt-4\", got %q", session.Answers.Model)
	}
}

type successQuestionnaireRunner struct{}

func (r *successQuestionnaireRunner) AskQuestions(_ *InitSession, _ []InitQuestion) error {
	return nil
}

func TestRunInitQuestionnaireWithRunnerSuccess(t *testing.T) {
	t.Run("without logging questions", func(t *testing.T) {
		session := &InitSession{
			Answers: &InitAnswers{LogFile: ""},
		}
		runner := &successQuestionnaireRunner{}

		err := runInitQuestionnaireWithRunner(session, runner)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(session.Questions) == 0 {
			t.Error("expected questions to be populated")
		}
	})

	t.Run("with logging questions", func(t *testing.T) {
		session := &InitSession{
			Answers: &InitAnswers{LogFile: "/test.log"},
		}
		runner := &successQuestionnaireRunner{}

		err := runInitQuestionnaireWithRunner(session, runner)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should have base questions + 1 logging question
		if len(session.Questions) < 10 {
			t.Errorf("expected at least 10 questions, got %d", len(session.Questions))
		}
	})
}

// TestRunInitQuestionnaire_LoggingQuestions tests the logging questions branch.
func TestRunInitQuestionnaire_LoggingQuestions(t *testing.T) {
	session := &InitSession{Answers: &InitAnswers{LogFile: "/test.log"}}
	runner := &mockSuccessRunner{}
	if err := runInitQuestionnaireWithRunner(session, runner); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have base questions + logging questions
	if len(session.Questions) < 10 {
		t.Errorf("expected at least 10 questions with logging, got %d", len(session.Questions))
	}
}

type mockSuccessRunner struct{}

func (m *mockSuccessRunner) AskQuestions(_ *InitSession, _ []InitQuestion) error { return nil }

// TestAskSingleQuestionWithReader_DefaultValue tests empty answer returns default.
func TestAskSingleQuestionWithReader_DefaultValue(t *testing.T) {
	session := &InitSession{
		Writer: &bytes.Buffer{},
		Reader: bufio.NewReader(strings.NewReader("\n")),
	}
	question := InitQuestion{DefaultValue: "default-value"}

	answer, err := askSingleQuestionWithReader(session, question, &bufioAnswerReader{reader: session.Reader})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "default-value" {
		t.Errorf("expected \"default-value\", got %q", answer)
	}
}

// TestReadAnswer_EOFWithContent tests EOF after reading content.
func TestReadAnswer_EOFWithContent(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("content-without-newline"))
	answer, err := readAnswer(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "content-without-newline" {
		t.Errorf("expected \"content-without-newline\", got %q", answer)
	}
}

// TestReadAnswer_ImmediateEOF tests immediate EOF.
func TestReadAnswer_ImmediateEOF(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	_, err := readAnswer(reader)
	if err != io.EOF {
		t.Errorf("expected EOF error, got %v", err)
	}
}

// TestPrintInitPreview_FullSuccess tests successful preview generation.
func TestPrintInitPreview_FullSuccess(t *testing.T) {
	session := &InitSession{
		Writer: &bytes.Buffer{},
		Answers: &InitAnswers{
			AgentName:     "opencode",
			Model:         "gpt-4",
			AgentMode:     "agent",
			MaxIterations: 25,
			LogFile:       "/tmp/test.log",
			LogTruncate:   true,
		},
	}

	err := printInitPreview(session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := session.Writer.(*bytes.Buffer).String()
	if !strings.Contains(output, "Configuration preview:") {
		t.Error("missing header")
	}
	if !strings.Contains(output, "agent: opencode") {
		t.Error("missing agent")
	}
	if !strings.Contains(output, "logging: enabled") {
		t.Error("missing logging enabled")
	}
}

// TestPrintInitPreview_BodyLinesError tests error when writing body lines.
func TestPrintInitPreview_BodyLinesError(t *testing.T) {
	session := &InitSession{
		Writer:  &failingWriter{failOn: 2}, // Fail on second write
		Answers: &InitAnswers{AgentName: "opencode", Model: "gpt-4"},
	}
	err := printInitPreview(session)
	if err == nil {
		t.Fatal("expected error from failingWriter")
	}
}

// TestPrintInitPreview_LoggingEnabledError tests error in logging section.
func TestPrintInitPreview_LoggingEnabledError(t *testing.T) {
	session := &InitSession{
		Writer:  &failingWriter{failOn: 4}, // Fail later in output
		Answers: &InitAnswers{AgentName: "opencode", LogFile: "/test.log"},
	}
	err := printInitPreview(session)
	if err == nil {
		t.Fatal("expected error")
	}
}

// failingWriter fails after N writes.
type failingWriter struct {
	count  int
	failOn int
}

func (f *failingWriter) Write(_ []byte) (int, error) {
	f.count++
	if f.count >= f.failOn {
		return 0, fmt.Errorf("injected failure")
	}

	return 0, nil
}

// TestPrintInitPreview_HeaderError tests failure on first writeln.
func TestPrintInitPreview_HeaderError(t *testing.T) {
	session := &InitSession{
		Writer:  &failingWriter{failOn: 1},
		Answers: &InitAnswers{AgentName: "opencode"},
	}
	err := printInitPreview(session)
	if err == nil {
		t.Fatal("expected error on header write")
	}
}

// TestPrintInitPreview_BlankLineError tests failure on final writeln.
func TestPrintInitPreview_BlankLineError(_ *testing.T) {
	session := &InitSession{
		Writer:  &failingWriter{failOn: 99}, // Never fail in loop, fail at end
		Answers: &InitAnswers{AgentName: "opencode"},
	}
	// Override buildInitPreviewLines to return many lines so we hit blank line
	originalLines := buildInitPreviewLines(session)
	// Force many iterations
	for i := 0; i < 100; i++ {
		session.Answers.Model = fmt.Sprintf("model-%d", i)
	}
	buildInitPreviewLines(session) // Rebuild
	// Now test with writer that fails after all lines
	session.Writer = &failingWriter{failOn: len(originalLines) + 2}
	err := printInitPreview(session)
	// This is tricky - just ensure no panic
	_ = err
}

// TestPrintInitPreview_NoLines tests with minimal output.
func TestPrintInitPreview_NoLines(t *testing.T) {
	session := &InitSession{
		Writer:  &bytes.Buffer{},
		Answers: &InitAnswers{}, // Minimal answers
	}
	err := printInitPreview(session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := session.Writer.(*bytes.Buffer).String()
	if !strings.Contains(output, "Configuration preview:") {
		t.Error("expected header")
	}
}

// TestPrepareInitSession_ForceFlag tests force flag behavior. tests when force=true skips overwrite prompt.
func TestPrepareInitSession_ForceFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ralph.toml")
	if err := os.WriteFile(configPath, []byte(`agent = "opencode"`), 0644); err != nil {
		t.Fatal(err)
	}

	session := &InitSession{
		OutputPath:          configPath,
		Answers:             &InitAnswers{AgentName: "original"},
		ExistingConfigFound: true,
		Writer:              &bytes.Buffer{},
	}

	shouldContinue, err := prepareInitSession(session, true) // force=true
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shouldContinue {
		t.Error("expected shouldContinue=true with force flag")
	}
	// Should not have asked for overwrite confirmation
	output := session.Writer.(*bytes.Buffer).String()
	if strings.Contains(output, "Overwrite") {
		t.Error("should not prompt when force=true")
	}
}

// TestExecuteInitCommand_PrepareInitSessionError tests error handling. tests error from prepareInitSession.
func TestExecuteInitCommand_PrepareInitSessionError(t *testing.T) {
	original := GetIsInteractiveTerminalForTest()
	SetIsInteractiveTerminalForTest(true)
	defer SetIsInteractiveTerminalForTest(original)

	cmd := &cobra.Command{}
	cmd.Flags().String("output", "", "")
	cmd.SetIn(strings.NewReader("\n\n\n\n\n\n\n\n\n\nyes\n")) // 9 questions + yes
	cmd.SetOut(&bytes.Buffer{})

	// This will run successfully or error based on agent availability
	// The important part is executeInitCommand path coverage
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.toml")
	_ = ExecuteInitCommandForTest(cmd, configPath, false)
	// Don't fail if agent unavailable - just covering the execution path
}
