package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunCompleteOnceMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"ralph-test-agent"}
	getEnv := func(_ string) string {
		return ""
	}

	code := run(args, getEnv, &stdout, &stderr)
	if code != exitCodeSuccess {
		t.Fatalf("expected exit code %d, got %d", exitCodeSuccess, code)
	}
	if !strings.Contains(stdout.String(), "<promise>COMPLETE</promise>") {
		t.Fatal("expected complete signal in output")
	}
}

func TestRunNeverCompleteMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"ralph-test-agent"}
	getEnv := func(_ string) string {
		return modeNeverComplete
	}

	code := run(args, getEnv, &stdout, &stderr)
	if code != exitCodeSuccess {
		t.Fatalf("expected exit code %d, got %d", exitCodeSuccess, code)
	}
	if strings.Contains(stdout.String(), "<promise>COMPLETE</promise>") {
		t.Fatal("did not expect complete signal in output")
	}
}

func TestRunReturnErrorMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"ralph-test-agent"}
	getEnv := func(_ string) string {
		return modeReturnError
	}

	code := run(args, getEnv, &stdout, &stderr)
	if code != exitCodeError {
		t.Fatalf("expected exit code %d, got %d", exitCodeError, code)
	}
	if !strings.Contains(stderr.String(), "Simulated agent failure") {
		t.Fatal("expected error message in stderr")
	}
}

func TestRunSlowCompleteMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"ralph-test-agent"}
	getEnv := func(_ string) string {
		return modeSlowComplete
	}

	code := run(args, getEnv, &stdout, &stderr)
	if code != exitCodeSuccess {
		t.Fatalf("expected exit code %d, got %d", exitCodeSuccess, code)
	}
	if !strings.Contains(stdout.String(), "<promise>COMPLETE</promise>") {
		t.Fatal("expected complete signal in output")
	}
}

func TestRunUnknownMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"ralph-test-agent"}
	getEnv := func(_ string) string {
		return "unknown_mode_xyz"
	}

	code := run(args, getEnv, &stdout, &stderr)
	if code != exitCodeUnknown {
		t.Fatalf("expected exit code %d, got %d", exitCodeUnknown, code)
	}
}

func TestEmitRequestedEnv(t *testing.T) {
	var stderr bytes.Buffer
	getEnv := func(key string) string {
		if key == "RALPH_TEST_AGENT_ECHO_ENV_KEYS" {
			return "KEY1,KEY2"
		}
		if key == "KEY1" {
			return "value1"
		}
		if key == "KEY2" {
			return "value2"
		}

		return ""
	}

	emitRequestedEnv(getEnv, &stderr)
	output := stderr.String()
	if !strings.Contains(output, "KEY1=value1") {
		t.Fatal("expected KEY1 in output")
	}
	if !strings.Contains(output, "KEY2=value2") {
		t.Fatal("expected KEY2 in output")
	}
}

func TestEmitRequestedEnvEmpty(t *testing.T) {
	var stderr bytes.Buffer
	getEnv := func(_ string) string {
		return ""
	}

	emitRequestedEnv(getEnv, &stderr)
	if stderr.Len() != 0 {
		t.Fatal("expected no output for empty env keys")
	}
}

func TestEmitRequestedEnvWithSpaces(t *testing.T) {
	var stderr bytes.Buffer
	getEnv := func(key string) string {
		if key == "RALPH_TEST_AGENT_ECHO_ENV_KEYS" {
			return "  KEY1  ,  ,  KEY2  "
		}
		if key == "KEY1" {
			return "v1"
		}
		if key == "KEY2" {
			return "v2"
		}

		return ""
	}

	emitRequestedEnv(getEnv, &stderr)
	output := stderr.String()
	if !strings.Contains(output, "KEY1=v1") {
		t.Fatal("expected KEY1 in output")
	}
	if !strings.Contains(output, "KEY2=v2") {
		t.Fatal("expected KEY2 in output")
	}
}
