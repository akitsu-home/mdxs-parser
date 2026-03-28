package cli

import (
	"strings"
	"testing"
)

func TestE2ECommandUsesDefaultSpecDir(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"e2e", "/path/that/does/not/exist"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing e2e spec directory")
	}
	if !strings.Contains(err.Error(), "run e2e specs") {
		t.Fatalf("unexpected error: %v", err)
	}
}
