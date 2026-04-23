package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCommandRejectsConflictingFormats(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"parse", "input.md", "--json", "--markdown"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected an error for conflicting format flags")
	}

	if !strings.Contains(err.Error(), "--json and --markdown cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCommandMarkdownImportModeLink(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "input.md")

	if err := os.WriteFile(mainPath, []byte("# Root\n\n```python\n# import(./missing.py)\n```\n"), 0o644); err != nil {
		t.Fatalf("write input markdown: %v", err)
	}

	cmd := NewRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"parse", mainPath, "--markdown", "--import-mode=link"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	expected := "# Root\n\n[python](./missing.py)\n\n"
	if output.String() != expected {
		t.Fatalf("unexpected output:\nexpected:\n%s\ngot:\n%s", expected, output.String())
	}
}

func TestParseCommandMarkdownImportModeEmbed(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "input.md")
	codePath := filepath.Join(tempDir, "hello.py")

	if err := os.WriteFile(codePath, []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("write code file: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("# Root\n\n```\n# import(./hello.py)\n```\n"), 0o644); err != nil {
		t.Fatalf("write input markdown: %v", err)
	}

	cmd := NewRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"parse", mainPath, "--markdown", "--import-mode=embed"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	expected := "# Root\n\n```\nprint('hello')\n```\n\n"
	if output.String() != expected {
		t.Fatalf("unexpected output:\nexpected:\n%s\ngot:\n%s", expected, output.String())
	}
}

func TestParseCommandRejectsInvalidImportMode(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "input.md")

	if err := os.WriteFile(mainPath, []byte("# Root\n"), 0o644); err != nil {
		t.Fatalf("write input markdown: %v", err)
	}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"parse", mainPath, "--markdown", "--import-mode=unknown"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected an error for invalid import mode")
	}

	if !strings.Contains(err.Error(), `invalid import mode "unknown"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCommandJSONIgnoresImportMode(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "input.md")
	codePath := filepath.Join(tempDir, "hello.py")

	if err := os.WriteFile(codePath, []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("write code file: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("# Root\n\n## Script\n\n```\n# import(./hello.py)\n```\n"), 0o644); err != nil {
		t.Fatalf("write input markdown: %v", err)
	}

	cmd := NewRootCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"parse", mainPath, "--json", "--import-mode=link"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(output.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	body, ok := parsed["body"].(map[string]any)
	if !ok {
		t.Fatalf("body missing: %#v", parsed["body"])
	}
	root, ok := body["Root"].(map[string]any)
	if !ok {
		t.Fatalf("Root missing: %#v", body["Root"])
	}
	script, ok := root["Script"].(map[string]any)
	if !ok {
		t.Fatalf("Script missing: %#v", root["Script"])
	}

	if script["code"] != "print('hello')" {
		t.Fatalf("unexpected imported code: %#v", script["code"])
	}
}
