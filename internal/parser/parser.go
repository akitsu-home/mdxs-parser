package parser

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	descriptionKey = "description"
	defaultCodeKey = "code"
	includeStart   = "<!-- mdxs-parser:include-start -->"
	includeEnd     = "<!-- mdxs-parser:include-end -->"
	includeMetadataPrefix = "<!-- mdxs-parser:include-metadata "
	includeMetadataSuffix = " -->"
)

var (
	headingPattern          = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	unorderedListPattern    = regexp.MustCompile(`^\s*[-*+]\s+(.*)$`)
	orderedListPattern      = regexp.MustCompile(`^\s*\d+\.\s+(.*)$`)
	boldPattern             = regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)
	italicAsteriskPattern   = regexp.MustCompile(`\*([^*\n]+)\*`)
	italicUnderscorePattern = regexp.MustCompile(`(^|[^_])_([^_\n]+)_([^_]|$)`)
	inlineCodePattern       = regexp.MustCompile("`([^`\n]+)`")
)

type section struct {
	level       int
	title       string
	parent      map[string]any
	node        map[string]any
	childTitles map[string]struct{}
	contentKind contentKind
}

type contentKind string

const (
	contentKindNone  contentKind = ""
	contentKindMap   contentKind = "map"
	contentKindList  contentKind = "list"
	contentKindTable contentKind = "table"
)

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

	return expandFileWithMode(absolutePath, map[string]bool{}, false)
}

func ParseMarkdown(markdown string) (map[string]any, error) {
	normalizedMarkdown := strings.ReplaceAll(markdown, "\r\n", "\n")
	metadata, bodyMarkdown, err := parseFrontMatter(normalizedMarkdown)
	if err != nil {
		return nil, err
	}

	root := map[string]any{}
	stack := []section{{level: 0, node: root, childTitles: map[string]struct{}{}}}
	stackSnapshots := [][]section{}
	includeBaseLevels := []int{}
	lines := strings.Split(bodyMarkdown, "\n")

	var (
		paragraphLines []string
		listItems      []string
		codeLines      []string
		inCodeBlock    bool
		codeBlockKey   string
	)

	currentSection := func() *section {
		return &stack[len(stack)-1]
	}

	currentNode := func() map[string]any {
		current := currentSection()
		if current.node != nil {
			return current.node
		}

		child := ensureSection(current.parent, current.title)
		current.node = child
		if current.contentKind == contentKindNone {
			current.contentKind = contentKindMap
		}
		return child
	}

	addSectionValue := func(value any) {
		current := currentSection()
		if current.parent == nil {
			return
		}

		if existing, exists := current.parent[current.title]; exists {
			switch current.contentKind {
			case contentKindList:
				existingValues, ok := existing.([]any)
				if !ok {
					break
				}
				incomingValues, ok := value.([]any)
				if !ok {
					break
				}
				current.parent[current.title] = append(existingValues, incomingValues...)
				return
			case contentKindTable:
				existingRows, ok := existing.([]map[string]string)
				if !ok {
					break
				}
				incomingRows, ok := value.([]map[string]string)
				if !ok {
					break
				}
				current.parent[current.title] = append(existingRows, incomingRows...)
				return
			}
		}

		current.parent[current.title] = value
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

		values := make([]any, 0, len(listItems))
		for _, item := range listItems {
			values = append(values, item)
		}
		addSectionValue(values)
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
			if currentSection().contentKind == contentKindList || currentSection().contentKind == contentKindTable {
				return nil, syntaxError(index+1, "content is not allowed after a list or table section")
			}
			flushParagraph()
			flushList()
			inCodeBlock = true
			codeBlockKey = sanitizeText(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
			if codeBlockKey == "" {
				codeBlockKey = defaultCodeKey
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
			stackSnapshots = append(stackSnapshots, copyStack(stack))
			includeBaseLevels = append(includeBaseLevels, currentSection().level)
			continue
		}

		if strings.HasPrefix(trimmed, includeMetadataPrefix) && strings.HasSuffix(trimmed, includeMetadataSuffix) {
			jsonStr := trimmed[len(includeMetadataPrefix) : len(trimmed)-len(includeMetadataSuffix)]
			var meta map[string]any
			if err := json.Unmarshal([]byte(jsonStr), &meta); err == nil {
				node := currentNode()
				for k, v := range meta {
					addValue(node, k, v)
				}
			}
			continue
		}

		if trimmed == includeEnd {
			flushParagraph()
			flushList()
			if len(stackSnapshots) == 0 {
				continue
			}
			stack = copyStack(stackSnapshots[len(stackSnapshots)-1])
			stackSnapshots = stackSnapshots[:len(stackSnapshots)-1]
			if len(includeBaseLevels) > 0 {
				includeBaseLevels = includeBaseLevels[:len(includeBaseLevels)-1]
			}
			continue
		}

		if matches := headingPattern.FindStringSubmatch(trimmed); matches != nil {
			flushParagraph()
			flushList()

			level := len(matches[1])
			if len(includeBaseLevels) > 0 {
				minimum := includeBaseLevels[len(includeBaseLevels)-1] + 1
				if level < minimum {
					level = minimum
				}
			}
			title := sanitizeText(matches[2])
			current := currentSection()
			if (current.contentKind == contentKindList || current.contentKind == contentKindTable) && level > current.level {
				return nil, syntaxError(index+1, "subheadings are not allowed after a list or table section")
			}
			for len(stack) > 1 && stack[len(stack)-1].level >= level {
				stack = stack[:len(stack)-1]
			}

			parentSection := currentSection()
			parent := currentNode()
			if _, exists := parentSection.childTitles[title]; exists {
				return nil, syntaxError(index+1, fmt.Sprintf("duplicate key %q from heading", title))
			}
			parentSection.childTitles[title] = struct{}{}

			stack = append(stack, section{
				level:       level,
				title:       title,
				parent:      parent,
				childTitles: map[string]struct{}{},
			})
			continue
		}

		if rows, consumed := parseTable(lines[index:]); consumed > 0 {
			current := currentSection()
			if len(stack) == 1 {
				return nil, syntaxError(index+1, "table must appear immediately after a heading")
			}
			if len(paragraphLines) > 0 {
				return nil, syntaxError(index+1, "table must appear immediately after its heading")
			}
			if current.contentKind == contentKindMap {
				return nil, syntaxError(index+1, "table must appear immediately after its heading")
			}
			if current.contentKind == contentKindList {
				return nil, syntaxError(index+1, "table cannot be mixed with a list under the same heading")
			}

			flushParagraph()
			flushList()
			current.contentKind = contentKindTable
			addSectionValue(rows)
			// The loop increments index after continue, so advance by consumed-1 here.
			index += consumed - 1
			continue
		}

		if matches := unorderedListPattern.FindStringSubmatch(line); matches != nil {
			current := currentSection()
			if len(stack) == 1 {
				return nil, syntaxError(index+1, "list must appear immediately after a heading")
			}
			if len(paragraphLines) > 0 {
				return nil, syntaxError(index+1, "list must appear immediately after its heading")
			}
			if current.contentKind == contentKindMap {
				return nil, syntaxError(index+1, "list must appear immediately after its heading")
			}
			if current.contentKind == contentKindTable {
				return nil, syntaxError(index+1, "list cannot be mixed with a table under the same heading")
			}

			flushParagraph()
			current.contentKind = contentKindList
			listItems = append(listItems, sanitizeText(matches[1]))
			continue
		}

		if matches := orderedListPattern.FindStringSubmatch(line); matches != nil {
			current := currentSection()
			if len(stack) == 1 {
				return nil, syntaxError(index+1, "list must appear immediately after a heading")
			}
			if len(paragraphLines) > 0 {
				return nil, syntaxError(index+1, "list must appear immediately after its heading")
			}
			if current.contentKind == contentKindMap {
				return nil, syntaxError(index+1, "list must appear immediately after its heading")
			}
			if current.contentKind == contentKindTable {
				return nil, syntaxError(index+1, "list cannot be mixed with a table under the same heading")
			}

			flushParagraph()
			current.contentKind = contentKindList
			listItems = append(listItems, sanitizeText(matches[1]))
			continue
		}

		if currentSection().contentKind == contentKindList || currentSection().contentKind == contentKindTable {
			return nil, syntaxError(index+1, "content is not allowed after a list or table section")
		}

		flushList()
		paragraphLines = append(paragraphLines, sanitizeText(trimmed))
	}

	flushParagraph()
	flushList()
	flushCodeBlock()

	return map[string]any{
		"metadata": metadata,
		"body":     root,
	}, nil
}

func parseFrontMatter(markdown string) (map[string]any, string, error) {
	metadata := map[string]any{}
	lines := strings.Split(markdown, "\n")
	if len(lines) == 0 || !isFrontMatterOpeningDelimiter(lines[0]) {
		return metadata, markdown, nil
	}

	endIndex := -1
	for index := 1; index < len(lines); index++ {
		if isFrontMatterClosingDelimiter(lines[index]) {
			endIndex = index
			break
		}
	}
	if endIndex == -1 {
		return nil, "", fmt.Errorf("syntax error: front matter is not closed")
	}

	frontMatterContent := strings.Join(lines[1:endIndex], "\n")
	if strings.TrimSpace(frontMatterContent) != "" {
		if err := yaml.Unmarshal([]byte(frontMatterContent), &metadata); err != nil {
			return nil, "", fmt.Errorf("parse front matter: %w", err)
		}
	}

	return metadata, strings.Join(lines[endIndex+1:], "\n"), nil
}

func isFrontMatterOpeningDelimiter(line string) bool {
	return strings.TrimRight(line, " \t") == "---"
}

func isFrontMatterClosingDelimiter(line string) bool {
	trimmed := strings.TrimRight(line, " \t")
	return trimmed == "---" || trimmed == "..."
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
	output = italicAsteriskPattern.ReplaceAllStringFunc(output, func(match string) string {
		submatches := italicAsteriskPattern.FindStringSubmatch(match)
		for _, submatch := range submatches[1:] {
			if submatch != "" {
				return submatch
			}
		}
		return match
	})
	output = italicUnderscorePattern.ReplaceAllString(output, "$1$2$3")
	output = inlineCodePattern.ReplaceAllString(output, "$1")
	return strings.TrimSpace(output)
}

func copyStack(stack []section) []section {
	return append([]section(nil), stack...)
}

func syntaxError(line int, message string) error {
	return fmt.Errorf("syntax error on line %d: %s", line, message)
}
