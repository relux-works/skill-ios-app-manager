package ioc

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const registryInfrastructureBuildersAnchor = "// MARK: - Infrastructure Builders (scaffolding anchor: infra-builders)"

var (
	managedPatchIDPattern    = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	swiftIdentifierPattern   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	reluxRegistrationPattern = regexp.MustCompile(`(?m)^([ \t]*)ioc\.register\(Relux\.self,\s*lifecycle:\s*\.container,\s*resolver:\s*(?:Self\.)?([A-Za-z_][A-Za-z0-9_]*)\)[ \t]*$`)
)

// RegistryManagedFoundationPatch describes a generator-owned Foundation
// registration and builder. The surrounding Registry remains user-owned.
type RegistryManagedFoundationPatch struct {
	ID                       string
	Imports                  []string
	Registration             string
	LegacyRegistrationMarker string
	Builder                  string
	LegacyBuilderMarker      string
}

// RegistryManagedReluxPatch describes a generator-owned wrapper around the
// Registry's existing Relux builder. The original builder remains untouched.
type RegistryManagedReluxPatch struct {
	ID               string
	WrapperName      string
	ModuleExpression string
}

// ConvergeManagedFoundationRegistry applies a byte-idempotent, focused patch
// to an existing Registry.swift file.
func ConvergeManagedFoundationRegistry(path string, appTypeName string, patch RegistryManagedFoundationPatch) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Registry.swift: %w", err)
	}

	updated, err := ConvergeManagedFoundationRegistryContent(string(payload), appTypeName, patch)
	if err != nil {
		return err
	}
	if updated == string(payload) {
		return nil
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write Registry.swift: %w", err)
	}
	return nil
}

// ConvergeManagedFoundationRegistryContent updates only the imports and
// explicitly marked registration/builder owned by patch.ID. A legacy owned
// slice may be adopted once through its narrow marker.
func ConvergeManagedFoundationRegistryContent(content string, appTypeName string, patch RegistryManagedFoundationPatch) (string, error) {
	if err := validateManagedPatchID(patch.ID); err != nil {
		return "", err
	}
	if strings.TrimSpace(patch.Registration) == "" || strings.TrimSpace(patch.Builder) == "" {
		return "", fmt.Errorf("managed Registry foundation patch %q requires registration and builder content", patch.ID)
	}

	updated := content
	for _, moduleName := range patch.Imports {
		moduleName = strings.TrimSpace(moduleName)
		if moduleName != "" {
			updated = EnsureImport(updated, moduleName)
		}
	}

	registrationBegin := managedMarker(patch.ID, "registration", "begin")
	registrationEnd := managedMarker(patch.ID, "registration", "end")
	registrationBody := strings.TrimSpace(patch.Registration)
	var err error
	updated, err = convergeManagedLineBlock(
		updated,
		registrationBegin,
		registrationEnd,
		registrationBody,
		patch.LegacyRegistrationMarker,
		registryFoundationAnchor,
	)
	if err != nil {
		return "", fmt.Errorf("converge %s registration: %w", patch.ID, err)
	}

	builderBegin := managedMarker(patch.ID, "builder", "begin")
	builderEnd := managedMarker(patch.ID, "builder", "end")
	if strings.Contains(updated, builderBegin) {
		updated, err = replaceManagedBlock(updated, builderBegin, builderEnd, strings.TrimSpace(patch.Builder))
	} else if strings.TrimSpace(patch.LegacyBuilderMarker) != "" && strings.Contains(updated, patch.LegacyBuilderMarker) {
		updated, err = adoptLegacyBuilder(updated, patch.LegacyBuilderMarker, builderBegin, builderEnd, patch.Builder)
	} else {
		updated, err = insertManagedBuilder(
			updated,
			appTypeName,
			registryFoundationBuildersAnchor,
			builderBegin,
			builderEnd,
			patch.Builder,
		)
	}
	if err != nil {
		return "", fmt.Errorf("converge %s builder: %w", patch.ID, err)
	}

	return updated, nil
}

// ConvergeManagedReluxWrapper applies a focused Relux registration wrapper to
// an existing Registry.swift file.
func ConvergeManagedReluxWrapper(path string, appTypeName string, patch RegistryManagedReluxPatch) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Registry.swift: %w", err)
	}

	updated, err := ConvergeManagedReluxWrapperContent(string(payload), appTypeName, patch)
	if err != nil {
		return err
	}
	if updated == string(payload) {
		return nil
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write Registry.swift: %w", err)
	}
	return nil
}

// ConvergeManagedReluxWrapperContent replaces the supported one-line Relux
// resolver registration with a managed wrapper and leaves the original Relux
// builder byte-for-byte intact.
func ConvergeManagedReluxWrapperContent(content string, appTypeName string, patch RegistryManagedReluxPatch) (string, error) {
	if err := validateManagedPatchID(patch.ID); err != nil {
		return "", err
	}
	if !swiftIdentifierPattern.MatchString(strings.TrimSpace(patch.WrapperName)) {
		return "", fmt.Errorf("managed Relux patch %q requires a valid wrapper name", patch.ID)
	}
	if strings.TrimSpace(patch.ModuleExpression) == "" {
		return "", fmt.Errorf("managed Relux patch %q requires a module expression", patch.ID)
	}

	registrationPrefix := "// ios-app-manager:" + patch.ID + "-relux-registration:begin"
	registrationEnd := managedMarker(patch.ID, "relux-registration", "end")
	originalBuilder := ""
	updated := content

	if strings.Contains(updated, registrationPrefix) {
		beginLine, err := uniqueLineContaining(updated, registrationPrefix)
		if err != nil {
			return "", fmt.Errorf("locate managed Relux registration: %w", err)
		}
		originalBuilder, err = managedOriginalBuilder(beginLine.text)
		if err != nil {
			return "", err
		}
		begin := registrationPrefix + ` original="` + originalBuilder + `"`
		body := "ioc.register(Relux.self, lifecycle: .container, resolver: Self." + patch.WrapperName + ")"
		updated, err = replaceManagedBlock(updated, registrationPrefix, registrationEnd, bodyWithCustomBegin(begin, body))
		if err != nil {
			return "", fmt.Errorf("converge managed Relux registration: %w", err)
		}
	} else {
		matches := reluxRegistrationPattern.FindAllStringSubmatchIndex(updated, -1)
		if len(matches) != 1 {
			return "", fmt.Errorf("expected exactly one supported Relux registration with a named resolver, found %d", len(matches))
		}
		match := matches[0]
		originalBuilder = updated[match[4]:match[5]]
		indent := updated[match[2]:match[3]]
		begin := registrationPrefix + ` original="` + originalBuilder + `"`
		block := managedBlockWithBegin(indent, begin, registrationEnd,
			"ioc.register(Relux.self, lifecycle: .container, resolver: Self."+patch.WrapperName+")")
		updated = updated[:match[0]] + block + updated[match[1]:]
	}

	wrapperBegin := managedMarker(patch.ID, "relux-wrapper", "begin")
	wrapperEnd := managedMarker(patch.ID, "relux-wrapper", "end")
	wrapper := "private static func " + patch.WrapperName + "() async -> Relux {\n" +
		"    let relux = await " + originalBuilder + "()\n" +
		"    return relux.register(" + strings.TrimSpace(patch.ModuleExpression) + ")\n" +
		"}"

	var err error
	if strings.Contains(updated, wrapperBegin) {
		updated, err = replaceManagedBlock(updated, wrapperBegin, wrapperEnd, wrapper)
	} else {
		updated, err = insertManagedBuilder(
			updated,
			appTypeName,
			registryInfrastructureBuildersAnchor,
			wrapperBegin,
			wrapperEnd,
			wrapper,
		)
	}
	if err != nil {
		return "", fmt.Errorf("converge managed Relux wrapper: %w", err)
	}

	return updated, nil
}

func validateManagedPatchID(id string) error {
	if !managedPatchIDPattern.MatchString(strings.TrimSpace(id)) {
		return fmt.Errorf("managed Registry patch ID %q must contain lowercase letters, digits, and single hyphens", id)
	}
	return nil
}

func managedMarker(id string, section string, boundary string) string {
	return "// ios-app-manager:" + id + "-" + section + ":" + boundary
}

func convergeManagedLineBlock(content, begin, end, body, legacyMarker, anchor string) (string, error) {
	if strings.Contains(content, begin) {
		return replaceManagedBlock(content, begin, end, body)
	}
	if strings.TrimSpace(legacyMarker) != "" && strings.Contains(content, legacyMarker) {
		line, err := uniqueLineContaining(content, legacyMarker)
		if err != nil {
			return "", fmt.Errorf("adopt legacy registration: %w", err)
		}
		block := managedBlock(line.indent, begin, end, body)
		return content[:line.start] + block + content[line.end:], nil
	}

	anchorLine, err := uniqueLineContaining(content, anchor)
	if err != nil {
		return "", fmt.Errorf("locate Registry anchor: %w", err)
	}
	block := managedBlock(anchorLine.indent, begin, end, body)
	insertion := "\n" + block
	return content[:anchorLine.end] + insertion + content[anchorLine.end:], nil
}

func replaceManagedBlock(content, beginMarker, endMarker, body string) (string, error) {
	beginLine, err := uniqueLineContaining(content, beginMarker)
	if err != nil {
		return "", fmt.Errorf("locate begin marker %q: %w", beginMarker, err)
	}
	endLine, err := uniqueLineContaining(content, endMarker)
	if err != nil {
		return "", fmt.Errorf("locate end marker %q: %w", endMarker, err)
	}
	if endLine.start < beginLine.start {
		return "", fmt.Errorf("managed block end marker precedes begin marker")
	}

	beginText := strings.TrimSpace(beginLine.text)
	if strings.HasPrefix(body, "\x00custom-begin\x00") {
		parts := strings.SplitN(strings.TrimPrefix(body, "\x00custom-begin\x00"), "\x00", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid managed block begin metadata")
		}
		beginText = parts[0]
		body = parts[1]
	}
	block := managedBlockWithBegin(beginLine.indent, beginText, strings.TrimSpace(endMarker), body)
	return content[:beginLine.start] + block + content[endLine.end:], nil
}

func bodyWithCustomBegin(begin string, body string) string {
	return "\x00custom-begin\x00" + begin + "\x00" + body
}

func managedBlock(indent, begin, end, body string) string {
	return managedBlockWithBegin(indent, begin, end, body)
}

func managedBlockWithBegin(indent, begin, end, body string) string {
	return indent + strings.TrimSpace(begin) + "\n" +
		indentRegistryBlock(strings.TrimSpace(body), indent) + "\n" +
		indent + strings.TrimSpace(end)
}

func adoptLegacyBuilder(content, marker, begin, end, builder string) (string, error) {
	if strings.Count(content, marker) != 1 {
		return "", fmt.Errorf("legacy builder marker %q must occur exactly once", marker)
	}
	markerIndex := strings.Index(content, marker)
	lineStart := strings.LastIndex(content[:markerIndex], "\n") + 1
	openingOffset := strings.Index(content[markerIndex:], "{")
	if openingOffset < 0 {
		return "", fmt.Errorf("opening brace not found after legacy builder marker %q", marker)
	}
	openingIndex := markerIndex + openingOffset
	closingIndex := matchingRegistryBrace(content, openingIndex)
	if closingIndex < 0 {
		return "", fmt.Errorf("closing brace not found for legacy builder marker %q", marker)
	}
	indent := leadingRegistryIndent(content[lineStart:markerIndex])
	block := managedBlock(indent, begin, end, builder)
	return content[:lineStart] + block + content[closingIndex+1:], nil
}

func insertManagedBuilder(content, appTypeName, anchor, begin, end, builder string) (string, error) {
	anchorIndex := strings.Index(content, anchor)
	if anchorIndex < 0 {
		appTypeName = strings.TrimSpace(appTypeName)
		if !swiftIdentifierPattern.MatchString(appTypeName) {
			return "", fmt.Errorf("valid app type name is required when Registry.swift has no %s anchor", anchor)
		}
		block := managedBlock("    ", begin, end, builder)
		return strings.TrimRight(content, "\n") + "\n\n" + anchor + "\n" +
			"extension " + appTypeName + ".Registry {\n" + block + "\n}\n", nil
	}

	openingOffset := strings.Index(content[anchorIndex:], "{")
	if openingOffset < 0 {
		return "", fmt.Errorf("extension opening brace not found after %s anchor", anchor)
	}
	openingIndex := anchorIndex + openingOffset
	closingIndex := matchingRegistryBrace(content, openingIndex)
	if closingIndex < 0 {
		return "", fmt.Errorf("extension closing brace not found after %s anchor", anchor)
	}
	extensionLineStart := strings.LastIndex(content[:openingIndex], "\n") + 1
	extensionIndent := leadingRegistryIndent(content[extensionLineStart:openingIndex])
	block := managedBlock(extensionIndent+"    ", begin, end, builder)
	closingLineStart := strings.LastIndex(content[:closingIndex], "\n") + 1
	if closingLineStart <= openingIndex {
		if strings.TrimSpace(content[openingIndex+1:closingIndex]) != "" {
			return "", fmt.Errorf("single-line extension after %s anchor contains unsupported content", anchor)
		}
		return content[:closingIndex] + "\n" + block + "\n" + extensionIndent + content[closingIndex:], nil
	}
	prefix := content[:closingLineStart]
	separator := ""
	if !strings.HasSuffix(prefix, "\n") {
		separator = "\n"
	}
	if !strings.HasSuffix(prefix, "\n\n") && strings.TrimSpace(content[openingIndex+1:closingLineStart]) != "" {
		separator += "\n"
	}
	return prefix + separator + block + "\n" + content[closingLineStart:], nil
}

type registryLine struct {
	start  int
	end    int
	indent string
	text   string
}

func uniqueLineContaining(content string, marker string) (registryLine, error) {
	if strings.Count(content, marker) != 1 {
		return registryLine{}, fmt.Errorf("marker %q must occur exactly once", marker)
	}
	index := strings.Index(content, marker)
	start := strings.LastIndex(content[:index], "\n") + 1
	endOffset := strings.Index(content[index:], "\n")
	end := len(content)
	if endOffset >= 0 {
		end = index + endOffset
	}
	text := content[start:end]
	return registryLine{start: start, end: end, indent: leadingRegistryIndent(text), text: text}, nil
}

func managedOriginalBuilder(beginLine string) (string, error) {
	const prefix = `original="`
	start := strings.Index(beginLine, prefix)
	if start < 0 {
		return "", fmt.Errorf("managed Relux registration is missing original builder metadata")
	}
	start += len(prefix)
	endOffset := strings.Index(beginLine[start:], `"`)
	if endOffset < 0 {
		return "", fmt.Errorf("managed Relux registration has invalid original builder metadata")
	}
	original := beginLine[start : start+endOffset]
	if !swiftIdentifierPattern.MatchString(original) {
		return "", fmt.Errorf("managed Relux registration has invalid original builder %q", original)
	}
	return original, nil
}
