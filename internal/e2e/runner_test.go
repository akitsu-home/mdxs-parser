package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithExecutor_PassAndFailCases(t *testing.T) {
	t.Parallel()

	specDir := t.TempDir()
	specPath := filepath.Join(specDir, "spec.md")
	spec := `# pass case

## command

- cmd-pass

## stdout contains

- hello

## stderr not contains

- panic

# fail case

## command

- cmd-fail

## exit code

1

## stdout not contains

- denied
`
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	execFn := func(_ context.Context, command []string, _ string) (commandResult, error) {
		switch strings.Join(command, " ") {
		case "cmd-pass":
			return commandResult{stdout: "hello world", stderr: "", exitCode: 0}, nil
		case "cmd-fail":
			return commandResult{stdout: "access denied", stderr: "", exitCode: 1}, nil
		default:
			return commandResult{}, fmt.Errorf("unexpected command: %v", command)
		}
	}

	var output bytes.Buffer
	err := runWithExecutor(context.Background(), specDir, &output, execFn)
	if err == nil {
		t.Fatal("expected failure when one case fails")
	}
	if !strings.Contains(err.Error(), "1 e2e case(s) failed") {
		t.Fatalf("unexpected error: %v", err)
	}

	report := output.String()
	if !strings.Contains(report, "PASS") {
		t.Fatalf("expected PASS in output: %s", report)
	}
	if !strings.Contains(report, "FAIL") {
		t.Fatalf("expected FAIL in output: %s", report)
	}
	if !strings.Contains(report, `stdout unexpectedly contains "denied"`) {
		t.Fatalf("expected assertion detail in output: %s", report)
	}
	if !strings.Contains(report, "Summary: total=2 passed=1 failed=1") {
		t.Fatalf("unexpected summary: %s", report)
	}
}

func TestRunWithExecutor_InvalidSpec(t *testing.T) {
	t.Parallel()

	specDir := t.TempDir()
	specPath := filepath.Join(specDir, "invalid.md")
	spec := `# invalid

## stdout contains

- x
`
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	var output bytes.Buffer
	err := runWithExecutor(context.Background(), specDir, &output, func(_ context.Context, _ []string, _ string) (commandResult, error) {
		return commandResult{}, nil
	})
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if !strings.Contains(err.Error(), "missing required 'command' section") {
		t.Fatalf("unexpected error: %v", err)
	}
}
