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

func TestRunWithExecutor_ExactOutputAssertion(t *testing.T) {
	t.Parallel()

	specDir := t.TempDir()
	specPath := filepath.Join(specDir, "exact.md")
	spec := `# exact

## command

- cmd

## stdout equals

expected output

## stderr equals

error output
`
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	var output bytes.Buffer
	err := runWithExecutor(context.Background(), specDir, &output, func(_ context.Context, _ []string, _ string) (commandResult, error) {
		return commandResult{stdout: "expected output\n", stderr: "error output\n", exitCode: 0}, nil
	})
	if err != nil {
		t.Fatalf("expected exact assertions to pass: %v", err)
	}
	if !strings.Contains(output.String(), "PASS") {
		t.Fatalf("expected PASS in output: %s", output.String())
	}
}

func TestRunWithExecutor_ExactOutputFromFile(t *testing.T) {
	t.Parallel()

	specDir := t.TempDir()
	expectedDir := filepath.Join(specDir, "expected")
	if err := os.MkdirAll(expectedDir, 0o755); err != nil {
		t.Fatalf("mkdir expected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(expectedDir, "out.txt"), []byte("from-file\n"), 0o644); err != nil {
		t.Fatalf("write expected file: %v", err)
	}

	specPath := filepath.Join(specDir, "exact-file.md")
	spec := `# exact from file

## command

- cmd

## stdout equals file

expected/out.txt
`
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	var output bytes.Buffer
	err := runWithExecutor(context.Background(), specDir, &output, func(_ context.Context, _ []string, _ string) (commandResult, error) {
		return commandResult{stdout: "from-file\n", stderr: "", exitCode: 0}, nil
	})
	if err != nil {
		t.Fatalf("expected file-based exact assertion to pass: %v", err)
	}
	if !strings.Contains(output.String(), "PASS") {
		t.Fatalf("expected PASS in output: %s", output.String())
	}
}

func TestRunWithExecutor_CommandCodeBlock(t *testing.T) {
	t.Parallel()

	specDir := t.TempDir()
	specPath := filepath.Join(specDir, "command-block.md")
	spec := "# command block\n\n## command\n\n```command\ncmd\n--flag\n```\n\n## stdout equals\n\n```expected\nok\n```\n"
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	var output bytes.Buffer
	err := runWithExecutor(context.Background(), specDir, &output, func(_ context.Context, command []string, _ string) (commandResult, error) {
		if strings.Join(command, " ") != "cmd --flag" {
			return commandResult{}, fmt.Errorf("unexpected command: %v", command)
		}
		return commandResult{stdout: "ok\n", stderr: "", exitCode: 0}, nil
	})
	if err != nil {
		t.Fatalf("expected command block case to pass: %v", err)
	}
}

func TestValidateResult_JSONSemanticEquals(t *testing.T) {
	t.Parallel()

	testCase := Case{
		StdoutEquals: "{\n  \"a\": 1,\n  \"b\": [\n    1,\n    2\n  ]\n}",
	}
	result := commandResult{
		stdout: "{\"b\":[1,2],\"a\":1}\n",
	}
	assertions := validateResult(testCase, result)
	if len(assertions) != 0 {
		t.Fatalf("expected semantic JSON match without assertions: %v", assertions)
	}
}
