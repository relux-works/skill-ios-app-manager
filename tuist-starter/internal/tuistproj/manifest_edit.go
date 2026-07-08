package tuistproj

import (
	"fmt"
	"os"
	"strings"
)

// EditType defines manifest edit operation kind.
type EditType string

const (
	AddTarget        EditType = "AddTarget"
	RemoveTarget     EditType = "RemoveTarget"
	AddDependency    EditType = "AddDependency"
	RemoveDependency EditType = "RemoveDependency"
	AddProduct       EditType = "AddProduct"
	RemoveProduct    EditType = "RemoveProduct"
)

// ManifestEdit is one add/remove operation for a manifest section.
type ManifestEdit struct {
	Type    EditType
	Name    string
	Content string
}

// ApplyManifestEdits applies add/remove operations to Swift manifest text.
func ApplyManifestEdits(content string, edits ...ManifestEdit) (string, error) {
	updated := content
	for _, edit := range edits {
		next, err := applyManifestEdit(updated, edit)
		if err != nil {
			return "", err
		}
		updated = next
	}
	return updated, nil
}

// ApplyManifestEditsToFile loads a manifest, applies edits, and writes it back.
func ApplyManifestEditsToFile(path string, edits ...ManifestEdit) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read manifest file %q: %w", path, err)
	}

	updated, err := ApplyManifestEdits(string(payload), edits...)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write manifest file %q: %w", path, err)
	}
	return nil
}

func applyManifestEdit(content string, edit ManifestEdit) (string, error) {
	sectionKind, addOperation, err := resolveEditSection(edit.Type)
	if err != nil {
		return "", err
	}

	doc, err := parseManifestDocument(content)
	if err != nil {
		return "", err
	}

	section, ok := doc.section(sectionKind)
	if !ok {
		return "", fmt.Errorf("manifest section %q not found", sectionKind)
	}

	if addOperation {
		return applyManifestAdd(content, section, edit)
	}
	return applyManifestRemove(content, section, edit)
}

func resolveEditSection(editType EditType) (manifestSectionKind, bool, error) {
	switch editType {
	case AddTarget:
		return manifestSectionTargets, true, nil
	case RemoveTarget:
		return manifestSectionTargets, false, nil
	case AddDependency:
		return manifestSectionDependencies, true, nil
	case RemoveDependency:
		return manifestSectionDependencies, false, nil
	case AddProduct:
		return manifestSectionProducts, true, nil
	case RemoveProduct:
		return manifestSectionProducts, false, nil
	default:
		return "", false, fmt.Errorf("unsupported manifest edit type %q", editType)
	}
}

func applyManifestAdd(content string, section manifestSection, edit ManifestEdit) (string, error) {
	name := strings.TrimSpace(edit.Name)
	if name == "" {
		return "", fmt.Errorf("edit %s requires Name", edit.Type)
	}
	if strings.TrimSpace(edit.Content) == "" {
		return "", fmt.Errorf("edit %s requires Content", edit.Type)
	}

	for _, item := range section.Items {
		if item.Name == name {
			return "", fmt.Errorf("manifest section %q already contains %q", section.Kind, name)
		}
	}

	lines, hasTrailingNewline := splitEditableLines(content)
	insertAt := section.CloseLine - 1
	if insertAt < 0 || insertAt > len(lines) {
		return "", fmt.Errorf("invalid insertion line %d for section %q", section.CloseLine, section.Kind)
	}

	indent := insertionIndent(section, lines)
	insertLines, err := formatManifestInsertLines(edit.Content, indent)
	if err != nil {
		return "", err
	}

	updatedLines := make([]string, 0, len(lines)+len(insertLines))
	if len(section.Items) > 0 {
		previousItemEnd := section.Items[len(section.Items)-1].EndLine - 1
		if previousItemEnd >= 0 && previousItemEnd < len(lines) {
			trimmed := strings.TrimSpace(lines[previousItemEnd])
			if trimmed != "" && !strings.HasSuffix(trimmed, ",") {
				lines[previousItemEnd] += ","
			}
		}
	}
	updatedLines = append(updatedLines, lines[:insertAt]...)
	updatedLines = append(updatedLines, insertLines...)
	updatedLines = append(updatedLines, lines[insertAt:]...)

	return joinEditableLines(updatedLines, hasTrailingNewline), nil
}

func applyManifestRemove(content string, section manifestSection, edit ManifestEdit) (string, error) {
	name := strings.TrimSpace(edit.Name)
	if name == "" {
		return "", fmt.Errorf("edit %s requires Name", edit.Type)
	}

	var target *ManifestItem
	for i := range section.Items {
		if section.Items[i].Name == name {
			target = &section.Items[i]
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("manifest section %q does not contain %q", section.Kind, name)
	}

	lines, hasTrailingNewline := splitEditableLines(content)
	start := target.StartLine - 1
	end := target.EndLine - 1
	if start < 0 || end < start || end >= len(lines) {
		return "", fmt.Errorf("invalid removal range %d:%d for %q", target.StartLine, target.EndLine, name)
	}

	updatedLines := make([]string, 0, len(lines)-(end-start+1))
	updatedLines = append(updatedLines, lines[:start]...)
	updatedLines = append(updatedLines, lines[end+1:]...)

	return joinEditableLines(updatedLines, hasTrailingNewline), nil
}

func insertionIndent(section manifestSection, lines []string) string {
	for _, item := range section.Items {
		index := item.StartLine - 1
		if index < 0 || index >= len(lines) {
			continue
		}
		return leadingIndentPrefix(lines[index])
	}

	closeIndex := section.CloseLine - 1
	if closeIndex < 0 || closeIndex >= len(lines) {
		return "    "
	}

	baseIndent := leadingIndentPrefix(lines[closeIndex])
	if strings.Contains(baseIndent, "\t") {
		return baseIndent + "\t"
	}
	return baseIndent + "    "
}

func formatManifestInsertLines(content string, indent string) ([]string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, fmt.Errorf("insert content is empty")
	}

	lines := strings.Split(trimmed, "\n")
	lines = dedentLines(lines)

	for i := range lines {
		if strings.TrimSpace(lines[i]) == "" {
			lines[i] = ""
			continue
		}
		lines[i] = indent + lines[i]
	}

	lastNonEmpty := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		lastNonEmpty = i
		break
	}
	if lastNonEmpty == -1 {
		return nil, fmt.Errorf("insert content is empty")
	}

	if !strings.HasSuffix(strings.TrimSpace(lines[lastNonEmpty]), ",") {
		lines[lastNonEmpty] += ","
	}

	return lines, nil
}

func dedentLines(lines []string) []string {
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		indent := leadingIndentWidth(line)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return lines
	}

	dedented := make([]string, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			dedented[i] = ""
			continue
		}
		if minIndent >= len(line) {
			dedented[i] = strings.TrimSpace(line)
			continue
		}
		dedented[i] = line[minIndent:]
	}
	return dedented
}

func leadingIndentPrefix(line string) string {
	end := 0
	for end < len(line) {
		if line[end] != ' ' && line[end] != '\t' {
			break
		}
		end++
	}
	return line[:end]
}
