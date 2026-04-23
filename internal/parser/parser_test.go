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

func TestParseMarkdown_FrontMatterCanCloseWithDots(t *testing.T) {
	t.Parallel()

	parsed, err := ParseMarkdown("---\ntitle: sample\n...\n# Root\n")
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
}

func TestRenderJSON_IncludedFileWithFrontMatter(t *testing.T) {
	tempDir := t.TempDir()
	childPath := filepath.Join(tempDir, "child.md")
	mainPath := filepath.Join(tempDir, "main.md")

	// child.md: Front Matter + level-3 heading (nests under level-2 Section)
	if err := os.WriteFile(childPath, []byte("---\ntitle: Child Title\nauthor: Alice\n---\n\n### Details\n\nSome details.\n"), 0o644); err != nil {
		t.Fatalf("write child markdown: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte("# Root\n\n## Section\n\n[child](child.md)\n"), 0o644); err != nil {
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

	body, ok := parsed["body"].(map[string]any)
	if !ok {
		t.Fatalf("body missing: %#v", parsed["body"])
	}
	root, ok := body["Root"].(map[string]any)
	if !ok {
		t.Fatalf("Root missing: %#v", body)
	}
	section, ok := root["Section"].(map[string]any)
	if !ok {
		t.Fatalf("Section missing: %#v", root)
	}

	if section["title"] != "Child Title" {
		t.Fatalf("expected title from front matter, got: %#v", section["title"])
	}
	if section["author"] != "Alice" {
		t.Fatalf("expected author from front matter, got: %#v", section["author"])
	}

	details, ok := section["Details"].(map[string]any)
	if !ok {
		t.Fatalf("Details sub-section missing: %#v", section)
	}
	if details["description"] != "Some details." {
		t.Fatalf("unexpected Details description: %#v", details["description"])
	}
}

func TestRenderMarkdown_StripsIncludedFrontMatter(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.md")
	childPath := filepath.Join(tempDir, "child.md")

	if err := os.WriteFile(childPath, []byte("---\ntitle: foo\n---\n\n## Included\n\nIncluded text.\n"), 0o644); err != nil {
		t.Fatalf("write child markdown: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte("# Root\n\n[Local](child.md)\n"), 0o644); err != nil {
		t.Fatalf("write main markdown: %v", err)
	}

	output, err := RenderMarkdown(mainPath)
	if err != nil {
		t.Fatalf("RenderMarkdown returned error: %v", err)
	}

	expected := "# Root\n\n## Included\n\nIncluded text.\n"
	if output != expected {
		t.Fatalf("unexpected expanded markdown:\nexpected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestRenderJSON_IncludedTopHeadingDoesNotEscapeParentSection(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.md")
	childPath := filepath.Join(tempDir, "child.md")

	if err := os.WriteFile(childPath, []byte("---\ntitle: Hello World\n---\n\n# hello\n\ntesttesttest\n"), 0o644); err != nil {
		t.Fatalf("write child markdown: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte("# Runtime\n\n## Runtime Details\n\n### Part 1\n\n[part1](child.md)\n"), 0o644); err != nil {
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

	body, ok := parsed["body"].(map[string]any)
	if !ok {
		t.Fatalf("body missing: %#v", parsed["body"])
	}

	runtimeSection, ok := body["Runtime"].(map[string]any)
	if !ok {
		t.Fatalf("Runtime section missing: %#v", body)
	}
	details, ok := runtimeSection["Runtime Details"].(map[string]any)
	if !ok {
		t.Fatalf("Runtime Details section missing: %#v", runtimeSection)
	}
	part1, ok := details["Part 1"].(map[string]any)
	if !ok {
		t.Fatalf("Part 1 section missing: %#v", details)
	}

	if part1["title"] != "Hello World" {
		t.Fatalf("expected title in Part 1: %#v", part1["title"])
	}
	hello, ok := part1["hello"].(map[string]any)
	if !ok {
		t.Fatalf("hello should be nested in Part 1: %#v", part1)
	}
	if hello["description"] != "testtesttest" {
		t.Fatalf("unexpected hello description: %#v", hello["description"])
	}

	if _, exists := body["hello"]; exists {
		t.Fatalf("hello section must not exist at root: %#v", body["hello"])
	}
}

func TestRenderMarkdown_ImportsCodeBlockContent(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.md")
	codePath := filepath.Join(tempDir, "test.python")

	if err := os.WriteFile(codePath, []byte("print('hello')\nprint('world')\n"), 0o644); err != nil {
		t.Fatalf("write code file: %v", err)
	}

	markdown := "# Root\n\n```\n# import(./test.python)\n```\n"
	if err := os.WriteFile(mainPath, []byte(markdown), 0o644); err != nil {
		t.Fatalf("write main markdown: %v", err)
	}

	output, err := RenderMarkdown(mainPath)
	if err != nil {
		t.Fatalf("RenderMarkdown returned error: %v", err)
	}

	expected := "# Root\n\n```\nprint('hello')\nprint('world')\n```\n"
	if output != expected {
		t.Fatalf("unexpected markdown output:\nexpected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestRenderMarkdownWithOptions_LinksCodeBlockImport(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.md")

	markdown := "# Root\n\n```python\n# import(./missing.py)\n```\n"
	if err := os.WriteFile(mainPath, []byte(markdown), 0o644); err != nil {
		t.Fatalf("write main markdown: %v", err)
	}

	output, err := RenderMarkdownWithOptions(mainPath, MarkdownOptions{ImportMode: ImportModeLink})
	if err != nil {
		t.Fatalf("RenderMarkdownWithOptions returned error: %v", err)
	}

	expected := "# Root\n\n[python](./missing.py)\n"
	if output != expected {
		t.Fatalf("unexpected markdown output:\nexpected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestRenderMarkdownWithOptions_LinkModeStillExpandsMarkdownIncludes(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.md")
	childPath := filepath.Join(tempDir, "child.md")

	if err := os.WriteFile(childPath, []byte("## Included\n\nIncluded text.\n"), 0o644); err != nil {
		t.Fatalf("write child markdown: %v", err)
	}

	markdown := "# Root\n\n[child](child.md)\n\n```python\n# import(./missing.py)\n```\n"
	if err := os.WriteFile(mainPath, []byte(markdown), 0o644); err != nil {
		t.Fatalf("write main markdown: %v", err)
	}

	output, err := RenderMarkdownWithOptions(mainPath, MarkdownOptions{ImportMode: ImportModeLink})
	if err != nil {
		t.Fatalf("RenderMarkdownWithOptions returned error: %v", err)
	}

	expected := "# Root\n\n## Included\n\nIncluded text.\n\n[python](./missing.py)\n"
	if output != expected {
		t.Fatalf("unexpected markdown output:\nexpected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestRenderMarkdownWithOptions_LinkModeUsesPathWhenInfoStringIsEmpty(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.md")

	markdown := "# Root\n\n```\n# import(./missing.py)\n```\n"
	if err := os.WriteFile(mainPath, []byte(markdown), 0o644); err != nil {
		t.Fatalf("write main markdown: %v", err)
	}

	output, err := RenderMarkdownWithOptions(mainPath, MarkdownOptions{ImportMode: ImportModeLink})
	if err != nil {
		t.Fatalf("RenderMarkdownWithOptions returned error: %v", err)
	}

	expected := "# Root\n\n[./missing.py](./missing.py)\n"
	if output != expected {
		t.Fatalf("unexpected markdown output:\nexpected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestRenderMarkdown_AdjustsIncludedHeadingLevels(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.md")
	childPath := filepath.Join(tempDir, "child.md")

	if err := os.WriteFile(childPath, []byte("# hello\n\ntesttesttest\n\n## test\n\nん？\n"), 0o644); err != nil {
		t.Fatalf("write child markdown: %v", err)
	}

	markdown := "# Runtime\n\n## Runtime Details\n\n### Part 1\n\n[part1](child.md)\n"
	if err := os.WriteFile(mainPath, []byte(markdown), 0o644); err != nil {
		t.Fatalf("write main markdown: %v", err)
	}

	output, err := RenderMarkdown(mainPath)
	if err != nil {
		t.Fatalf("RenderMarkdown returned error: %v", err)
	}

	expected := "# Runtime\n\n## Runtime Details\n\n### Part 1\n\n#### hello\n\ntesttesttest\n\n##### test\n\nん？\n"
	if output != expected {
		t.Fatalf("unexpected markdown output:\nexpected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestRenderJSON_ImportsCodeBlockContent(t *testing.T) {
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.md")
	codePath := filepath.Join(tempDir, "test.python")

	if err := os.WriteFile(codePath, []byte("print('hello')\nprint('world')\n"), 0o644); err != nil {
		t.Fatalf("write code file: %v", err)
	}

	markdown := "# Root\n\n## Script\n\n```\n# import(./test.python)\n```\n"
	if err := os.WriteFile(mainPath, []byte(markdown), 0o644); err != nil {
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

	if script["code"] != "print('hello')\nprint('world')" {
		t.Fatalf("unexpected imported code: %#v", script["code"])
	}
}
