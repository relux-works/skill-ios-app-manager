package dsl

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/testutil"
)

func TestParseExpressionGolden(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		expression string
		goldenName string
	}{
		{
			name:       "summary",
			expression: "summary()",
			goldenName: "dsl/parser_summary",
		},
		{
			name:       "modules filtered",
			expression: "modules(type=feature)",
			goldenName: "dsl/parser_modules_filtered",
		},
		{
			name:       "get with projection",
			expression: "get(module=Auth) { name type deps }",
			goldenName: "dsl/parser_get_projection",
		},
		{
			name:       "quoted values",
			expression: `create_module(name="Auth(Core)", type='feature', note="needs, commas and (parens)")`,
			goldenName: "dsl/parser_quoted_values",
		},
		{
			name:       "nested parentheses",
			expression: "deps(module=resolve(name=Auth, level=2))",
			goldenName: "dsl/parser_nested_parentheses",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression, err := ParseExpression(tc.expression)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tc.expression, err)
			}

			payload, err := json.MarshalIndent(expression, "", "  ")
			if err != nil {
				t.Fatalf("json.MarshalIndent() error = %v", err)
			}

			testutil.AssertGoldenFile(t, tc.goldenName, string(payload)+"\n")
		})
	}
}

func TestParseExpressionErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expr     string
		contains string
	}{
		{
			name:     "empty expression",
			expr:     "  ",
			contains: "empty",
		},
		{
			name:     "missing parentheses",
			expr:     "summary",
			contains: "missing '('",
		},
		{
			name:     "duplicate parameter",
			expr:     "modules(type=feature, type=core)",
			contains: "duplicate parameter",
		},
		{
			name:     "empty field projection",
			expr:     "get(module=Auth) { }",
			contains: "field projection cannot be empty",
		},
		{
			name:     "unterminated quote",
			expr:     `create_module(name="Auth)`,
			contains: "unterminated",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseExpression(tc.expr)
			if err == nil {
				t.Fatalf("ParseExpression(%q) error = nil, want error", tc.expr)
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("ParseExpression(%q) error = %q, want substring %q", tc.expr, err.Error(), tc.contains)
			}
		})
	}
}
