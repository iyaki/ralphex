package logger_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iyaki/ralphex/internal/config"
	"github.com/iyaki/ralphex/internal/logger"
)

func TestNewLoggerDisabledWhenLogFileEmpty(t *testing.T) {
	cfg := &config.Config{LogFile: ""}
	l, err := logger.NewLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.Enabled() {
		t.Fatal("expected logger to be disabled when LogFile is empty")
	}
	if l.File() != nil {
		t.Fatal("expected no file when disabled")
	}
}

func TestNewLoggerCreatesAndAppendsFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "ralph.log")
	cfg := &config.Config{LogFile: logPath, LogTruncate: false}

	l, err := logger.NewLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !l.Enabled() || l.File() == nil {
		t.Fatal("expected logger to be enabled with file")
	}
	if err := l.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "Ralphex run started") {
		t.Fatalf("expected log header in file, got %q", string(content))
	}
	if !strings.Contains(string(content), "Git branch:") {
		t.Fatalf("expected git branch line, got %q", string(content))
	}
}

func TestNewLoggerTruncatesWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "ralph.log")
	if err := os.WriteFile(logPath, []byte("old-content\n"), 0600); err != nil {
		t.Fatalf("failed to seed log file: %v", err)
	}

	cfg := &config.Config{LogFile: logPath, LogTruncate: true}

	l, err := logger.NewLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if strings.Contains(string(content), "old-content") {
		t.Fatalf("expected old content to be truncated, got %q", string(content))
	}
}

func TestNewLoggerTruncateCreatesSecureFilePermissions(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "new-ralph.log")

	cfg := &config.Config{LogFile: logPath, LogTruncate: true}

	l, err := logger.NewLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("failed to stat log file: %v", err)
	}

	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected log permissions 0600, got %04o", got)
	}
}

func TestCloseWithoutFile(t *testing.T) {
	l := &logger.Logger{}
	if err := l.Close(); err != nil {
		t.Fatalf("expected nil error for close without file, got %v", err)
	}
}
