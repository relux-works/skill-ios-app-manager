package tuistproj

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	manifestItemKindPattern = regexp.MustCompile(`^\s*\.([A-Za-z_][A-Za-z0-9_]*)`)
	manifestNameArgPattern  = regexp.MustCompile(`\bname\s*:\s*"([^"]+)"`)
	manifestPathArgPattern  = regexp.MustCompile(`\bpath\s*:\s*"([^"]+)"`)
	manifestURLArgPattern   = regexp.MustCompile(`\burl\s*:\s*"([^"]+)"`)
)

// Manifest describes extracted package-level manifest sections.
type Manifest struct {
	Targets      []ManifestItem
	Dependencies []ManifestItem
	Products     []ManifestItem
}

// ManifestItem is one parsed entry from a manifest array.
type ManifestItem struct {
	Name      string
	Kind      string
	StartLine int
	EndLine   int
	Content   string
}

type manifestSectionKind string

const (
	manifestSectionTargets      manifestSectionKind = "targets"
	manifestSectionDependencies manifestSectionKind = "dependencies"
	manifestSectionProducts     manifestSectionKind = "products"
)

type manifestDocument struct {
	content            string
	lines              []string
	hasTrailingNewline bool
	lineOffsets        []int
	sections           map[manifestSectionKind]manifestSection
}

type manifestSection struct {
	Kind      manifestSectionKind
	KeyLine   int
	OpenLine  int
	CloseLine int
	Indent    int
	Items     []ManifestItem
}

// ReadManifestFile loads and parses a Swift manifest file from disk.
func ReadManifestFile(path string) (Manifest, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest file %q: %w", path, err)
	}
	return ParseManifest(string(contents))
}

// ParseManifest extracts target/dependency/product arrays from Swift manifest source.
func ParseManifest(content string) (Manifest, error) {
	doc, err := parseManifestDocument(content)
	if err != nil {
		return Manifest{}, err
	}

	manifest := Manifest{
		Targets:      make([]ManifestItem, 0),
		Dependencies: make([]ManifestItem, 0),
		Products:     make([]ManifestItem, 0),
	}

	if section, ok := doc.section(manifestSectionTargets); ok {
		manifest.Targets = cloneManifestItems(section.Items)
	}
	if section, ok := doc.section(manifestSectionDependencies); ok {
		manifest.Dependencies = cloneManifestItems(section.Items)
	}
	if section, ok := doc.section(manifestSectionProducts); ok {
		manifest.Products = cloneManifestItems(section.Items)
	}

	return manifest, nil
}

func parseManifestDocument(content string) (manifestDocument, error) {
	lines, hasTrailingNewline := splitEditableLines(content)
	parseLines := lines
	if hasTrailingNewline {
		parseLines = append(parseLines, "")
	}

	lineOffsets := buildLineOffsets(content)
	doc := manifestDocument{
		content:            content,
		lines:              parseLines,
		hasTrailingNewline: hasTrailingNewline,
		lineOffsets:        lineOffsets,
		sections:           make(map[manifestSectionKind]manifestSection),
	}

	for _, kind := range []manifestSectionKind{
		manifestSectionTargets,
		manifestSectionDependencies,
		manifestSectionProducts,
	} {
		candidates, err := parseSectionCandidates(content, parseLines, lineOffsets, kind)
		if err != nil {
			return manifestDocument{}, err
		}

		section, ok := pickSection(candidates)
		if ok {
			doc.sections[kind] = section
		}
	}

	return doc, nil
}

func (d manifestDocument) section(kind manifestSectionKind) (manifestSection, bool) {
	section, ok := d.sections[kind]
	return section, ok
}

func parseSectionCandidates(
	content string,
	lines []string,
	lineOffsets []int,
	kind manifestSectionKind,
) ([]manifestSection, error) {
	sectionPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(string(kind)) + `\b\s*:`)

	candidates := make([]manifestSection, 0)
	for lineIndex, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}

		match := sectionPattern.FindStringIndex(line)
		if match == nil {
			continue
		}

		openLine, openCol, ok := findArrayOpenBracket(lines, lineIndex, match[1])
		if !ok {
			continue
		}

		closeLine, closeCol, ok := findMatchingBracket(lines, openLine, openCol)
		if !ok {
			return nil, fmt.Errorf("array %q has no matching closing bracket", kind)
		}

		startOffset := lineColToOffset(lineOffsets, openLine, openCol+1)
		endOffset := lineColToOffset(lineOffsets, closeLine, closeCol)
		items := parseArrayItems(content, lineOffsets, startOffset, endOffset, kind)

		candidates = append(candidates, manifestSection{
			Kind:      kind,
			KeyLine:   lineIndex + 1,
			OpenLine:  openLine + 1,
			CloseLine: closeLine + 1,
			Indent:    leadingIndentWidth(line),
			Items:     items,
		})
	}

	return candidates, nil
}

func pickSection(candidates []manifestSection) (manifestSection, bool) {
	if len(candidates) == 0 {
		return manifestSection{}, false
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Indent == candidates[j].Indent {
			return candidates[i].OpenLine < candidates[j].OpenLine
		}
		return candidates[i].Indent < candidates[j].Indent
	})

	return candidates[0], true
}

func findArrayOpenBracket(lines []string, keyLine int, searchStart int) (int, int, bool) {
	for lineIndex := keyLine; lineIndex < len(lines); lineIndex++ {
		line := lines[lineIndex]
		start := 0
		if lineIndex == keyLine {
			start = searchStart
		}

		col := indexOutsideStringAndComment(line, start, '[')
		if col >= 0 {
			return lineIndex, col, true
		}

		if lineIndex > keyLine {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "//") {
				continue
			}
			if strings.Contains(trimmed, ":") {
				break
			}
		}
	}

	return 0, 0, false
}

func findMatchingBracket(lines []string, openLine int, openCol int) (int, int, bool) {
	depth := 0
	inString := false
	escaped := false

	for lineIndex := openLine; lineIndex < len(lines); lineIndex++ {
		line := lines[lineIndex]
		start := 0
		if lineIndex == openLine {
			start = openCol
		}

		for col := start; col < len(line); col++ {
			ch := line[col]

			if inString {
				if escaped {
					escaped = false
					continue
				}
				if ch == '\\' {
					escaped = true
					continue
				}
				if ch == '"' {
					inString = false
				}
				continue
			}

			if ch == '/' && col+1 < len(line) && line[col+1] == '/' {
				break
			}
			if ch == '"' {
				inString = true
				continue
			}

			switch ch {
			case '[':
				depth++
			case ']':
				if depth > 0 {
					depth--
				}
				if depth == 0 {
					return lineIndex, col, true
				}
			}
		}
	}

	return 0, 0, false
}

func parseArrayItems(
	content string,
	lineOffsets []int,
	startOffset int,
	endOffset int,
	kind manifestSectionKind,
) []ManifestItem {
	if startOffset >= endOffset {
		return nil
	}

	items := make([]ManifestItem, 0)
	depthParen := 0
	depthBracket := 0
	depthBrace := 0
	inString := false
	escaped := false
	inLineComment := false
	itemStart := -1

	appendItem := func(rawStart int, rawEnd int) {
		rawStart, rawEnd = trimByteRange(content, rawStart, rawEnd)
		if rawStart >= rawEnd {
			return
		}

		raw := content[rawStart:rawEnd]
		startLine := offsetToLine(lineOffsets, rawStart) + 1
		endLine := offsetToLine(lineOffsets, rawEnd-1) + 1

		items = append(items, ManifestItem{
			Name:      extractManifestItemName(kind, raw),
			Kind:      extractManifestItemKind(raw),
			StartLine: startLine,
			EndLine:   endLine,
			Content:   raw,
		})
	}

	for offset := startOffset; offset < endOffset; offset++ {
		ch := content[offset]

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '/' && offset+1 < endOffset && content[offset+1] == '/' {
			inLineComment = true
			offset++
			continue
		}
		if ch == '"' {
			inString = true
			if itemStart == -1 {
				itemStart = offset
			}
			continue
		}

		if itemStart == -1 {
			if isWhitespaceByte(ch) || ch == ',' {
				continue
			}
			itemStart = offset
		}

		switch ch {
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '[':
			depthBracket++
		case ']':
			if depthBracket > 0 {
				depthBracket--
			}
		case '{':
			depthBrace++
		case '}':
			if depthBrace > 0 {
				depthBrace--
			}
		case ',':
			if depthParen == 0 && depthBracket == 0 && depthBrace == 0 {
				appendItem(itemStart, offset)
				itemStart = -1
			}
		}
	}

	if itemStart != -1 {
		appendItem(itemStart, endOffset)
	}

	return items
}

func extractManifestItemKind(raw string) string {
	match := manifestItemKindPattern.FindStringSubmatch(raw)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func extractManifestItemName(kind manifestSectionKind, raw string) string {
	if match := manifestNameArgPattern.FindStringSubmatch(raw); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	if kind != manifestSectionDependencies {
		return ""
	}

	if match := manifestPathArgPattern.FindStringSubmatch(raw); len(match) > 1 {
		pathValue := strings.TrimSpace(match[1])
		base := filepath.Base(pathValue)
		base = strings.TrimSpace(base)
		if base != "" && base != "." && base != "/" {
			return base
		}
		if pathValue != "" {
			return pathValue
		}
	}

	if match := manifestURLArgPattern.FindStringSubmatch(raw); len(match) > 1 {
		urlValue := strings.TrimSpace(match[1])
		cleaned := strings.TrimSuffix(urlValue, ".git")
		if slash := strings.LastIndex(cleaned, "/"); slash >= 0 && slash+1 < len(cleaned) {
			return cleaned[slash+1:]
		}
		return cleaned
	}

	return ""
}

func splitEditableLines(content string) ([]string, bool) {
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(content, "\n")
	if hasTrailingNewline && len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, hasTrailingNewline
	}
	return lines, hasTrailingNewline
}

func joinEditableLines(lines []string, hasTrailingNewline bool) string {
	joined := strings.Join(lines, "\n")
	if hasTrailingNewline {
		return joined + "\n"
	}
	return joined
}

func cloneManifestItems(items []ManifestItem) []ManifestItem {
	cloned := make([]ManifestItem, len(items))
	copy(cloned, items)
	return cloned
}

func buildLineOffsets(content string) []int {
	offsets := make([]int, 1, strings.Count(content, "\n")+1)
	offsets[0] = 0
	for index := 0; index < len(content); index++ {
		if content[index] == '\n' {
			offsets = append(offsets, index+1)
		}
	}
	return offsets
}

func lineColToOffset(lineOffsets []int, line int, col int) int {
	if line < 0 {
		return 0
	}
	if line >= len(lineOffsets) {
		return lineOffsets[len(lineOffsets)-1]
	}
	return lineOffsets[line] + col
}

func offsetToLine(lineOffsets []int, offset int) int {
	if offset <= 0 {
		return 0
	}
	if len(lineOffsets) == 0 {
		return 0
	}

	index := sort.Search(len(lineOffsets), func(i int) bool {
		return lineOffsets[i] > offset
	})
	if index == 0 {
		return 0
	}
	return index - 1
}

func indexOutsideStringAndComment(line string, start int, target byte) int {
	inString := false
	escaped := false

	for index := start; index < len(line); index++ {
		ch := line[index]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '/' && index+1 < len(line) && line[index+1] == '/' {
			return -1
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == target {
			return index
		}
	}

	return -1
}

func leadingIndentWidth(line string) int {
	width := 0
	for index := 0; index < len(line); index++ {
		switch line[index] {
		case ' ', '\t':
			width++
		default:
			return width
		}
	}
	return width
}

func trimByteRange(content string, start int, end int) (int, int) {
	for start < end && isWhitespaceByte(content[start]) {
		start++
	}
	for end > start && isWhitespaceByte(content[end-1]) {
		end--
	}
	return start, end
}

func isWhitespaceByte(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
