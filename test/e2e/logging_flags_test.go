package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ELoggingFlags(t *testing.T) {
	testCases := []struct {
		name string
		tc   TestCase
	}{
		{name: "DefaultNoLog", tc: loggingDefaultNoLogCase()},
		{name: "EnabledViaFlag", tc: loggingEnabledViaFlagCase()},
		{name: "EnabledViaConfig", tc: loggingEnabledViaConfigCase()},
		{name: "FlagOverridesConfig", tc: loggingFlagOverridesConfigCase()},
		{name: "LogTruncate", tc: loggingTruncateCase()},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			runTestCase(t, testCase.tc)
		})
	}
}

func loggingDefaultNoLogCase() TestCase {
	return TestCase{
		Name: "Logging: Disabled by default",
		Args: []string{"--prompt-file", "prompt.txt"},
		Env: map[string]string{
			"RALPH_TEST_AGENT_MODE": "complete_once",
		},
		Files: map[string]string{
			"prompt.txt": "Just a simple prompt",
		},
		ExpectedExitCode: 0,
		ForbiddenFiles:   []string{"ralph.log"},
	}
}

func loggingEnabledViaFlagCase() TestCase {
	return TestCase{
		Name: "Logging: Enabled via --log-file flag",
		Args: []string{"--log-file", "ralph.log", "--prompt-file", "prompt.txt"},
		Env: map[string]string{
			"RALPH_TEST_AGENT_MODE": "complete_once",
		},
		Files: map[string]string{
			"prompt.txt": "Just a simple prompt",
		},
		ExpectedExitCode: 0,
		ExpectedFiles:    []string{"ralph.log"},
	}
}

func loggingEnabledViaConfigCase() TestCase {
	return TestCase{
		Name: "Logging: Enabled via config log-file",
		Args: []string{
			"--config", "ralph.toml",
			"--prompt-file", "prompt.txt",
		},
		Env: map[string]string{
			"RALPH_TEST_AGENT_MODE": "complete_once",
		},
		Files: map[string]string{
			"prompt.txt": "Just a simple prompt",
			"ralph.toml": `log-file = "ralph.log"
`,
		},
		ExpectedExitCode: 0,
		ExpectedFiles:    []string{"ralph.log"},
	}
}

func loggingFlagOverridesConfigCase() TestCase {
	return TestCase{
		Name: "Logging: --log-file overrides missing config log-file",
		Args: []string{
			"--config", "ralph.toml",
			"--prompt-file", "prompt.txt",
			"--log-file", "ralph.log",
		},
		Env: map[string]string{
			"RALPH_TEST_AGENT_MODE": "complete_once",
		},
		Files: map[string]string{
			"prompt.txt": "Just a simple prompt",
			"ralph.toml": "# no log-file set\n",
		},
		ExpectedExitCode: 0,
		ExpectedFiles:    []string{"ralph.log"},
	}
}

func loggingTruncateCase() TestCase {
	return TestCase{
		Name: "Logging: Truncate with --log-truncate",
		Args: []string{"--log-file", "ralph.log", "--log-truncate", "--prompt-file", "prompt.txt"},
		Env: map[string]string{
			"RALPH_TEST_AGENT_MODE": "complete_once",
		},
		Files: map[string]string{
			"prompt.txt": "Just a simple prompt",
			"ralph.log":  "OLD LOG CONTENT",
		},
		ExpectedExitCode: 0,
		ExpectedFiles:    []string{"ralph.log"},
		ExpectedFileContent: map[string][]string{
			"ralph.log": {"===== Ralphex run started at"},
		},
		ForbiddenFileContent: map[string][]string{
			"ralph.log": {"OLD LOG CONTENT"},
		},
	}
}

func TestE2ELoggingPermissions(t *testing.T) {
	tc := TestCase{
		Name: "Logging: File permissions are 0600",
		Args: []string{"--log-file", "ralph.log", "--prompt-file", "prompt.txt"},
		Env: map[string]string{
			"RALPH_TEST_AGENT_MODE": "complete_once",
		},
		Files: map[string]string{
			"prompt.txt": "Just a simple prompt",
		},
		ExpectedExitCode: 0,
		ExpectedFiles:    []string{"ralph.log"},
	}

	workDir := prepareTestEnv(t, tc)
	res := executeRalph(t, workDir, tc)
	verifyResult(t, workDir, tc, res)

	logInfo, err := os.Stat(filepath.Join(workDir, "ralph.log"))
	if err != nil {
		t.Fatalf("failed to stat log file: %v", err)
	}

	if got := logInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected log file mode 0600, got %04o", got)
	}
}

func TestE2ELoggingStdoutParity(t *testing.T) {
	tc := TestCase{
		Name: "Logging: Log file contains stdout stream",
		Args: []string{"--log-file", "ralph.log", "--prompt-file", "prompt.txt"},
		Env: map[string]string{
			"RALPH_TEST_AGENT_MODE": "complete_once",
		},
		Files: map[string]string{
			"prompt.txt": "Just a simple prompt",
		},
		ExpectedExitCode: 0,
		ExpectedFiles:    []string{"ralph.log"},
		ExpectedStdoutContains: []string{
			"Starting Ralphex - Max iterations",
			"Using agent: opencode",
			"<promise>COMPLETE</promise>",
		},
	}

	workDir := prepareTestEnv(t, tc)
	res := executeRalph(t, workDir, tc)
	verifyResult(t, workDir, tc, res)

	logContent, err := os.ReadFile(filepath.Join(workDir, "ralph.log"))
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(logContent), res.Stdout) {
		t.Fatalf("expected log file to contain full stdout stream")
	}
}
