package cli

import "testing"

func TestE2ECommandUsesDefaultSpecDir(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"e2e"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error because default e2e-mdxs does not exist in test")
	}
}

