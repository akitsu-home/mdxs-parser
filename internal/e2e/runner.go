package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/owner/mdxs-parser/internal/parser"
)

type Case struct {
	Name              string
	Command           []string
	StdoutEquals      string
	StderrEquals      string
	StdoutContains    []string
	StderrContains    []string
	StdoutNotContains []string
	StderrNotContains []string
	ExitCode          int
}

type commandResult struct {
	stdout   string
	stderr   string
	exitCode int
}

type executor func(ctx context.Context, command []string, workDir string) (commandResult, error)

var commandBlockKeys = []string{"code", "bash", "sh", "shell"}

func Run(ctx context.Context, specDir string, out io.Writer) error {
	return runWithExecutor(ctx, specDir, out, runCommand)
}

func runWithExecutor(ctx context.Context, specDir string, out io.Writer, execFn executor) error {
	specFiles, err := discoverSpecFiles(specDir)
	if err != nil {
		return err
	}
	if len(specFiles) == 0 {
		return fmt.Errorf("no markdown spec files found under %q", specDir)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	failedCount := 0
	totalCount := 0
	for _, file := range specFiles {
		cases, err := loadCases(file)
		if err != nil {
			return err
		}
		for _, testCase := range cases {
			totalCount++
			fmt.Fprintf(out, "RUN  %s :: %s\n", file, testCase.Name)

			result, err := execFn(ctx, testCase.Command, workDir)
			if err != nil {
				return fmt.Errorf("run %q: %w", testCase.Name, err)
			}

			if assertions := validateResult(testCase, result); len(assertions) > 0 {
				failedCount++
				fmt.Fprintf(out, "FAIL %s :: %s\n", file, testCase.Name)
				for _, assertion := range assertions {
					fmt.Fprintf(out, "  - %s\n", assertion)
				}
				continue
			}

			fmt.Fprintf(out, "PASS %s :: %s\n", file, testCase.Name)
		}
	}

	fmt.Fprintf(out, "\nSummary: total=%d passed=%d failed=%d\n", totalCount, totalCount-failedCount, failedCount)
	if failedCount > 0 {
		return fmt.Errorf("%d e2e case(s) failed", failedCount)
	}
	return nil
}

func discoverSpecFiles(specDir string) ([]string, error) {
	entries := []string{}
	err := filepath.WalkDir(specDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == "expected" {
				return filepath.SkipDir
			}
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension == ".md" || extension == ".markdown" {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk spec dir %q: %w", specDir, err)
	}
	sort.Strings(entries)
	return entries, nil
}

func loadCases(path string) ([]Case, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec file %q: %w", path, err)
	}

	parsed, err := parser.ParseMarkdown(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse spec file %q: %w", path, err)
	}

	body, ok := parsed["body"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("spec file %q has invalid body", path)
	}

	names := make([]string, 0, len(body))
	for name := range body {
		names = append(names, name)
	}
	sort.Strings(names)

	cases := make([]Case, 0, len(names))
	for _, name := range names {
		raw, ok := body[name].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("spec %q in %q must be a heading section", name, path)
		}
		testCase, err := parseCase(path, name, raw)
		if err != nil {
			return nil, fmt.Errorf("spec %q in %q: %w", name, path, err)
		}
		cases = append(cases, testCase)
	}

	return cases, nil
}

func parseCase(specPath string, name string, raw map[string]any) (Case, error) {
	command, err := getCommand(raw)
	if err != nil {
		return Case{}, err
	}

	exitCode := 0
	if value, ok := findField(raw, "exit_code"); ok {
		parsedExitCode, parseErr := parseExitCode(value)
		if parseErr != nil {
			return Case{}, parseErr
		}
		exitCode = parsedExitCode
	}

	stdoutEquals := toExpectedText(findFieldValue(raw, "stdout_equals"))
	if stdoutEquals == "" {
		expectedPath, ok := findField(raw, "stdout_equals_file")
		if ok {
			stdoutEquals, err = loadExpectedFile(specPath, toExpectedText(expectedPath))
			if err != nil {
				return Case{}, err
			}
		}
	}

	stderrEquals := toExpectedText(findFieldValue(raw, "stderr_equals"))
	if stderrEquals == "" {
		expectedPath, ok := findField(raw, "stderr_equals_file")
		if ok {
			stderrEquals, err = loadExpectedFile(specPath, toExpectedText(expectedPath))
			if err != nil {
				return Case{}, err
			}
		}
	}

	return Case{
		Name:              name,
		Command:           command,
		StdoutEquals:      stdoutEquals,
		StderrEquals:      stderrEquals,
		StdoutContains:    toStringSlice(findFieldValue(raw, "stdout_contains")),
		StderrContains:    toStringSlice(findFieldValue(raw, "stderr_contains")),
		StdoutNotContains: toStringSlice(findFieldValue(raw, "stdout_not_contains")),
		StderrNotContains: toStringSlice(findFieldValue(raw, "stderr_not_contains")),
		ExitCode:          exitCode,
	}, nil
}

func getCommand(raw map[string]any) ([]string, error) {
	value, ok := findField(raw, "command")
	if !ok {
		return nil, errors.New("missing required 'command' section")
	}

	switch typed := value.(type) {
	case []any:
		command := toStringSlice(typed)
		if len(command) == 0 {
			return nil, errors.New("'command' list must not be empty")
		}
		return command, nil
	case map[string]any:
		for _, key := range commandBlockKeys {
			if code, exists := findField(typed, key); exists {
				text := strings.TrimSpace(fmt.Sprintf("%v", code))
				if text == "" {
					return nil, errors.New("'command' code block must not be empty")
				}
				return parseCommandBlock(text), nil
			}
		}
	}

	return nil, errors.New("'command' must be a list or code block")
}

func parseExitCode(raw any) (int, error) {
	switch typed := raw.(type) {
	case string:
		value := strings.TrimSpace(typed)
		if value == "" {
			return 0, errors.New("'exit_code' must not be empty")
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("parse exit_code %q: %w", value, err)
		}
		return parsed, nil
	case int:
		return typed, nil
	case float64:
		return int(typed), nil
	case map[string]any:
		description, ok := findField(typed, "description")
		if !ok {
			return 0, errors.New("'exit_code' section must contain a description")
		}
		return parseExitCode(description)
	default:
		return 0, errors.New("'exit_code' must be a string or number")
	}
}

func validateResult(testCase Case, result commandResult) []string {
	assertions := []string{}

	if testCase.ExitCode != result.exitCode {
		assertions = append(assertions, fmt.Sprintf("exit code mismatch: expected %d got %d", testCase.ExitCode, result.exitCode))
	}

	if testCase.StdoutEquals != "" {
		expected := normalizeExpectedOutput(testCase.StdoutEquals)
		actual := normalizeExpectedOutput(result.stdout)
		if actual != expected {
			assertions = append(assertions, fmt.Sprintf("stdout mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual))
		}
	}
	if testCase.StderrEquals != "" {
		expected := normalizeExpectedOutput(testCase.StderrEquals)
		actual := normalizeExpectedOutput(result.stderr)
		if actual != expected {
			assertions = append(assertions, fmt.Sprintf("stderr mismatch:\nexpected:\n%s\nactual:\n%s", expected, actual))
		}
	}

	for _, expected := range testCase.StdoutContains {
		if !strings.Contains(result.stdout, expected) {
			assertions = append(assertions, fmt.Sprintf("stdout does not contain %q", expected))
		}
	}
	for _, expected := range testCase.StderrContains {
		if !strings.Contains(result.stderr, expected) {
			assertions = append(assertions, fmt.Sprintf("stderr does not contain %q", expected))
		}
	}
	for _, unexpected := range testCase.StdoutNotContains {
		if strings.Contains(result.stdout, unexpected) {
			assertions = append(assertions, fmt.Sprintf("stdout unexpectedly contains %q", unexpected))
		}
	}
	for _, unexpected := range testCase.StderrNotContains {
		if strings.Contains(result.stderr, unexpected) {
			assertions = append(assertions, fmt.Sprintf("stderr unexpectedly contains %q", unexpected))
		}
	}

	return assertions
}

func runCommand(ctx context.Context, command []string, workDir string) (commandResult, error) {
	if len(command) == 0 {
		return commandResult{}, errors.New("empty command")
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = workDir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return commandResult{
			stdout:   stdout.String(),
			stderr:   stderr.String(),
			exitCode: 0,
		}, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return commandResult{
			stdout:   stdout.String(),
			stderr:   stderr.String(),
			exitCode: exitErr.ExitCode(),
		}, nil
	}

	return commandResult{}, err
}

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			result = append(result, strings.TrimSpace(fmt.Sprintf("%v", item)))
		}
		return result
	case []string:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			result = append(result, strings.TrimSpace(item))
		}
		return result
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	default:
		trimmed := strings.TrimSpace(fmt.Sprintf("%v", typed))
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	}
}

func parseCommandBlock(text string) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	command := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		command = append(command, trimmed)
	}
	return command
}

func toExpectedText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		for _, key := range []string{"description", "text", "code", "json", "yaml", "markdown", "md"} {
			if nested, ok := findField(typed, key); ok {
				return toExpectedText(nested)
			}
		}
	}
	return ""
}

func normalizeExpectedOutput(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	return strings.TrimRight(normalized, "\n")
}

func loadExpectedFile(specPath string, relativePath string) (string, error) {
	trimmed := strings.TrimSpace(relativePath)
	if trimmed == "" {
		return "", errors.New("'stdout_equals_file' and 'stderr_equals_file' must not be empty")
	}
	resolvedPath := filepath.Clean(filepath.Join(filepath.Dir(specPath), trimmed))
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("read expected output file %q: %w", resolvedPath, err)
	}
	return string(content), nil
}

func findFieldValue(values map[string]any, normalized string) any {
	value, _ := findField(values, normalized)
	return value
}

func findField(values map[string]any, normalized string) (any, bool) {
	for key, value := range values {
		if normalizeKey(key) == normalized {
			return value, true
		}
	}
	return nil, false
}

func normalizeKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	return value
}
