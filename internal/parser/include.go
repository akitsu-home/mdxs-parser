package parser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var linkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
var codeImportPattern = regexp.MustCompile("(?ms)```([^\\n]*)\\n\\s*#\\s*import\\(([^)\\n]+)\\)\\s*\\n```")

func expandMarkdownLinks(content string, currentPath string, stack map[string]bool) (string, error) {
	return expandMarkdownLinksWithMode(content, currentPath, stack, false)
}

func expandCodeImports(content string, currentPath string) (string, error) {
	matches := codeImportPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	var builder strings.Builder
	lastIndex := 0

	for _, match := range matches {
		start := match[0]
		end := match[1]
		infoStart := match[2]
		infoEnd := match[3]
		targetStart := match[4]
		targetEnd := match[5]

		builder.WriteString(content[lastIndex:start])

		info := content[infoStart:infoEnd]
		target := strings.TrimSpace(content[targetStart:targetEnd])
		resolvedPath := filepath.Clean(filepath.Join(filepath.Dir(currentPath), target))
		imported, err := os.ReadFile(resolvedPath)
		if err != nil {
			return "", fmt.Errorf("read import file %q: %w", resolvedPath, err)
		}

		builder.WriteString("```")
		builder.WriteString(info)
		builder.WriteByte('\n')
		builder.WriteString(strings.TrimSuffix(strings.ReplaceAll(string(imported), "\r\n", "\n"), "\n"))
		builder.WriteByte('\n')
		builder.WriteString("```")

		lastIndex = end
	}

	builder.WriteString(content[lastIndex:])
	return builder.String(), nil
}

func expandMarkdownLinksWithMode(content string, currentPath string, stack map[string]bool, preserveContext bool) (string, error) {
	matches := linkPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	var builder strings.Builder
	lastIndex := 0

	for _, match := range matches {
		start := match[0]
		end := match[1]
		targetStart := match[2]
		targetEnd := match[3]

		// Leave image syntax untouched so only markdown document links are expanded.
		if isImageLink(content, start) {
			continue
		}

		target := strings.TrimSpace(content[targetStart:targetEnd])
		if !shouldInclude(target) {
			continue
		}

		builder.WriteString(content[lastIndex:start])

		targetPath, fragment := splitTarget(target)
		resolvedPath := filepath.Clean(filepath.Join(filepath.Dir(currentPath), targetPath))
		included, err := expandFileWithMode(resolvedPath, stack, preserveContext)
		if err != nil {
			return "", err
		}
		if fragment != "" {
			included, err = extractSection(included, fragment)
			if err != nil {
				return "", err
			}
		}

		includedMeta, includedBody, fmErr := parseFrontMatter(included)
		if fmErr != nil {
			return "", fmErr
		}
		included = strings.TrimPrefix(includedBody, "\n")
		included = strings.TrimSuffix(included, "\n")
		if preserveContext {
			builder.WriteString(includeStart)
			builder.WriteByte('\n')
			if len(includedMeta) > 0 {
				if metaJSON, jsonErr := json.Marshal(includedMeta); jsonErr == nil {
					builder.WriteString(includeMetadataPrefix)
					builder.Write(metaJSON)
					builder.WriteString(includeMetadataSuffix)
					builder.WriteByte('\n')
				}
			}
			builder.WriteString(included)
			builder.WriteByte('\n')
			builder.WriteString(includeEnd)
		} else {
			builder.WriteString(included)
		}
		lastIndex = end
	}

	if lastIndex == 0 {
		return content, nil
	}

	builder.WriteString(content[lastIndex:])
	return builder.String(), nil
}

func shouldInclude(target string) bool {
	if target == "" || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return false
	}

	parsed, err := url.Parse(target)
	if err == nil && parsed.Scheme != "" {
		return false
	}

	path, _ := splitTarget(target)
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".md" || extension == ".markdown"
}

func splitTarget(target string) (string, string) {
	path, fragment, found := strings.Cut(target, "#")
	if !found {
		return target, ""
	}
	return path, fragment
}

func extractSection(content string, fragment string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	fragment = normalizeSlug(fragment)
	start := -1
	level := 0

	for index, line := range lines {
		matches := headingPattern.FindStringSubmatch(strings.TrimSpace(line))
		if matches == nil {
			continue
		}

		title := sanitizeText(matches[2])
		if normalizeSlug(title) != fragment {
			continue
		}

		start = index
		level = len(matches[1])
		break
	}

	if start == -1 {
		return "", fmt.Errorf("section %q not found", fragment)
	}

	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		matches := headingPattern.FindStringSubmatch(strings.TrimSpace(lines[index]))
		if matches == nil {
			continue
		}
		if len(matches[1]) <= level {
			end = index
			break
		}
	}

	return strings.TrimSpace(strings.Join(lines[start:end], "\n")), nil
}

func normalizeSlug(input string) string {
	input = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(input, "#")))
	var builder strings.Builder
	lastHyphen := false
	for _, character := range input {
		switch {
		case character >= 'a' && character <= 'z':
			builder.WriteRune(character)
			lastHyphen = false
		case character >= '0' && character <= '9':
			builder.WriteRune(character)
			lastHyphen = false
		default:
			if !lastHyphen {
				builder.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func isImageLink(content string, start int) bool {
	return start > 0 && content[start-1] == '!'
}

func expandFileForParse(path string, stack map[string]bool) (string, error) {
	return expandFileWithMode(path, stack, true)
}

func expandFileWithMode(path string, stack map[string]bool, preserveContext bool) (string, error) {
	if stack[path] {
		return "", fmt.Errorf("circular include detected for %q", path)
	}

	stack[path] = true
	defer delete(stack, path)

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read markdown file %q: %w", path, err)
	}

	expandedImports, err := expandCodeImports(string(content), path)
	if err != nil {
		return "", err
	}

	return expandMarkdownLinksWithMode(expandedImports, path, stack, preserveContext)
}
