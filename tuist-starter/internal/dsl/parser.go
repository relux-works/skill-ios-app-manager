package dsl

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Expression is a parsed DSL expression: operation(params) { fields }.
type Expression struct {
	Operation string            `json:"operation"`
	Params    map[string]string `json:"params"`
	Fields    []string          `json:"fields"`
}

// Parse parses a DSL expression.
func Parse(input string) (Expression, error) {
	return ParseExpression(input)
}

// ParseExpression parses operation(params) { fields } into Expression.
func ParseExpression(input string) (Expression, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return Expression{}, fmt.Errorf("expression is empty")
	}

	openParen := strings.IndexRune(raw, '(')
	if openParen < 0 {
		return Expression{}, fmt.Errorf("expression %q is missing '('", raw)
	}

	operation := strings.TrimSpace(raw[:openParen])
	if !isIdentifier(operation) {
		return Expression{}, fmt.Errorf("invalid operation name %q", operation)
	}

	closeParen, err := findMatchingDelimiter(raw, openParen, '(', ')')
	if err != nil {
		return Expression{}, err
	}

	paramsRaw := strings.TrimSpace(raw[openParen+1 : closeParen])
	params, err := parseParams(paramsRaw)
	if err != nil {
		return Expression{}, err
	}

	fields := []string{}
	remainder := strings.TrimSpace(raw[closeParen+1:])
	if remainder != "" {
		if !strings.HasPrefix(remainder, "{") {
			return Expression{}, fmt.Errorf("unexpected trailing content %q", remainder)
		}

		closeBrace, err := findMatchingDelimiter(remainder, 0, '{', '}')
		if err != nil {
			return Expression{}, err
		}

		fieldsRaw := strings.TrimSpace(remainder[1:closeBrace])
		fields, err = parseFields(fieldsRaw)
		if err != nil {
			return Expression{}, err
		}

		trailing := strings.TrimSpace(remainder[closeBrace+1:])
		if trailing != "" {
			return Expression{}, fmt.Errorf("unexpected trailing content %q", trailing)
		}
	}

	return Expression{
		Operation: operation,
		Params:    params,
		Fields:    fields,
	}, nil
}

func parseParams(raw string) (map[string]string, error) {
	params := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return params, nil
	}

	parts, err := splitTopLevel(raw, ',')
	if err != nil {
		return nil, err
	}

	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			return nil, fmt.Errorf("parameter list contains empty parameter")
		}

		eqIdx, err := findTopLevelRune(token, '=')
		if err != nil {
			return nil, err
		}
		if eqIdx < 0 {
			return nil, fmt.Errorf("invalid parameter %q: expected key=value", token)
		}

		key := strings.TrimSpace(token[:eqIdx])
		if !isIdentifier(key) {
			return nil, fmt.Errorf("invalid parameter name %q", key)
		}
		if _, exists := params[key]; exists {
			return nil, fmt.Errorf("duplicate parameter %q", key)
		}

		valueRaw := strings.TrimSpace(token[eqIdx+1:])
		value, err := normalizeValue(valueRaw)
		if err != nil {
			return nil, err
		}

		params[key] = value
	}

	return params, nil
}

func parseFields(raw string) ([]string, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return nil, fmt.Errorf("field projection cannot be empty")
	}

	normalized = strings.ReplaceAll(normalized, ",", " ")
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return nil, fmt.Errorf("field projection cannot be empty")
	}

	fields := make([]string, 0, len(parts))
	for _, field := range parts {
		if field != "*" && !isIdentifier(field) {
			return nil, fmt.Errorf("invalid field name %q", field)
		}
		fields = append(fields, field)
	}

	return fields, nil
}

func normalizeValue(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("parameter value is empty")
	}

	first := trimmed[0]
	last := trimmed[len(trimmed)-1]
	if (first == '"' || first == '\'') && first == last && len(trimmed) >= 2 {
		return decodeQuoted(trimmed[1:len(trimmed)-1], first)
	}

	if first == '"' || first == '\'' || last == '"' || last == '\'' {
		return "", fmt.Errorf("unterminated quoted value %q", trimmed)
	}

	return trimmed, nil
}

func decodeQuoted(value string, quote byte) (string, error) {
	var b strings.Builder
	escaped := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if escaped {
			switch ch {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\', '\'', '"':
				b.WriteByte(ch)
			default:
				b.WriteByte(ch)
			}
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if ch == quote {
			return "", fmt.Errorf("unexpected quote in quoted value")
		}

		b.WriteByte(ch)
	}

	if escaped {
		return "", fmt.Errorf("unterminated escape sequence in quoted value")
	}

	return b.String(), nil
}

func splitTopLevel(input string, delimiter rune) ([]string, error) {
	parts := []string{}
	start := 0
	state := parserState{}

	for idx, r := range input {
		if r == delimiter && state.isTopLevel() {
			parts = append(parts, input[start:idx])
			start = idx + utf8.RuneLen(r)
			continue
		}

		if err := state.consume(r); err != nil {
			return nil, err
		}
	}

	if err := state.finalize(); err != nil {
		return nil, err
	}

	parts = append(parts, input[start:])
	return parts, nil
}

func findTopLevelRune(input string, target rune) (int, error) {
	state := parserState{}
	for idx, r := range input {
		if r == target && state.isTopLevel() {
			return idx, nil
		}
		if err := state.consume(r); err != nil {
			return -1, err
		}
	}

	if err := state.finalize(); err != nil {
		return -1, err
	}

	return -1, nil
}

func findMatchingDelimiter(input string, start int, openRune rune, closeRune rune) (int, error) {
	if start < 0 || start >= len(input) {
		return -1, fmt.Errorf("invalid delimiter start index")
	}
	if rune(input[start]) != openRune {
		return -1, fmt.Errorf("expected %q at index %d", openRune, start)
	}

	depth := 0
	state := quoteState{}

	for offset, r := range input[start:] {
		idx := start + offset

		if state.inQuote != 0 {
			state.consume(r)
			continue
		}

		if r == '\'' || r == '"' {
			state.consume(r)
			continue
		}

		if r == openRune {
			depth++
			continue
		}

		if r == closeRune {
			depth--
			if depth == 0 {
				return idx, nil
			}
			continue
		}
	}

	if state.inQuote != 0 {
		return -1, fmt.Errorf("unterminated quoted string")
	}

	return -1, fmt.Errorf("missing closing %q", closeRune)
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}

	for idx, r := range value {
		if idx == 0 {
			if unicode.IsLetter(r) || r == '_' {
				continue
			}
			return false
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			continue
		}

		return false
	}

	return true
}

type parserState struct {
	inQuote      rune
	escaped      bool
	parenDepth   int
	braceDepth   int
	bracketDepth int
}

func (s *parserState) consume(r rune) error {
	if s.inQuote != 0 {
		if s.escaped {
			s.escaped = false
			return nil
		}
		if r == '\\' {
			s.escaped = true
			return nil
		}
		if r == s.inQuote {
			s.inQuote = 0
		}
		return nil
	}

	switch r {
	case '\'', '"':
		s.inQuote = r
	case '(':
		s.parenDepth++
	case ')':
		if s.parenDepth == 0 {
			return fmt.Errorf("unexpected ')' in expression")
		}
		s.parenDepth--
	case '{':
		s.braceDepth++
	case '}':
		if s.braceDepth == 0 {
			return fmt.Errorf("unexpected '}' in expression")
		}
		s.braceDepth--
	case '[':
		s.bracketDepth++
	case ']':
		if s.bracketDepth == 0 {
			return fmt.Errorf("unexpected ']' in expression")
		}
		s.bracketDepth--
	}

	return nil
}

func (s *parserState) finalize() error {
	if s.inQuote != 0 {
		return fmt.Errorf("unterminated quoted string")
	}
	if s.escaped {
		return fmt.Errorf("unterminated escape sequence")
	}
	if s.parenDepth != 0 {
		return fmt.Errorf("unbalanced parentheses in expression")
	}
	if s.braceDepth != 0 {
		return fmt.Errorf("unbalanced braces in expression")
	}
	if s.bracketDepth != 0 {
		return fmt.Errorf("unbalanced brackets in expression")
	}
	return nil
}

func (s *parserState) isTopLevel() bool {
	return s.inQuote == 0 && s.parenDepth == 0 && s.braceDepth == 0 && s.bracketDepth == 0
}

type quoteState struct {
	inQuote rune
	escaped bool
}

func (s *quoteState) consume(r rune) {
	if s.inQuote == 0 {
		if r == '\'' || r == '"' {
			s.inQuote = r
			s.escaped = false
		}
		return
	}

	if s.escaped {
		s.escaped = false
		return
	}
	if r == '\\' {
		s.escaped = true
		return
	}
	if r == s.inQuote {
		s.inQuote = 0
	}
}
