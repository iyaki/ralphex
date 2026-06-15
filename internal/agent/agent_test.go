package agent_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/iyaki/ralphex/internal/agent"
)

type synchronizedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *synchronizedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func writeExecutable(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write executable: %v", err)
	}

	return path
}

func testAgentExecutionStreamsOutputInRealTime(
	t *testing.T,
	commandName string,
	execute func(string, io.Writer) (string, error),
) {
	t.Helper()

	tmp := t.TempDir()
	writeExecutable(t, tmp, commandName, "#!/bin/sh\nprintf 'first\\n'; sleep 1; printf 'second\\n'\n")
	t.Setenv("PATH", tmp+":"+os.Getenv("PATH"))

	out := &synchronizedBuffer{}

	done := make(chan struct{})
	var (
		result string
		err    error
	)

	go func() {
		result, err = execute("prompt", out)
		close(done)
	}()

	assertFirstLineStreamedBeforeCompletion(t, out, done)
	waitForCommandCompletion(t, done)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "first") || !strings.Contains(result, "second") {
		t.Fatalf("expected complete output in result, got %q", result)
	}
}

func assertFirstLineStreamedBeforeCompletion(t *testing.T, out *synchronizedBuffer, done <-chan struct{}) {
	t.Helper()

	deadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), "first") {
			select {
			case <-done:
				t.Fatal("command completed before second line delay elapsed; output was not streamed in real time")
			default:
			}

			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected first line to be streamed before command completion, got %q", out.String())
}

func waitForCommandCompletion(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for command completion")
	}
}

func TestGetAgentReturnsExpectedType(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		expected  string
	}{
		{name: "omp", agentName: "omp", expected: "omp"},
		{name: "claude", agentName: "claude", expected: "claude"},
		{name: "cursor", agentName: "cursor", expected: "cursor"},
		{name: "opencode", agentName: "opencode", expected: "opencode"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, err := agent.GetAgent(tc.agentName, "model-x", "reviewer", nil)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if a.Name() != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, a.Name())
			}
		})
	}
}

func TestGetAgentReturnsErrorForUnknownConfiguredAgent(t *testing.T) {
	_, err := agent.GetAgent("unknown", "model-x", "reviewer", nil)
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(err.Error(), "unknown agent") {
		t.Fatalf("expected unknown agent error, got %v", err)
	}
}

func TestGetAgentCapturesEnvironmentSnapshot(t *testing.T) {
	tests := []struct {
		name      string
		agentName string
		command   string
	}{
		{name: "omp", agentName: "omp", command: "omp"},
		{name: "opencode", agentName: "opencode", command: "opencode"},
		{name: "claude", agentName: "claude", command: "claude"},
		{name: "cursor", agentName: "cursor", command: "cursor"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			writeExecutable(t, tmp, tc.command, "#!/bin/sh\nprintf 'SNAPSHOT:%s\\n' \"$SNAPSHOT\"\n")
			t.Setenv("PATH", tmp)

			env := []string{"SNAPSHOT=original"}
			a, err := agent.GetAgent(tc.agentName, "", "", env)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			env[0] = "SNAPSHOT=mutated"

			result, err := a.Execute("prompt", &bytes.Buffer{})
			if err != nil {
				t.Fatalf("expected successful execute, got %v", err)
			}
			if !strings.Contains(result, "SNAPSHOT:original") {
				t.Fatalf("expected original environment snapshot in result, got %q", result)
			}
			if strings.Contains(result, "SNAPSHOT:mutated") {
				t.Fatalf("did not expect mutated environment value in result, got %q", result)
			}
		})
	}
}

func TestBuildEffectiveEnvAppliesOverridesOnTopOfInheritedEnvironment(t *testing.T) {
	t.Setenv("INHERITED_ONLY", "from-parent")
	t.Setenv("OVERRIDE_ME", "from-parent")

	effectiveEnv, err := agent.BuildEffectiveEnv(map[string]string{
		"OVERRIDE_ME": "from-override",
		"COMPLEX":     "a=b=c",
	})
	if err != nil {
		t.Fatalf("expected no error building effective env, got %v", err)
	}

	parsed := parseEnvironmentEntries(t, effectiveEnv)
	if got := parsed["INHERITED_ONLY"]; got != "from-parent" {
		t.Fatalf("expected inherited env to be preserved, got %q", got)
	}
	if got := parsed["OVERRIDE_ME"]; got != "from-override" {
		t.Fatalf("expected override env value, got %q", got)
	}
	if got := parsed["COMPLEX"]; got != "a=b=c" {
		t.Fatalf("expected env value with '=' preserved, got %q", got)
	}

	assertEnvironmentEntriesAreSorted(t, effectiveEnv)
}

func TestBuildEffectiveEnvRejectsInvalidKeyWithoutLeakingValue(t *testing.T) {
	_, err := agent.BuildEffectiveEnv(map[string]string{
		"1INVALID": "super-secret-token",
	})
	if err == nil {
		t.Fatal("expected invalid env key error")
	}
	if !strings.Contains(err.Error(), "invalid environment key") {
		t.Fatalf("expected invalid key error, got %v", err)
	}
	if strings.Contains(err.Error(), "super-secret-token") {
		t.Fatalf("expected error to redact env value, got %v", err)
	}
}

func TestClaudeExecuteSuccessAndFailure(t *testing.T) {
	tmp := t.TempDir()
	claudeScript := "#!/bin/sh\n" +
		"echo \"out:$*\"\n" +
		"echo \"err:$*\" 1>&2\n" +
		"if [ \"$FAIL\" = \"1\" ]; then exit 1; fi\n"
	writeExecutable(t, tmp, "claude", claudeScript)
	t.Setenv("PATH", tmp)

	a := &agent.ClaudeAgent{Model: "m1", AgentMode: "planner"}
	var out bytes.Buffer
	result, err := a.Execute("hello", &out)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if !strings.Contains(result, "out:--dangerously-skip-permissions --model m1 --agent planner hello") {
		t.Fatalf("unexpected result: %q", result)
	}
	if !strings.Contains(out.String(), "err:") {
		t.Fatalf("expected stderr content in output writer: %q", out.String())
	}

	t.Setenv("FAIL", "1")
	_, err = a.Execute("hello", &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "claude execution failed") {
		t.Fatalf("expected wrapped error, got %v", err)
	}

	t.Setenv("PATH", t.TempDir())
	if a.IsAvailable() {
		t.Fatal("expected claude to be unavailable")
	}
}

func TestCursorExecuteAndAvailability(t *testing.T) {
	tmp := t.TempDir()
	writeExecutable(t, tmp, "cursor", "#!/bin/sh\necho \"cursor:$*\"\n")
	t.Setenv("PATH", tmp)

	a := &agent.CursorAgent{Model: "m2"}
	if !a.IsAvailable() {
		t.Fatal("expected cursor to be available")
	}
	if a.Name() != "cursor" {
		t.Fatalf("unexpected name: %s", a.Name())
	}

	result, err := a.Execute("prompt", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "cursor:--model m2 prompt") {
		t.Fatalf("unexpected result: %q", result)
	}

	t.Setenv("PATH", t.TempDir())
	if a.IsAvailable() {
		t.Fatal("expected cursor to be unavailable")
	}
}

func TestOmpExecuteAndAvailability(t *testing.T) {
	tmp := t.TempDir()
	writeExecutable(t, tmp, "omp", "#!/bin/sh\necho \"omp:$*\"\n")
	t.Setenv("PATH", tmp)

	a := &agent.OmpAgent{Model: "m4"}
	if !a.IsAvailable() {
		t.Fatal("expected omp to be available")
	}
	if a.Name() != "omp" {
		t.Fatalf("unexpected name: %s", a.Name())
	}

	result, err := a.Execute("prompt", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "omp:launch --print --model m4 prompt") {
		t.Fatalf("unexpected result: %q", result)
	}

	t.Setenv("PATH", t.TempDir())
	if a.IsAvailable() {
		t.Fatal("expected omp to be unavailable")
	}
}

func TestAllAgentsExecuteWithProvidedEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		executeFn func(prompt string, output io.Writer) (string, error)
	}{
		{
			name:    "omp",
			command: "omp",
			executeFn: (&agent.OmpAgent{
				Env: []string{"OVERRIDE_ME=from-agent"},
			}).Execute,
		},
		{
			name:    "opencode",
			command: "opencode",
			executeFn: (&agent.OpencodeAgent{
				Env: []string{"OVERRIDE_ME=from-agent"},
			}).Execute,
		},
		{
			name:    "claude",
			command: "claude",
			executeFn: (&agent.ClaudeAgent{
				Env: []string{"OVERRIDE_ME=from-agent"},
			}).Execute,
		},
		{
			name:    "cursor",
			command: "cursor",
			executeFn: (&agent.CursorAgent{
				Env: []string{"OVERRIDE_ME=from-agent"},
			}).Execute,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			writeExecutable(t, tmp, tc.command, "#!/bin/sh\nprintf 'OVERRIDE_ME:%s\\n' \"$OVERRIDE_ME\"\n")
			t.Setenv("PATH", tmp)

			result, err := tc.executeFn("prompt", &bytes.Buffer{})
			if err != nil {
				t.Fatalf("expected successful execute, got %v", err)
			}
			if !strings.Contains(result, "OVERRIDE_ME:from-agent") {
				t.Fatalf("expected provided environment override in result, got %q", result)
			}
		})
	}
}

func TestOpencodeExecuteStreamsOutputInRealTime(t *testing.T) {
	a := &agent.OpencodeAgent{}
	testAgentExecutionStreamsOutputInRealTime(t, "opencode", a.Execute)
}

func TestCursorExecuteStreamsOutputInRealTime(t *testing.T) {
	a := &agent.CursorAgent{}
	testAgentExecutionStreamsOutputInRealTime(t, "cursor", a.Execute)
}

func TestOmpExecuteStreamsOutputInRealTime(t *testing.T) {
	a := &agent.OmpAgent{}
	testAgentExecutionStreamsOutputInRealTime(t, "omp", a.Execute)
}

func parseEnvironmentEntries(t *testing.T, entries []string) map[string]string {
	t.Helper()

	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			t.Fatalf("expected KEY=VALUE environment entry, got %q", entry)
		}

		result[key] = value
	}

	return result
}

func assertEnvironmentEntriesAreSorted(t *testing.T, entries []string) {
	t.Helper()

	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			t.Fatalf("expected KEY=VALUE environment entry, got %q", entry)
		}

		keys = append(keys, key)
	}

	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)

	if !equalStringSlices(keys, sorted) {
		t.Fatalf("expected sorted environment keys, got %v", keys)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
