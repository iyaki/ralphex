package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/iyaki/ralphex/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var isInteractiveTerminal = func() bool {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	if (stdinInfo.Mode()&os.ModeCharDevice) == 0 || (stdoutInfo.Mode()&os.ModeCharDevice) == 0 {
		return false
	}

	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

var getWorkingDir = os.Getwd

const defaultInitMaxIterations = 25

const (
	defaultInitAgentName              = "opencode"
	defaultInitSpecsDir               = "specs"
	defaultInitSpecsIndexFile         = "README.md"
	defaultInitImplementationPlanName = "IMPLEMENTATION_PLAN.md"
	defaultInitPromptsDir             = ".ralph/prompts"
	defaultInitLogFile                = "./ralph.log"
	confirmYes                        = "yes"
	confirmNo                         = "no"
)

const (
	questionTypeSelect  = "select"
	questionTypeInput   = "input"
	questionTypeConfirm = "confirm"
)

const (
	questionKeyAgentName              = "agent"
	questionKeyModel                  = "model"
	questionKeyAgentMode              = "agent-mode"
	questionKeyMaxIterations          = "max-iterations"
	questionKeySpecsDir               = "specs-dir"
	questionKeySpecsIndexFile         = "specs-index-file"
	questionKeyImplementationPlanName = "implementation-plan-name"
	questionKeyPromptsDir             = "prompts-dir"
	questionKeyOverwriteExisting      = "overwrite-existing"
	questionKeyLogFile                = "log-file"
	questionKeyLogTruncate            = "log-truncate"
	questionKeyWriteConfiguration     = "write-configuration"
)

var supportedInitAgents = []string{"omp", "opencode", "claude", "cursor", "oh-my-pi"}

var errInvalidConfirmAnswer = errors.New("please answer yes or no")

var errInitValueRequired = errors.New("value cannot be empty")

// answerReader abstracts reading user input for testability.
type answerReader interface {
	ReadAnswer() (string, error)
}

// bufioAnswerReader wraps bufio.Reader to implement answerReader.
type bufioAnswerReader struct {
	reader *bufio.Reader
}

func (r *bufioAnswerReader) ReadAnswer() (string, error) {
	return readAnswer(r.reader)
}

// questionnaireRunner abstracts the questionnaire flow for testability.
type questionnaireRunner interface {
	AskQuestions(session *InitSession, questions []InitQuestion) error
}

// standardQuestionnaireRunner implements questionnaireRunner for production.
type standardQuestionnaireRunner struct{}

func (r *standardQuestionnaireRunner) AskQuestions(session *InitSession, questions []InitQuestion) error {
	return askQuestions(session, questions)
}

func runInitQuestionnaireWithRunner(session *InitSession, runner questionnaireRunner) error {
	session.Questions = baseInitQuestions(session.Answers)
	if err := runner.AskQuestions(session, session.Questions); err != nil {
		return err
	}

	if session.Answers.LogFile != "" {
		loggingQuestions := loggingInitQuestions(session.Answers)
		session.Questions = append(session.Questions, loggingQuestions...)

		if err := runner.AskQuestions(session, loggingQuestions); err != nil {
			return err
		}
	}

	return nil
}

func askQuestionsWithReader(session *InitSession, questions []InitQuestion, reader answerReader) error {
	for _, question := range questions {
		answer, err := promptForAnswerWithReader(session, question, reader)
		if err != nil {
			return err
		}

		if err := applyInitAnswer(session.Answers, question.Key, answer); err != nil {
			return err
		}
	}

	return nil
}

// InitSession represents one interactive run of ralph init.
type InitSession struct {
	OutputPath          string
	IsTTY               bool
	ExistingConfigFound bool
	Questions           []InitQuestion
	Answers             *InitAnswers
	Confirmed           bool
	Reader              *bufio.Reader
	Writer              io.Writer
}

// InitQuestion represents a single question in the interactive flow.
type InitQuestion struct {
	Key          string
	Prompt       string
	Type         string // "select", "input", "confirm"
	DefaultValue string
	Options      []string
	Required     bool
	Validator    func(string) error
}

// InitAnswers mirrors configuration fields for collection.
type InitAnswers struct {
	AgentName              string
	Model                  string
	AgentMode              string
	MaxIterations          int
	SpecsDir               string
	SpecsIndexFile         string
	ImplementationPlanName string
	PromptsDir             string
	LogFile                string
	LogTruncate            bool
}

type initAnswerApplier func(*InitAnswers, string) error

var initAnswerAppliers = map[string]initAnswerApplier{
	questionKeyAgentName:              setInitAnswerAgentName,
	questionKeyModel:                  setInitAnswerModel,
	questionKeyAgentMode:              setInitAnswerAgentMode,
	questionKeyMaxIterations:          setInitAnswerMaxIterations,
	questionKeySpecsDir:               setInitAnswerSpecsDir,
	questionKeySpecsIndexFile:         setInitAnswerSpecsIndexFile,
	questionKeyImplementationPlanName: setInitAnswerImplementationPlanName,
	questionKeyPromptsDir:             setInitAnswerPromptsDir,
	questionKeyLogFile:                setInitAnswerLogFile,
	questionKeyLogTruncate:            setInitAnswerLogTruncate,
}

// NewInitCommand creates the init command.
func NewInitCommand() *cobra.Command {
	var force bool
	var output string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Ralphex configuration",
		Long:  `Interactive command to generate a ralph.toml configuration file.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeInitCommand(cmd, output, force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config without prompt")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Target file path (default: ./ralph.toml)")

	return cmd
}

func executeInitCommand(cmd *cobra.Command, outputPath string, force bool) error {
	session, err := newInitSession(cmd, outputPath)
	if err != nil {
		return err
	}

	shouldContinue, err := prepareInitSession(session, force)
	if err != nil {
		return err
	}
	if !shouldContinue {
		return nil
	}

	if err := runInitQuestionnaire(session); err != nil {
		return err
	}

	shouldWrite, err := confirmInitWrite(session)
	if err != nil {
		return err
	}
	if !shouldWrite {
		return nil
	}

	if err := writeInitConfig(session); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(session.Writer, "Initialized Ralphex configuration at %s\n", session.OutputPath)

	return nil
}

func newInitSession(cmd *cobra.Command, outputPath string) (*InitSession, error) {
	session := &InitSession{
		OutputPath: outputPath,
		Answers:    defaultInitAnswers(),
		Reader:     bufio.NewReader(cmd.InOrStdin()),
		Writer:     cmd.OutOrStdout(),
	}

	session.IsTTY = isInteractiveTerminal()
	if !session.IsTTY {
		return nil, fmt.Errorf("ralph init requires an interactive terminal")
	}

	if session.OutputPath == "" {
		cwd, err := getWorkingDir()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve current working directory: %w", err)
		}

		session.OutputPath = filepath.Join(cwd, "ralph.toml")
	}

	return session, nil
}

func prepareInitSession(session *InitSession, force bool) (bool, error) {
	existingConfig, err := initConfigExists(session.OutputPath)
	if err != nil {
		return false, err
	}
	if !existingConfig {
		return true, nil
	}

	session.ExistingConfigFound = true
	seedInitDefaultsFromExistingConfig(session)

	if force {
		return true, nil
	}

	overwriteConfirmed, err := confirmExistingConfigOverwrite(session)
	if err != nil {
		return false, err
	}
	if !overwriteConfirmed {
		_, _ = fmt.Fprintln(session.Writer, "Initialization cancelled; existing configuration was not changed.")

		return false, nil
	}

	return true, nil
}

func seedInitDefaultsFromExistingConfig(session *InitSession) {
	existingConfig, meta, ok := loadExistingInitConfig(session.OutputPath)
	if !ok {
		return
	}

	seedInitAgentDefault(session.Answers, existingConfig.AgentName)
	seedInitMaxIterationsDefault(session.Answers, existingConfig.MaxIterations)
	seedInitStringDefaults(session.Answers, existingConfig)
	seedInitBoolDefaults(session.Answers, existingConfig, meta)
}

func loadExistingInitConfig(path string) (*config.Config, toml.MetaData, bool) {
	existingConfig := &config.Config{}
	meta, err := toml.DecodeFile(path, existingConfig)
	if err != nil {
		return nil, toml.MetaData{}, false
	}

	return existingConfig, meta, true
}

func seedInitAgentDefault(answers *InitAnswers, agentName string) {
	if validateInitAgent(agentName) == nil {
		answers.AgentName = agentName
	}
}

func seedInitMaxIterationsDefault(answers *InitAnswers, maxIterations int) {
	if maxIterations > 0 {
		answers.MaxIterations = maxIterations
	}
}

func seedInitStringDefaults(answers *InitAnswers, existingConfig *config.Config) {
	for _, field := range []struct {
		value string
		apply func(string)
	}{
		{existingConfig.Model, func(value string) { answers.Model = value }},
		// AgentMode intentionally omitted - always shows no default
		{existingConfig.SpecsDir, func(value string) { answers.SpecsDir = value }},
		{existingConfig.SpecsIndexFile, func(value string) { answers.SpecsIndexFile = value }},
		{existingConfig.ImplementationPlanName, func(value string) { answers.ImplementationPlanName = value }},
		{existingConfig.PromptsDir, func(value string) { answers.PromptsDir = value }},
		// LogFile intentionally omitted - always shows no default
	} {
		if strings.TrimSpace(field.value) == "" {
			continue
		}

		field.apply(field.value)
	}
}

func seedInitBoolDefaults(answers *InitAnswers, existingConfig *config.Config, meta toml.MetaData) {
	if meta.IsDefined("log-truncate") {
		answers.LogTruncate = existingConfig.LogTruncate
	}
}

func initConfigExists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("failed to inspect existing configuration at %s: %w", path, err)
	}

	return false, nil
}

func writeInitConfig(session *InitSession) error {
	cfg := buildConfigFromAnswers(session.Answers)
	if err := config.WriteConfig(session.OutputPath, cfg); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	return nil
}
func confirmExistingConfigOverwrite(session *InitSession) (bool, error) {
	return confirmExistingConfigOverwriteWithReader(session, &bufioAnswerReader{reader: session.Reader})
}

func confirmExistingConfigOverwriteWithReader(session *InitSession, reader answerReader) (bool, error) {
	answer, err := promptForAnswerWithReader(session, newConfirmQuestion(
		questionKeyOverwriteExisting,
		"Overwrite existing configuration?",
		confirmNo,
	), reader)
	if err != nil {
		return false, err
	}

	confirmed, _ := parseConfirmAnswer(answer)

	return confirmed, nil
}

func confirmInitWrite(session *InitSession) (bool, error) {
	return confirmInitWriteWithReader(session, &bufioAnswerReader{reader: session.Reader})
}

func confirmInitWriteWithReader(session *InitSession, reader answerReader) (bool, error) {
	if err := printInitPreview(session); err != nil {
		return false, err
	}

	answer, err := promptForAnswerWithReader(session, newConfirmQuestion(
		questionKeyWriteConfiguration,
		"Write configuration now?",
		confirmYes,
	), reader)
	if err != nil {
		return false, err
	}

	confirmed, _ := parseConfirmAnswer(answer)
	session.Confirmed = confirmed

	if !confirmed {
		_, _ = fmt.Fprintln(session.Writer, "Initialization cancelled; configuration was not written.")
	}

	return confirmed, nil
}

func printInitPreview(session *InitSession) error {
	if _, err := fmt.Fprintln(session.Writer, "Configuration preview:"); err != nil {
		return err
	}

	for _, line := range buildInitPreviewLines(session) {
		if _, err := fmt.Fprintf(session.Writer, "  %s\n", line); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(session.Writer); err != nil {
		return err
	}

	return nil
}

func buildInitPreviewLines(session *InitSession) []string {
	answers := session.Answers

	lines := []string{
		fmt.Sprintf("output path: %s", session.OutputPath),
		fmt.Sprintf("agent: %s", answers.AgentName),
	}

	if model := strings.TrimSpace(answers.Model); model != "" {
		lines = append(lines, fmt.Sprintf("model: %s", model))
	}
	if agentMode := strings.TrimSpace(answers.AgentMode); agentMode != "" {
		lines = append(lines, fmt.Sprintf("agent-mode: %s", agentMode))
	}

	lines = append(lines,
		fmt.Sprintf("max-iterations: %d", answers.MaxIterations),
		fmt.Sprintf("specs-dir: %s", answers.SpecsDir),
		fmt.Sprintf("specs-index-file: %s", answers.SpecsIndexFile),
		fmt.Sprintf("implementation-plan-name: %s", answers.ImplementationPlanName),
		fmt.Sprintf("prompts-dir: %s", answers.PromptsDir),
	)

	if answers.LogFile == "" {
		return append(lines, "logging: disabled")
	}

	return append(lines,
		"logging: enabled",
		fmt.Sprintf("log-file: %s", answers.LogFile),
		fmt.Sprintf("log-truncate: %s", boolToConfirmValue(answers.LogTruncate)),
	)
}

func defaultInitAnswers() *InitAnswers {
	return &InitAnswers{
		AgentName:              defaultInitAgentName,
		MaxIterations:          defaultInitMaxIterations,
		SpecsDir:               defaultInitSpecsDir,
		SpecsIndexFile:         defaultInitSpecsIndexFile,
		ImplementationPlanName: defaultInitImplementationPlanName,
		PromptsDir:             defaultInitPromptsDir,
		LogFile:                "",
		LogTruncate:            false,
	}
}

func runInitQuestionnaire(session *InitSession) error {
	session.Questions = baseInitQuestions(session.Answers)
	if err := askQuestions(session, session.Questions); err != nil {
		return err
	}

	if session.Answers.LogFile != "" {
		loggingQuestions := loggingInitQuestions(session.Answers)
		session.Questions = append(session.Questions, loggingQuestions...)

		if err := askQuestions(session, loggingQuestions); err != nil {
			return err
		}
	}

	return nil
}

func baseInitQuestions(defaults *InitAnswers) []InitQuestion {
	return []InitQuestion{
		newSelectQuestion(
			questionKeyAgentName,
			"AI agent (omp/opencode/claude/cursor/oh-my-pi)",
			defaults.AgentName,
			supportedInitAgents,
			validateInitAgent,
		),
		newInputQuestion(questionKeyModel, "Model (optional)", defaults.Model, false, nil),
		newInputQuestion(questionKeyAgentMode, "Agent mode/sub-agent (optional)", defaults.AgentMode, false, nil),
		newInputQuestion(
			questionKeyMaxIterations,
			"Maximum iterations",
			strconv.Itoa(defaults.MaxIterations),
			true,
			validatePositiveInitInteger,
		),
		newInputQuestion(questionKeySpecsDir, "Specs directory", defaults.SpecsDir, true, nil),
		newInputQuestion(questionKeySpecsIndexFile, "Specs index file", defaults.SpecsIndexFile, true, nil),
		newInputQuestion(
			questionKeyImplementationPlanName,
			"Implementation plan file",
			defaults.ImplementationPlanName,
			true,
			nil,
		),
		newInputQuestion(questionKeyPromptsDir, "Prompts directory", defaults.PromptsDir, true, nil),
		newInputQuestion(questionKeyLogFile, "Log file path (leave empty to disable logging)", defaults.LogFile, false, nil),
	}
}

func loggingInitQuestions(defaults *InitAnswers) []InitQuestion {
	return []InitQuestion{
		newConfirmQuestion(
			questionKeyLogTruncate,
			"Truncate log file on each run?",
			boolToConfirmValue(defaults.LogTruncate),
		),
	}
}

func newSelectQuestion(
	key, prompt, defaultValue string,
	options []string,
	validator func(string) error,
) InitQuestion {
	return InitQuestion{
		Key:          key,
		Prompt:       prompt,
		Type:         questionTypeSelect,
		DefaultValue: defaultValue,
		Options:      options,
		Required:     true,
		Validator:    validator,
	}
}

func newInputQuestion(
	key, prompt, defaultValue string,
	required bool,
	validator func(string) error,
) InitQuestion {
	return InitQuestion{
		Key:          key,
		Prompt:       prompt,
		Type:         questionTypeInput,
		DefaultValue: defaultValue,
		Required:     required,
		Validator:    validator,
	}
}

func newConfirmQuestion(key, prompt, defaultValue string) InitQuestion {
	return InitQuestion{
		Key:          key,
		Prompt:       prompt,
		Type:         questionTypeConfirm,
		DefaultValue: defaultValue,
	}
}

func askQuestions(session *InitSession, questions []InitQuestion) error {
	for _, question := range questions {
		answer, err := promptForAnswer(session, question)
		if err != nil {
			return err
		}

		if err := applyInitAnswer(session.Answers, question.Key, answer); err != nil {
			return err
		}
	}

	return nil
}

func promptForAnswer(session *InitSession, question InitQuestion) (string, error) {
	return promptForAnswerWithReader(session, question, &bufioAnswerReader{reader: session.Reader})
}

func promptForAnswerWithReader(session *InitSession, question InitQuestion, reader answerReader) (string, error) {
	for {
		answer, err := askSingleQuestionWithReader(session, question, reader)
		if err != nil {
			return "", err
		}

		if validationErr := validateQuestionAnswer(question, answer); validationErr != nil {
			if _, writeErr := fmt.Fprintln(session.Writer, validationErr.Error()); writeErr != nil {
				return "", writeErr
			}

			continue
		}

		return answer, nil
	}
}
func askSingleQuestion(session *InitSession, question InitQuestion) (string, error) {
	return askSingleQuestionWithReader(session, question, &bufioAnswerReader{reader: session.Reader})
}

func askSingleQuestionWithReader(session *InitSession, question InitQuestion, reader answerReader) (string, error) {
	if err := printQuestion(session.Writer, question); err != nil {
		return "", err
	}

	answer, err := reader.ReadAnswer()
	if err != nil {
		return "", normalizeAnswerReadError(err)
	}

	if answer == "" {
		return question.DefaultValue, nil
	}

	return answer, nil
}

func normalizeAnswerReadError(err error) error {
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("unexpected end of input during init questionnaire")
	}

	return err
}

func validateQuestionAnswer(question InitQuestion, answer string) error {
	if question.Required && strings.TrimSpace(answer) == "" {
		return errInitValueRequired
	}

	if question.Type == questionTypeConfirm {
		if _, ok := parseConfirmAnswer(answer); !ok {
			return errInvalidConfirmAnswer
		}

		return nil
	}

	if question.Validator != nil {
		return question.Validator(answer)
	}

	return nil
}

func printQuestion(writer io.Writer, question InitQuestion) error {
	if question.DefaultValue == "" {
		_, err := fmt.Fprintf(writer, "%s: ", question.Prompt)

		return err
	}

	_, err := fmt.Fprintf(writer, "%s [%s]: ", question.Prompt, question.DefaultValue)

	return err
}

func readAnswer(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if errors.Is(err, io.EOF) {
		if line == "" {
			return "", io.EOF
		}

		return strings.TrimSpace(line), nil
	}
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}

func applyInitAnswer(answers *InitAnswers, key, value string) error {
	applier, ok := initAnswerAppliers[key]
	if !ok {
		return fmt.Errorf("unknown init question key: %s", key)
	}

	return applier(answers, value)
}

func setInitAnswerAgentName(answers *InitAnswers, value string) error {
	answers.AgentName = value

	return nil
}

func setInitAnswerModel(answers *InitAnswers, value string) error {
	answers.Model = value

	return nil
}

func setInitAnswerAgentMode(answers *InitAnswers, value string) error {
	answers.AgentMode = value

	return nil
}

func setInitAnswerMaxIterations(answers *InitAnswers, value string) error {
	maxIterations, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid max-iterations answer: %w", err)
	}

	answers.MaxIterations = maxIterations

	return nil
}

func setInitAnswerSpecsDir(answers *InitAnswers, value string) error {
	answers.SpecsDir = value

	return nil
}

func setInitAnswerSpecsIndexFile(answers *InitAnswers, value string) error {
	answers.SpecsIndexFile = value

	return nil
}

func setInitAnswerImplementationPlanName(answers *InitAnswers, value string) error {
	answers.ImplementationPlanName = value

	return nil
}

func setInitAnswerPromptsDir(answers *InitAnswers, value string) error {
	answers.PromptsDir = value

	return nil
}

func setInitAnswerLogFile(answers *InitAnswers, value string) error {
	answers.LogFile = value

	return nil
}

func setInitAnswerLogTruncate(answers *InitAnswers, value string) error {
	logTruncate, ok := parseConfirmAnswer(value)
	if !ok {
		return errInvalidConfirmAnswer
	}

	answers.LogTruncate = logTruncate

	return nil
}

func parseConfirmAnswer(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes", "true", "1":
		return true, true
	case "n", "no", "false", "0":
		return false, true
	default:
		return false, false
	}
}

func boolToConfirmValue(value bool) string {
	if value {
		return confirmYes
	}

	return confirmNo
}

func validateInitAgent(value string) error {
	for _, agentName := range supportedInitAgents {
		if value == agentName {
			return nil
		}
	}

	return fmt.Errorf("must be one of: %s", strings.Join(supportedInitAgents, ", "))
}

func validatePositiveInitInteger(value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fmt.Errorf("must be an integer greater than 0")
	}

	return nil
}

func buildConfigFromAnswers(answers *InitAnswers) *config.Config {
	return &config.Config{
		AgentName:              answers.AgentName,
		Model:                  answers.Model,
		AgentMode:              answers.AgentMode,
		MaxIterations:          answers.MaxIterations,
		SpecsDir:               answers.SpecsDir,
		SpecsIndexFile:         answers.SpecsIndexFile,
		ImplementationPlanName: answers.ImplementationPlanName,
		PromptsDir:             answers.PromptsDir,
		LogFile:                answers.LogFile,
		LogTruncate:            answers.LogTruncate,
	}
}

// ExecuteInitCommandForTest exports executeInitCommand for testing.
func ExecuteInitCommandForTest(cmd *cobra.Command, outputPath string, force bool) error {
	return executeInitCommand(cmd, outputPath, force)
}

// SetIsInteractiveTerminalForTest overrides isInteractiveTerminal for testing.
func SetIsInteractiveTerminalForTest(value bool) {
	isInteractiveTerminal = func() bool { return value }
}

// GetIsInteractiveTerminalForTest returns the current isInteractiveTerminal function for testing.
func GetIsInteractiveTerminalForTest() bool {
	return isInteractiveTerminal()
}

// BoolFlagOverride holds the result of reading a bool flag.
type BoolFlagOverride struct {
	Changed bool
	Value   bool
}

// ReadBoolFlagOverrideForTest exports readBoolFlagOverride for testing.
func ReadBoolFlagOverrideForTest(cmd *cobra.Command, flagName string) (BoolFlagOverride, error) {
	result, err := readBoolFlagOverride(cmd, flagName)

	return BoolFlagOverride{Changed: result.changed, Value: result.value}, err
}

// ReadEnvFlagOverridesForTest exports readEnvFlagOverrides for testing.
func ReadEnvFlagOverridesForTest(cmd *cobra.Command) (map[string]string, error) {
	return readEnvFlagOverrides(cmd)
}
