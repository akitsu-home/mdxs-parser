package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	descriptionKey = "description"
	listKey        = "list"
	tableKey       = "table"
	codeKey        = "code"
	includeStart   = "<!-- mdxs-parser:include-start -->"
	includeEnd     = "<!-- mdxs-parser:include-end -->"
)

var (
	headingPattern       = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	unorderedListPattern = regexp.MustCompile(`^\s*[-*+]\s+(.*)$`)
	orderedListPattern   = regexp.MustCompile(`^\s*\d+\.\s+(.*)$`)
	boldPattern          = regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)
	italicPattern        = regexp.MustCompile(`\*([^*\n]+)\*|_([^_\n]+)_`)
	inlineCodePattern    = regexp.MustCompile("`([^`\n]+)`")
)

type section struct {
	level int
	node  map[string]any
}

func RenderJSON(path string) ([]byte, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path %q: %w", path, err)
	}

	expanded, err := expandFileForParse(absolutePath, map[string]bool{})
	if err != nil {
		return nil, err
	}

	parsed, err := ParseMarkdown(expanded)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(parsed, "", "  ")
}

func RenderMarkdown(path string) (string, error) {
	return ExpandIncludes(path)
}

func ExpandIncludes(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}

	return expandFile(absolutePath, map[string]bool{})
}

func ParseMarkdown(markdown string) (map[string]any, error) {
	root := map[string]any{}
	stack := []section{{level: 0, node: root}}
	stackSnapshots := [][]section{}
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")

	var (
		paragraphLines []string
		listItems      []string
		codeLines      []string
		inCodeBlock    bool
		codeBlockKey   string
	)

	currentNode := func() map[string]any {
		return stack[len(stack)-1].node
	}

	flushParagraph := func() {
		if len(paragraphLines) == 0 {
			return
		}
		addValue(currentNode(), descriptionKey, strings.Join(paragraphLines, " "))
		paragraphLines = nil
	}

	flushList := func() {
		if len(listItems) == 0 {
			return
		}
		existing, ok := currentNode()[listKey]
		if !ok {
			values := make([]any, 0, len(listItems))
			for _, item := range listItems {
				values = append(values, item)
			}
			currentNode()[listKey] = values
			listItems = nil
			return
		}

		values, ok := existing.([]any)
		if !ok {
			values = []any{existing}
		}
		for _, item := range listItems {
			values = append(values, item)
		}
		currentNode()[listKey] = values
		listItems = nil
	}

	flushCodeBlock := func() {
		if !inCodeBlock {
			return
		}
		addValue(currentNode(), codeBlockKey, strings.Join(codeLines, "\n"))
		codeLines = nil
		inCodeBlock = false
		codeBlockKey = ""
	}

	for index := 0; index < len(lines); index++ {
		line := lines[index]
		trimmed := strings.TrimSpace(line)

		if inCodeBlock {
			if strings.HasPrefix(trimmed, "```") {
				flushCodeBlock()
				continue
			}
			codeLines = append(codeLines, line)
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			flushParagraph()
			flushList()
			inCodeBlock = true
			codeBlockKey = sanitizeText(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
			if codeBlockKey == "" {
				codeBlockKey = codeKey
			}
			codeLines = nil
			continue
		}

		if trimmed == "" {
			flushParagraph()
			flushList()
			continue
		}

		if trimmed == includeStart {
			flushParagraph()
			flushList()
			stackSnapshots = append(stackSnapshots, append([]section(nil), stack...))
			continue
		}

		if trimmed == includeEnd {
			flushParagraph()
			flushList()
			if len(stackSnapshots) == 0 {
				continue
			}
			stack = append([]section(nil), stackSnapshots[len(stackSnapshots)-1]...)
			stackSnapshots = stackSnapshots[:len(stackSnapshots)-1]
			continue
		}

		if matches := headingPattern.FindStringSubmatch(trimmed); matches != nil {
			flushParagraph()
			flushList()

			level := len(matches[1])
			title := sanitizeText(matches[2])
			for len(stack) > 1 && stack[len(stack)-1].level >= level {
				stack = stack[:len(stack)-1]
			}

			parent := currentNode()
			child := ensureSection(parent, title)
			stack = append(stack, section{level: level, node: child})
			continue
		}

		if rows, consumed := parseTable(lines[index:]); consumed > 0 {
			flushParagraph()
			flushList()
			addValue(currentNode(), tableKey, rows)
			index += consumed - 1
			continue
		}

		if matches := unorderedListPattern.FindStringSubmatch(line); matches != nil {
			flushParagraph()
			listItems = append(listItems, sanitizeText(matches[1]))
			continue
		}

		if matches := orderedListPattern.FindStringSubmatch(line); matches != nil {
			flushParagraph()
			listItems = append(listItems, sanitizeText(matches[1]))
			continue
		}

		flushList()
		paragraphLines = append(paragraphLines, sanitizeText(trimmed))
	}

	flushParagraph()
	flushList()
	flushCodeBlock()

	return root, nil
}

func ensureSection(parent map[string]any, title string) map[string]any {
	if existing, ok := parent[title]; ok {
		if node, ok := existing.(map[string]any); ok {
			return node
		}
	}

	child := map[string]any{}
	parent[title] = child
	return child
}

func addValue(node map[string]any, key string, value any) {
	switch key {
	case descriptionKey:
		if existing, ok := node[key].(string); ok && existing != "" {
			node[key] = existing + "\n\n" + value.(string)
			return
		}
	case listKey:
		if existing, ok := node[key].([]any); ok {
			if incoming, ok := value.([]any); ok {
				node[key] = append(existing, incoming...)
				return
			}
		}
	case tableKey:
		if existing, ok := node[key].([]map[string]string); ok {
			if incoming, ok := value.([]map[string]string); ok {
				node[key] = append(existing, incoming...)
				return
			}
		}
	}

	if existing, exists := node[key]; exists {
		switch current := existing.(type) {
		case []any:
			node[key] = append(current, value)
		default:
			node[key] = []any{existing, value}
		}
		return
	}

	node[key] = value
}

func parseTable(lines []string) ([]map[string]string, int) {
	if len(lines) < 2 {
		return nil, 0
	}

	headerLine := strings.TrimSpace(lines[0])
	separatorLine := strings.TrimSpace(lines[1])
	if !isTableRow(headerLine) || !isTableSeparator(separatorLine) {
		return nil, 0
	}

	headers := splitTableRow(headerLine)
	if len(headers) == 0 {
		return nil, 0
	}

	rows := []map[string]string{}
	consumed := 2
	for consumed < len(lines) {
		line := strings.TrimSpace(lines[consumed])
		if line == "" || !isTableRow(line) {
			break
		}

		cells := splitTableRow(line)
		row := map[string]string{}
		for index, header := range headers {
			value := ""
			if index < len(cells) {
				value = sanitizeText(cells[index])
			}
			row[sanitizeText(header)] = value
		}
		rows = append(rows, row)
		consumed++
	}

	if len(rows) == 0 {
		return nil, 0
	}

	return rows, consumed
}

func isTableRow(line string) bool {
	return strings.Count(line, "|") >= 2
}

func isTableSeparator(line string) bool {
	cells := splitTableRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			return false
		}
		for _, character := range cell {
			if character != '-' && character != ':' {
				return false
			}
		}
	}
	return true
}

func splitTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	if trimmed == "" {
		return nil
	}

	parts := strings.Split(trimmed, "|")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, strings.TrimSpace(part))
	}
	return values
}

func sanitizeText(input string) string {
	output := boldPattern.ReplaceAllStringFunc(input, func(match string) string {
		submatches := boldPattern.FindStringSubmatch(match)
		for _, submatch := range submatches[1:] {
			if submatch != "" {
				return submatch
			}
		}
		return match
	})
	output = italicPattern.ReplaceAllStringFunc(output, func(match string) string {
		submatches := italicPattern.FindStringSubmatch(match)
		for _, submatch := range submatches[1:] {
			if submatch != "" {
				return submatch
			}
		}
		return match
	})
	output = inlineCodePattern.ReplaceAllString(output, "$1")
	return strings.TrimSpace(output)
}

func expandFile(path string, stack map[string]bool) (string, error) {
	if stack[path] {
		return "", fmt.Errorf("circular include detected for %q", path)
	}

	stack[path] = true
	defer delete(stack, path)

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read markdown file %q: %w", path, err)
	}

	return expandMarkdownLinks(string(content), path, stack)
}
