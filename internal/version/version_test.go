package version

import "testing"

func TestDefaultsAreSet(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
	if Commit == "" {
		t.Fatal("Commit must not be empty")
	}
	if Date == "" {
		t.Fatal("Date must not be empty")
	}
}
