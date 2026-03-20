package cli

import (
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
