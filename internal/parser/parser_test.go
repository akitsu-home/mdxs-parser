package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderJSONParsesStructuredMarkdownWithIncludes(t *testing.T) {
	tempDir := t.TempDir()
	childPath := filepath.Join(tempDir, "child.md")
	mainPath := filepath.Join(tempDir, "main.md")

	if err := os.WriteFile(childPath, []byte("# Child\n\n## Child Section\n\nChild paragraph.\n\n- nested item\n"), 0o644); err != nil {
		t.Fatalf("write child markdown: %v", err)
	}

	mainMarkdown := "# Root\n\nIntro with **bold** text.\n\n- first item\n- second item\n\n[Include child](child.md#child-section)\n\nVisit [site](https://example.com).\n\n```go\nfmt.Println(\"hello\")\n```\n\n| Name | Value |\n| ---- | ----- |\n| one  | 1     |\n"
	if err := os.WriteFile(mainPath, []byte(mainMarkdown), 0o644); err != nil {
		t.Fatalf("write main markdown: %v", err)
	}

	output, err := RenderJSON(mainPath)
	if err != nil {
		t.Fatalf("RenderJSON returned error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	root, ok := parsed["Root"].(map[string]any)
	if !ok {
		t.Fatalf("Root section missing or wrong type: %#v", parsed["Root"])
	}

	if got := root["description"]; got != "Intro with bold text.\n\nVisit [site](https://example.com)." {
		t.Fatalf("unexpected description: %#v", got)
	}

	list, ok := root["list"].([]any)
	if !ok || len(list) != 2 || list[0] != "first item" || list[1] != "second item" {
		t.Fatalf("unexpected list: %#v", root["list"])
	}

	if got := root["go"]; got != "fmt.Println(\"hello\")" {
		t.Fatalf("unexpected code block: %#v", got)
	}

	table, ok := root["table"].([]any)
	if !ok || len(table) != 1 {
		t.Fatalf("unexpected table: %#v", root["table"])
	}

	row, ok := table[0].(map[string]any)
	if !ok || row["Name"] != "one" || row["Value"] != "1" {
		t.Fatalf("unexpected table row: %#v", table[0])
	}

	childSection, ok := root["Child Section"].(map[string]any)
	if !ok {
		t.Fatalf("child section missing: %#v", root["Child Section"])
	}

	if childSection["description"] != "Child paragraph." {
		t.Fatalf("unexpected child description: %#v", childSection["description"])
	}

	childList, ok := childSection["list"].([]any)
	if !ok || len(childList) != 1 || childList[0] != "nested item" {
		t.Fatalf("unexpected child list: %#v", childSection["list"])
	}
}

func TestRenderMarkdownExpandsOnlyRelativeMarkdownLinks(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.md")
	otherPath := filepath.Join(tempDir, "other.md")

	if err := os.WriteFile(otherPath, []byte("## Included\n\nIncluded text.\n"), 0o644); err != nil {
		t.Fatalf("write include markdown: %v", err)
	}

	content := "# Root\n\n[Local](other.md)\n\n[External](https://example.com/docs)\n"
	if err := os.WriteFile(mainPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write main markdown: %v", err)
	}

	output, err := RenderMarkdown(mainPath)
	if err != nil {
		t.Fatalf("RenderMarkdown returned error: %v", err)
	}

	expected := "# Root\n\n## Included\n\nIncluded text.\n\n[External](https://example.com/docs)\n"
	if output != expected {
		t.Fatalf("unexpected expanded markdown:\nexpected:\n%s\ngot:\n%s", expected, output)
	}
}
