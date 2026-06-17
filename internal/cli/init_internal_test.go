package cli

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func setupInteractiveInitCommand(t *testing.T, tmp string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	oldTTYCheck := isInteractiveTerminal
	oldGetwd := getWorkingDir
	t.Cleanup(func() {
		isInteractiveTerminal = oldTTYCheck
		getWorkingDir = oldGetwd
	})

	isInteractiveTerminal = func() bool { return true }
	getWorkingDir = func() (string, error) { return tmp, nil }

	cmd := NewInitCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)

	return cmd, &out
}

func defaultInitAnswersInput() string {
	return strings.Repeat("\n", 10)
}

func invalidAnswersWithRetriesInput() string {
	return strings.Join([]string{
		"bad-agent",
		"claude",
		"",
		"",
		"0",
		"2",
		"",
		"",
		"",
		"",
		"ralph.log",
		"maybe",
		"no",
		"",
	}, "\n") + "\n"
}

func assertOutputContainsPromptsInOrder(t *testing.T, output string, prompts []string) {
	t.Helper()

	lastIndex := -1
	for _, prompt := range prompts {
		idx := strings.Index(output, prompt)
		if idx == -1 {
			t.Fatalf("expected prompt %q to appear in output, got %q", prompt, output)
		}
		if idx <= lastIndex {
			t.Fatalf("expected prompt %q to appear after previous prompt in output %q", prompt, output)
		}
		lastIndex = idx
	}
}

func assertFileContainsAll(t *testing.T, path string, expectedFragments []string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected generated config file, got read error: %v", err)
	}

	contentText := string(content)
	for _, fragment := range expectedFragments {
		if !strings.Contains(contentText, fragment) {
			t.Fatalf("expected config to include %q, got %q", fragment, contentText)
		}
	}
}

func TestInitCommandWritesDefaultConfigFile(t *testing.T) {
	tmp := t.TempDir()
	cmd, _ := setupInteractiveInitCommand(t, tmp)
	cmd.SetIn(strings.NewReader(defaultInitAnswersInput()))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init to succeed, got %v", err)
	}

	configPath := filepath.Join(tmp, "ralph.toml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config file at %s, got read error: %v", configPath, err)
	}

	contentText := string(content)
	if !strings.Contains(contentText, `agent = "opencode"`) {
		t.Fatalf("expected config to include default agent, got %q", contentText)
	}
	
	// Verify empty optional fields are omitted
	if strings.Contains(contentText, `log-file`) {
		t.Errorf("empty log-file should be omitted, got %q", contentText)
	}
	if strings.Contains(contentText, `log-truncate`) {
		t.Errorf("false log-truncate should be omitted, got %q", contentText)
	}
	if strings.Contains(contentText, `model = ""`) {
		t.Errorf("empty model should be omitted, got %q", contentText)
	}
	if strings.Contains(contentText, `agent-mode = ""`) {
		t.Errorf("empty agent-mode should be omitted, got %q", contentText)
	}
}

func TestInitCommandWritesConfigToOutputPath(t *testing.T) {
	tmp := t.TempDir()
	targetPath := filepath.Join(tmp, "configs", "custom.toml")
	cmd, _ := setupInteractiveInitCommand(t, tmp)
	cmd.SetArgs([]string{"--output", targetPath})
	cmd.SetIn(strings.NewReader(defaultInitAnswersInput()))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init to succeed, got %v", err)
	}

	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("expected config file at %s, got stat error: %v", targetPath, err)
	}
}

func TestInitCommandAsksQuestionsInSpecifiedOrder(t *testing.T) {
	tmp := t.TempDir()
	cmd, out := setupInteractiveInitCommand(t, tmp)
	cmd.SetIn(strings.NewReader(defaultInitAnswersInput()))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init to succeed, got %v", err)
	}

	assertOutputContainsPromptsInOrder(t, out.String(), []string{
		"AI agent (omp/opencode/claude/cursor/oh-my-pi)",
		"Model (optional)",
		"Agent mode/sub-agent (optional)",
		"Maximum iterations",
		"Specs directory",
		"Specs index file",
		"Implementation plan file",
		"Prompts directory",
		"Log file path (leave empty to disable logging)",
		"Write configuration now?",
	})
}

func TestInitCommandPreviewDeclinedSkipsWrite(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "ralph.toml")
	cmd, out := setupInteractiveInitCommand(t, tmp)
	cmd.SetIn(strings.NewReader(strings.Repeat("\n", 9) + "no\n"))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init to exit cleanly when preview confirmation is declined, got %v", err)
	}

	if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no config file to be written, got stat err=%v", err)
	}

	output := out.String()
	assertOutputContainsAll(t, output, []string{
		"Configuration preview:",
		"agent: opencode",
		"logging: disabled",
		"Write configuration now?",
		"Initialization cancelled; configuration was not written.",
	})

	if strings.Contains(output, "Initialized Ralphex configuration") {
		t.Fatalf("expected no success output when preview confirmation is declined, got %q", output)
	}
}

func TestInitCommandRePromptsForInvalidAnswers(t *testing.T) {
	tmp := t.TempDir()
	cmd, out := setupInteractiveInitCommand(t, tmp)
	cmd.SetIn(strings.NewReader(invalidAnswersWithRetriesInput()))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init to succeed after retries, got %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "must be one of") {
		t.Fatalf("expected invalid agent validation message, got %q", output)
	}
	if !strings.Contains(output, "must be an integer greater than 0") {
		t.Fatalf("expected invalid max-iterations validation message, got %q", output)
	}
	if !strings.Contains(output, "please answer yes or no") {
		t.Fatalf("expected invalid confirmation validation message, got %q", output)
	}

	assertFileContainsAll(t, filepath.Join(tmp, "ralph.toml"), []string{
		`agent = "claude"`,
		"max-iterations = 2",
		`log-file = "ralph.log"`,
	})
}

func TestInitCommandDeclinedOverwriteLeavesExistingFileUnchanged(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "ralph.toml")
	originalConfig := "agent = \"claude\"\nmax-iterations = 99\n"

	if err := os.WriteFile(configPath, []byte(originalConfig), 0600); err != nil {
		t.Fatalf("expected setup to write existing config, got %v", err)
	}

	cmd, out := setupInteractiveInitCommand(t, tmp)
	cmd.SetIn(strings.NewReader("no\n"))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init to exit cleanly when overwrite is declined, got %v", err)
	}

	updatedContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected to read existing config after declined overwrite, got %v", err)
	}

	if string(updatedContent) != originalConfig {
		t.Fatalf("expected existing config to remain unchanged, got %q", string(updatedContent))
	}

	output := out.String()
	if !strings.Contains(output, "Overwrite existing configuration?") {
		t.Fatalf("expected overwrite confirmation prompt, got %q", output)
	}
	if !strings.Contains(output, "Initialization cancelled; existing configuration was not changed.") {
		t.Fatalf("expected declined overwrite cancellation message, got %q", output)
	}
	if strings.Contains(output, "Initialized Ralphex configuration") {
		t.Fatalf("expected no success message when overwrite is declined, got %q", output)
	}
}

func TestInitCommandConfirmedOverwriteRewritesExistingFile(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "ralph.toml")
	originalConfig := "agent = \"claude\"\nlegacy-marker = \"keep-out\"\n"

	if err := os.WriteFile(configPath, []byte(originalConfig), 0600); err != nil {
		t.Fatalf("expected setup to write existing config, got %v", err)
	}

	cmd, out := setupInteractiveInitCommand(t, tmp)
	cmd.SetIn(strings.NewReader("yes\n" + defaultInitAnswersInput()))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init to succeed when overwrite is confirmed, got %v", err)
	}

	updatedContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected overwritten config to be readable, got %v", err)
	}

	updatedText := string(updatedContent)
	if strings.Contains(updatedText, "legacy-marker") {
		t.Fatalf("expected overwritten config to drop legacy content, got %q", updatedText)
	}
	if !strings.Contains(updatedText, `agent = "claude"`) {
		t.Fatalf("expected overwritten config to seed defaults from existing valid values, got %q", updatedText)
	}

	if !strings.Contains(out.String(), "Initialized Ralphex configuration") {
		t.Fatalf("expected success output after confirmed overwrite, got %q", out.String())
	}
}

func seededInitConfigLines() []string {
	return []string{
		`agent = "claude"`,
		`model = "gpt-4o-mini"`,
		"max-iterations = 7",
		`specs-dir = "docs/specs"`,
		`specs-index-file = "INDEX.md"`,
		`implementation-plan-name = "PLAN.md"`,
		`prompts-dir = ".ralph/custom-prompts"`,
		"log-truncate = true",
	}
}

func seededInitPromptDefaults() []string {
	return []string{
		"Overwrite existing configuration? [no]:",
		"AI agent (omp/opencode/claude/cursor/oh-my-pi) [claude]:",
		"Model (optional) [gpt-4o-mini]:",
		"Agent mode/sub-agent (optional):",
		"Maximum iterations [7]:",
		"Specs directory [docs/specs]:",
		"Specs index file [INDEX.md]:",
		"Implementation plan file [PLAN.md]:",
		"Prompts directory [.ralph/custom-prompts]:",
		"Log file path (leave empty to disable logging):",
		"Configuration preview:",
		"Write configuration now? [yes]:",
	}
}

func assertOutputContainsAll(t *testing.T, output string, expectedFragments []string) {
	t.Helper()

	for _, expected := range expectedFragments {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to include %q, got %q", expected, output)
		}
	}
}


func TestInitCommandSeedsQuestionDefaultsFromExistingConfig(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "ralph.toml")
	existingConfig := strings.Join(seededInitConfigLines(), "\n") + "\n"

	if err := os.WriteFile(configPath, []byte(existingConfig), 0600); err != nil {
		t.Fatalf("expected setup to write existing config, got %v", err)
	}

	cmd, out := setupInteractiveInitCommand(t, tmp)
	cmd.SetIn(strings.NewReader("yes\n" + strings.Repeat("\n", 12)))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init to succeed when using existing defaults, got %v", err)
	}

	assertOutputContainsAll(t, out.String(), seededInitPromptDefaults())
	assertFileContainsAll(t, configPath, seededInitConfigLines())
}

func TestReadAnswerEOFBehaviors(t *testing.T) {
	t.Run("returns EOF on empty input", func(t *testing.T) {
		_, err := readAnswer(bufio.NewReader(strings.NewReader("")))
		if !errors.Is(err, io.EOF) {
			t.Fatalf("expected EOF, got %v", err)
		}
	})

	t.Run("returns partial line without error", func(t *testing.T) {
		answer, err := readAnswer(bufio.NewReader(strings.NewReader("hello")))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if answer != "hello" {
			t.Fatalf("expected answer hello, got %q", answer)
		}
	})
}

func TestNormalizeAnswerReadError(t *testing.T) {
	t.Run("normalizes EOF error", func(t *testing.T) {
		err := normalizeAnswerReadError(io.EOF)
		if err == nil || !strings.Contains(err.Error(), "unexpected end of input") {
			t.Fatalf("expected normalized EOF error, got %v", err)
		}
	})

	t.Run("passes through non-EOF", func(t *testing.T) {
		inErr := errors.New("boom")
		err := normalizeAnswerReadError(inErr)
		if !errors.Is(err, inErr) {
			t.Fatalf("expected passthrough error, got %v", err)
		}
	})
}

func TestApplyInitAnswerErrorPaths(t *testing.T) {
	answers := defaultInitAnswers()

	err := applyInitAnswer(answers, "unknown-key", "value")
	if err == nil || !strings.Contains(err.Error(), "unknown init question key") {
		t.Fatalf("expected unknown key error, got %v", err)
	}

	err = applyInitAnswer(answers, questionKeyMaxIterations, "abc")
	if err == nil || !strings.Contains(err.Error(), "invalid max-iterations answer") {
		t.Fatalf("expected invalid max-iterations error, got %v", err)
	}

	err = applyInitAnswer(answers, questionKeyLogTruncate, "maybe")
	if err == nil || !errors.Is(err, errInvalidConfirmAnswer) {
		t.Fatalf("expected invalid truncate confirm answer error, got %v", err)
	}
}

func TestAskQuestionsReturnsApplyErrors(t *testing.T) {
	session := &InitSession{
		Answers: defaultInitAnswers(),
		Reader:  bufio.NewReader(strings.NewReader("value\n")),
		Writer:  io.Discard,
	}

	err := askQuestions(session, []InitQuestion{{Key: "unknown-key", Prompt: "q", Type: questionTypeInput}})
	if err == nil || !strings.Contains(err.Error(), "unknown init question key") {
		t.Fatalf("expected apply error from askQuestions, got %v", err)
	}
}

func TestPromptForAnswerEOF(t *testing.T) {
	session := &InitSession{
		Reader: bufio.NewReader(strings.NewReader("")),
		Writer: io.Discard,
	}

	_, err := promptForAnswer(session, InitQuestion{Prompt: "q", Type: questionTypeInput})
	if err == nil || !strings.Contains(err.Error(), "unexpected end of input") {
		t.Fatalf("expected EOF prompt error, got %v", err)
	}
}

func TestValidateQuestionAnswerBranches(t *testing.T) {
	err := validateQuestionAnswer(InitQuestion{Required: true}, "")
	if !errors.Is(err, errInitValueRequired) {
		t.Fatalf("expected required value error, got %v", err)
	}

	err = validateQuestionAnswer(InitQuestion{Type: questionTypeConfirm}, "maybe")
	if !errors.Is(err, errInvalidConfirmAnswer) {
		t.Fatalf("expected confirm answer error, got %v", err)
	}

	validatorErr := errors.New("validation failed")
	err = validateQuestionAnswer(InitQuestion{Validator: func(string) error { return validatorErr }}, "x")
	if !errors.Is(err, validatorErr) {
		t.Fatalf("expected validator error, got %v", err)
	}
}

func TestBoolToConfirmValue(t *testing.T) {
	if got := boolToConfirmValue(true); got != confirmYes {
		t.Fatalf("expected %q, got %q", confirmYes, got)
	}
	if got := boolToConfirmValue(false); got != confirmNo {
		t.Fatalf("expected %q, got %q", confirmNo, got)
	}
}

func TestIsInteractiveTerminalRejectsDevNullStreams(t *testing.T) {
	stdinFile, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("failed to open %s for stdin: %v", os.DevNull, err)
	}
	t.Cleanup(func() {
		_ = stdinFile.Close()
	})

	stdoutFile, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("failed to open %s for stdout: %v", os.DevNull, err)
	}
	t.Cleanup(func() {
		_ = stdoutFile.Close()
	})

	originalStdin := os.Stdin
	originalStdout := os.Stdout
	t.Cleanup(func() {
		os.Stdin = originalStdin
		os.Stdout = originalStdout
	})

	os.Stdin = stdinFile
	os.Stdout = stdoutFile

	if isInteractiveTerminal() {
		t.Fatal("expected non-interactive terminal check for /dev/null streams")
	}
}

func TestAskSingleQuestionWithReaderSuccess(t *testing.T) {
	tmp := t.TempDir()
	cmd, out := setupInteractiveInitCommand(t, tmp)
	cmd.SetIn(strings.NewReader("yes\n"))

	question := newConfirmQuestion(questionKeyWriteConfiguration, "Write configuration now?", confirmYes)
	answer, err := askSingleQuestionWithReader(&InitSession{
		Reader: bufio.NewReader(cmd.InOrStdin()),
		Writer: out,
	}, question, &bufioAnswerReader{reader: bufio.NewReader(cmd.InOrStdin())})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer != "yes" {
		t.Errorf("expected answer yes, got %q", answer)
	}
}

func TestReadBoolFlagOverrideForTest(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("test-flag", false, "")

	result, err := ReadBoolFlagOverrideForTest(cmd, "test-flag")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Changed {
		t.Error("expected Changed=false")
	}
	if result.Value {
		t.Error("expected Value=false")
	}
}

func TestReadEnvFlagOverridesForTest(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("env", []string{}, "")
	if err := cmd.Flags().Set("env", "KEY=value"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	result, err := ReadEnvFlagOverridesForTest(cmd)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["KEY"] != "value" {
		t.Errorf("expected KEY=value, got %v", result)
	}
}

func TestAskQuestionsSuccess(t *testing.T) {
	cmd, out := setupInteractiveInitCommand(t, t.TempDir())
	cmd.SetIn(strings.NewReader("test-answer\n"))

	session := &InitSession{
		Reader: bufio.NewReader(cmd.InOrStdin()),
		Writer: out,
		Answers: &InitAnswers{},
	}

	questions := []InitQuestion{
		newInputQuestion(questionKeyModel, "Model", "", false, nil),
	}

	err := askQuestions(session, questions)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if session.Answers.Model != "test-answer" {
		t.Errorf("expected Model to be test-answer, got %q", session.Answers.Model)
	}
}

func TestStandardQuestionnaireRunnerAskQuestions(t *testing.T) {
	cmd, out := setupInteractiveInitCommand(t, t.TempDir())
	cmd.SetIn(strings.NewReader("answer1\nanswer2\n"))

	session := &InitSession{
		Reader: bufio.NewReader(cmd.InOrStdin()),
		Writer: out,
		Answers: &InitAnswers{},
	}
	
	runner := &standardQuestionnaireRunner{}
	questions := []InitQuestion{
		newInputQuestion(questionKeyModel, "Model", "", false, nil),
		newInputQuestion(questionKeySpecsDir, "Specs dir", "", false, nil),
	}

	err := runner.AskQuestions(session, questions)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if session.Answers.Model != "answer1" {
		t.Errorf("expected Model to be answer1, got %q", session.Answers.Model)
	}
	if session.Answers.SpecsDir != "answer2" {
		t.Errorf("expected SpecsDir to be answer2, got %q", session.Answers.SpecsDir)
	}
}

func TestAskSingleQuestion(t *testing.T) {
	cmd, out := setupInteractiveInitCommand(t, t.TempDir())
	cmd.SetIn(strings.NewReader("test-response\n"))

	session := &InitSession{
		Reader: bufio.NewReader(cmd.InOrStdin()),
		Writer: out,
	}

	question := newInputQuestion(questionKeyModel, "Model", "", false, nil)
	answer, err := askSingleQuestion(session, question)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if answer != "test-response" {
		t.Errorf("expected answer test-response, got %q", answer)
	}
}
