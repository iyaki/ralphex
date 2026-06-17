package prompt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/iyaki/ralphex/internal/config"
)

type Config = config.Config

func TestFindFileUpwardsInternal(t *testing.T) {
	t.Run("absolute path exists returns path", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result := findFileUpwards(testFile)
		if result != testFile {
			t.Errorf("expected %q, got %q", testFile, result)
		}
	})

	t.Run("absolute path missing returns empty", func(t *testing.T) {
		result := findFileUpwards("/nonexistent/file.txt")
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("relative file in cwd returns found path", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := "test.txt"
		fullPath := filepath.Join(tmpDir, testFile)
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		origDir, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()

		result := findFileUpwards(testFile)
		if result != fullPath {
			t.Errorf("expected %q, got %q", fullPath, result)
		}
	})

	t.Run("file nowhere returns empty", func(t *testing.T) {
		result := findFileUpwards("nonexistent-file-12345.txt")
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestCustomPromptInternal(t *testing.T) {
	cfg := &Config{CustomPrompt: "test custom prompt"}
	var out bytes.Buffer

	result, used, err := customPrompt(cfg, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !used {
		t.Fatal("expected custom prompt to be used")
	}
	if result != "test custom prompt" {
		t.Fatalf("expected 'test custom prompt', got %q", result)
	}
	if !bytes.Contains(out.Bytes(), []byte("INLINE CUSTOM PROMPT")) {
		t.Fatal("expected banner in output")
	}
}

func TestCustomPromptEmpty(t *testing.T) {
	cfg := &Config{CustomPrompt: ""}
	var out bytes.Buffer

	_, used, err := customPrompt(cfg, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used {
		t.Fatal("expected custom prompt to not be used")
	}
}

func TestStdinPrompt(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
		_ = w.Close()
	}()

	go func() {
		_, _ = w.WriteString("stdin prompt content")
		_ = w.Close()
	}()

	cfg := &Config{PromptFile: "-"}
	var out bytes.Buffer

	result, used, err := stdinPrompt(cfg, "test", &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !used {
		t.Fatal("expected stdin prompt to be used")
	}
	if result != "stdin prompt content" {
		t.Fatalf("expected 'stdin prompt content', got %q", result)
	}
}

func TestStdinPromptWithoutDash(t *testing.T) {
	cfg := &Config{PromptFile: "test.md"}
	var out bytes.Buffer

	_, used, err := stdinPrompt(cfg, "test", &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used {
		t.Fatal("expected stdin prompt to not be used")
	}
}

func TestStdinPromptReadError(t *testing.T) {
	// This is difficult to test reliably as io.ReadAll may not fail on a closed pipe
	// We skip this edge case for now
	t.Skip("stdin read error test skipped - hard to trigger reliable error")
}

func TestExplicitPromptFile(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "test.md")
	content := []byte("---\nmodel: test-model\n---\nTest prompt body")
	if err := os.WriteFile(promptFile, content, 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	cfg := &Config{PromptFile: promptFile}
	var out bytes.Buffer

	body, override, used, err := explicitPromptFile(cfg, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !used {
		t.Fatal("expected explicit prompt to be used")
	}
	if body != "Test prompt body" {
		t.Fatalf("expected 'Test prompt body', got %q", body)
	}
	if override == nil || override.Model != "test-model" {
		t.Fatal("expected model override")
	}
}

func TestExplicitPromptFileNotFound(t *testing.T) {
	cfg := &Config{PromptFile: "/nonexistent/file.md"}
	var out bytes.Buffer

	body, _, used, err := explicitPromptFile(cfg, &out)
	if body != "" && used {
		t.Fatal("did not expect file to be found")
	}
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestPromptFromDir(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "test.md")
	content := []byte("---\nmodel: dir-model\n---\nDir prompt body")
	if err := os.WriteFile(promptFile, content, 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	cfg := &Config{PromptsDir: tmpDir}
	var out bytes.Buffer

	body, override, used, err := promptFromDir(cfg, "test", &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !used {
		t.Fatal("expected prompt from dir to be used")
	}
	if body != "Dir prompt body" {
		t.Fatalf("expected 'Dir prompt body', got %q", body)
	}
	if override == nil || override.Model != "dir-model" {
		t.Fatal("expected model override")
	}
}

func TestPromptFromDirNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{PromptsDir: tmpDir}
	var out bytes.Buffer

	_, _, used, err := promptFromDir(cfg, "nonexistent", &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used {
		t.Fatal("expected prompt from dir to not be used")
	}
}

func TestFindFileUpwardsRelativeInParent(t *testing.T) {
	// Create a nested directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create file in parent
	parentFile := filepath.Join(tmpDir, "parent.txt")
	if err := os.WriteFile(parentFile, []byte("parent"), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to sub directory
	oldDir, _ := os.Getwd()
	if err := os.Chdir(subDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	// Search upwards for parent.txt
	result := findFileUpwards("parent.txt")
	if result != parentFile {
		t.Errorf("expected %q, got %q", parentFile, result)
	}
}
