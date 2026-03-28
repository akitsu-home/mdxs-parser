package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderJSON_StructuredMarkdownWithIncludes(t *testing.T) {
	tempDir := t.TempDir()
	childPath := filepath.Join(tempDir, "child.md")
	mainPath := filepath.Join(tempDir, "main.md")

	if err := os.WriteFile(childPath, []byte("# Child\n\n## Child Section\n\nChild paragraph.\n\n### Nested Items\n\n- nested item\n"), 0o644); err != nil {
		t.Fatalf("write child markdown: %v", err)
	}

	mainMarkdown := "# Root\n\nIntro with __bold__ text.\n\n[Include child](child.md#child-section)\n\nVisit [site](https://example.com).\n\n```go\nfmt.Println(\"hello\")\n```\n\n## Root Items\n\n- first item\n- second item\n\n## Values\n\n| Name | Value |\n| ---- | ----- |\n| one  | 1     |\n"
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

	metadata, ok := parsed["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing or wrong type: %#v", parsed["metadata"])
	}
	if len(metadata) != 0 {
		t.Fatalf("expected empty metadata: %#v", metadata)
	}

	body, ok := parsed["body"].(map[string]any)
	if !ok {
		t.Fatalf("body missing or wrong type: %#v", parsed["body"])
	}

	root, ok := body["Root"].(map[string]any)
	if !ok {
		t.Fatalf("Root section missing or wrong type: %#v", body["Root"])
	}

	if got := root["description"]; got != "Intro with bold text.\n\nVisit [site](https://example.com)." {
		t.Fatalf("unexpected description: %#v", got)
	}

	list, ok := root["Root Items"].([]any)
	if !ok || len(list) != 2 || list[0] != "first item" || list[1] != "second item" {
		t.Fatalf("unexpected list: %#v", root["Root Items"])
	}

	if got := root["go"]; got != "fmt.Println(\"hello\")" {
		t.Fatalf("unexpected code block: %#v", got)
	}

	table, ok := root["Values"].([]any)
	if !ok || len(table) != 1 {
		t.Fatalf("unexpected table: %#v", root["Values"])
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

	childList, ok := childSection["Nested Items"].([]any)
	if !ok || len(childList) != 1 || childList[0] != "nested item" {
		t.Fatalf("unexpected child list: %#v", childSection["Nested Items"])
	}
}

func TestRenderMarkdown_ExpandsOnlyRelativeMarkdownLinks(t *testing.T) {
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

func TestParseMarkdown_ListMustAppearImmediatelyAfterHeading(t *testing.T) {
	t.Parallel()

	_, err := ParseMarkdown("# Root\n\nParagraph first.\n- item\n")
	if err == nil {
		t.Fatal("expected syntax error")
	}
	if !strings.Contains(err.Error(), "list must appear immediately after its heading") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMarkdown_TableMustAppearImmediatelyAfterHeading(t *testing.T) {
	t.Parallel()

	_, err := ParseMarkdown("# Root\n\nParagraph first.\n| Name | Value |\n| ---- | ----- |\n| one  | 1     |\n")
	if err == nil {
		t.Fatal("expected syntax error")
	}
	if !strings.Contains(err.Error(), "table must appear immediately after its heading") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMarkdown_DuplicateHeadingNameReturnsSyntaxError(t *testing.T) {
	t.Parallel()

	_, err := ParseMarkdown("# Root\n\n## Items\n\n- first\n\n## Items\n\n- second\n")
	if err == nil {
		t.Fatal("expected syntax error")
	}
	if !strings.Contains(err.Error(), `duplicate key "Items" from heading`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMarkdown_WithFrontMatter(t *testing.T) {
	t.Parallel()

	markdown := "---\ntitle: sample\ntags:\n  - go\n  - parser\n---\n# Root\n\nHello.\n"
	parsed, err := ParseMarkdown(markdown)
	if err != nil {
		t.Fatalf("ParseMarkdown returned error: %v", err)
	}

	metadata, ok := parsed["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing or wrong type: %#v", parsed["metadata"])
	}
	if metadata["title"] != "sample" {
		t.Fatalf("unexpected metadata title: %#v", metadata["title"])
	}
	tags, ok := metadata["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "go" || tags[1] != "parser" {
		t.Fatalf("unexpected metadata tags: %#v", metadata["tags"])
	}

	body, ok := parsed["body"].(map[string]any)
	if !ok {
		t.Fatalf("body missing or wrong type: %#v", parsed["body"])
	}
	root, ok := body["Root"].(map[string]any)
	if !ok {
		t.Fatalf("Root section missing or wrong type: %#v", body["Root"])
	}
	if root["description"] != "Hello." {
		t.Fatalf("unexpected root description: %#v", root["description"])
	}
}

func TestParseMarkdown_UnclosedFrontMatterReturnsError(t *testing.T) {
	t.Parallel()

	_, err := ParseMarkdown("---\ntitle: sample\n# Root\n")
	if err == nil {
		t.Fatal("expected syntax error")
	}
	if !strings.Contains(err.Error(), "front matter is not closed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
