package relux

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ActionParameter describes one associated value in a Swift action case.
type ActionParameter struct {
	Name string
	Type string
}

// AddActionInput captures parameters for relux-add-action command.
type AddActionInput struct {
	ModuleName   string
	ModulePath   string
	ActionName   string
	ActionParams []ActionParameter
}

// AddActionCommand modifies actions.swift and reducer.swift for a module.
type AddActionCommand struct{}

// NewAddActionCommand creates relux-add-action command.
func NewAddActionCommand() *AddActionCommand {
	return &AddActionCommand{}
}

// Run executes relux-add-action and returns modified file paths.
func (c *AddActionCommand) Run(ctx context.Context, input AddActionInput) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	moduleName, err := normalizeModuleName(input.ModuleName)
	if err != nil {
		return nil, err
	}

	layout, err := resolveModuleLayout(moduleName, input.ModulePath)
	if err != nil {
		return nil, err
	}

	actionName, err := normalizeActionName(input.ActionName)
	if err != nil {
		return nil, err
	}

	actionParams, err := normalizeActionParameters(input.ActionParams)
	if err != nil {
		return nil, err
	}

	actionsPath, err := findFileByName(
		layout.ModulePath,
		"actions.swift",
		[]string{
			filepath.Join(layout.ImplSourcesDir, "actions.swift"),
			filepath.Join(layout.ModulePath, "Sources", "actions.swift"),
			filepath.Join(layout.ModulePath, "actions.swift"),
		},
	)
	if err != nil {
		return nil, err
	}

	reducerPath, err := findFileByName(
		layout.ModulePath,
		"reducer.swift",
		[]string{
			filepath.Join(layout.ImplSourcesDir, "reducer.swift"),
			filepath.Join(layout.ModulePath, "Sources", "reducer.swift"),
			filepath.Join(layout.ModulePath, "reducer.swift"),
		},
	)
	if err != nil {
		return nil, err
	}

	actionsContent, err := osReadFile(actionsPath)
	if err != nil {
		return nil, err
	}

	nextActionsContent, err := insertActionCase(actionsContent, moduleName, actionName, actionParams)
	if err != nil {
		return nil, fmt.Errorf("update actions.swift: %w", err)
	}

	reducerContent, err := osReadFile(reducerPath)
	if err != nil {
		return nil, err
	}

	nextReducerContent, err := insertReducerCase(reducerContent, actionName, actionParams)
	if err != nil {
		return nil, fmt.Errorf("update reducer.swift: %w", err)
	}

	if err := writeFile(actionsPath, []byte(nextActionsContent)); err != nil {
		return nil, err
	}
	if err := writeFile(reducerPath, []byte(nextReducerContent)); err != nil {
		return nil, err
	}

	return []string{actionsPath, reducerPath}, nil
}

func normalizeActionParameters(raw []ActionParameter) ([]ActionParameter, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	params := make([]ActionParameter, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, candidate := range raw {
		name := strings.TrimSpace(candidate.Name)
		typ := strings.TrimSpace(candidate.Type)

		if name == "" {
			return nil, errors.New("action parameter name is required")
		}
		if typ == "" {
			return nil, fmt.Errorf("action parameter %q type is required", name)
		}
		if !swiftIdentifierPattern.MatchString(name) {
			return nil, fmt.Errorf("action parameter name %q is not a valid Swift identifier", name)
		}

		name = lowerFirst(name)
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("duplicate action parameter name %q", name)
		}
		seen[name] = struct{}{}

		params = append(params, ActionParameter{
			Name: name,
			Type: typ,
		})
	}

	return params, nil
}

func insertActionCase(content string, moduleName string, actionName string, params []ActionParameter) (string, error) {
	enumName := moduleName + "Action"
	enumPattern := regexp.MustCompile(`\benum\s+` + regexp.QuoteMeta(enumName) + `\b`)
	enumIndex := enumPattern.FindStringIndex(content)
	if enumIndex == nil {
		return "", fmt.Errorf("enum %s not found", enumName)
	}

	openingBrace := strings.Index(content[enumIndex[0]:], "{")
	if openingBrace < 0 {
		return "", fmt.Errorf("enum %s opening brace not found", enumName)
	}
	openingBrace += enumIndex[0]

	closingBrace, err := findMatchingBrace(content, openingBrace)
	if err != nil {
		return "", fmt.Errorf("enum %s malformed: %w", enumName, err)
	}

	enumBody := content[openingBrace+1 : closingBrace]

	duplicatePattern := regexp.MustCompile(`(?m)^\s*case\s+` + regexp.QuoteMeta(actionName) + `\b`)
	if duplicatePattern.MatchString(enumBody) {
		return "", fmt.Errorf("action case %q already exists", actionName)
	}

	caseLine := "case " + actionName
	if len(params) > 0 {
		paramParts := make([]string, 0, len(params))
		for _, param := range params {
			paramParts = append(paramParts, fmt.Sprintf("%s: %s", param.Name, param.Type))
		}
		caseLine += "(" + strings.Join(paramParts, ", ") + ")"
	}

	insertAt := closingBrace
	forwardCasePattern := regexp.MustCompile(`(?m)^\s*case\s+forward\b`)
	if loc := forwardCasePattern.FindStringIndex(enumBody); loc != nil {
		insertAt = openingBrace + 1 + loc[0]
	}

	return insertIndentedLine(content, insertAt, "    "+caseLine), nil
}

func insertReducerCase(content string, actionName string, params []ActionParameter) (string, error) {
	switchPattern := regexp.MustCompile(`\bswitch\s+action\s*\{`)
	switchIndex := switchPattern.FindStringIndex(content)
	if switchIndex == nil {
		return "", errors.New("switch action block not found")
	}

	openingBrace := strings.Index(content[switchIndex[0]:], "{")
	if openingBrace < 0 {
		return "", errors.New("switch action opening brace not found")
	}
	openingBrace += switchIndex[0]

	closingBrace, err := findMatchingBrace(content, openingBrace)
	if err != nil {
		return "", fmt.Errorf("switch action block malformed: %w", err)
	}

	switchBody := content[openingBrace+1 : closingBrace]
	duplicatePattern := regexp.MustCompile(`(?m)^\s*case\s+\.` + regexp.QuoteMeta(actionName) + `\b`)
	if duplicatePattern.MatchString(switchBody) {
		return "", fmt.Errorf("reducer case for %q already exists", actionName)
	}

	reducerCasePattern := ".%s"
	if len(params) > 0 {
		bindings := make([]string, 0, len(params))
		for _, param := range params {
			bindings = append(bindings, "let "+param.Name)
		}
		reducerCasePattern = fmt.Sprintf(".%s(%s)", actionName, strings.Join(bindings, ", "))
	} else {
		reducerCasePattern = fmt.Sprintf(reducerCasePattern, actionName)
	}

	caseBlock := strings.Join([]string{
		"        case " + reducerCasePattern + ":",
		"            // TODO: handle " + actionName,
		"            break",
	}, "\n")

	insertAt := closingBrace
	forwardCasePattern := regexp.MustCompile(`(?m)^\s*case\s+\.forward\b`)
	if loc := forwardCasePattern.FindStringIndex(switchBody); loc != nil {
		insertAt = openingBrace + 1 + loc[0]
	}

	return insertIndentedLine(content, insertAt, caseBlock), nil
}

func insertIndentedLine(content string, insertAt int, line string) string {
	prefix := content[:insertAt]
	if !strings.HasSuffix(prefix, "\n") {
		line = "\n" + line
	}
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}

	return prefix + line + content[insertAt:]
}

func osReadFile(path string) (string, error) {
	data, err := readFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
