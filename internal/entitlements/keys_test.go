package entitlements

import (
	"strings"
	"testing"
)

func TestResolveKeySupportsAliasesAndPlistKeys(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input        string
		wantAlias    string
		wantPlistKey string
		wantKind     ValueKind
	}{
		{
			input:        "push",
			wantAlias:    "push",
			wantPlistKey: "aps-environment",
			wantKind:     ValueKindString,
		},
		{
			input:        "aps-environment",
			wantAlias:    "push",
			wantPlistKey: "aps-environment",
			wantKind:     ValueKindString,
		},
		{
			input:        "app-groups",
			wantAlias:    "app-groups",
			wantPlistKey: "com.apple.security.application-groups",
			wantKind:     ValueKindStringArray,
		},
		{
			input:        "com.apple.security.application-groups",
			wantAlias:    "app-groups",
			wantPlistKey: "com.apple.security.application-groups",
			wantKind:     ValueKindStringArray,
		},
		{
			input:        "keychain",
			wantAlias:    "keychain",
			wantPlistKey: "keychain-access-groups",
			wantKind:     ValueKindStringArray,
		},
		{
			input:        "icloud",
			wantAlias:    "icloud",
			wantPlistKey: "com.apple.developer.icloud-container-identifiers",
			wantKind:     ValueKindStringArray,
		},
		{
			input:        "healthkit",
			wantAlias:    "healthkit",
			wantPlistKey: "com.apple.developer.healthkit",
			wantKind:     ValueKindBool,
		},
		{
			input:        "associated-domains",
			wantAlias:    "associated-domains",
			wantPlistKey: "com.apple.developer.associated-domains",
			wantKind:     ValueKindStringArray,
		},
		{
			input:        "background-modes",
			wantAlias:    "background-modes",
			wantPlistKey: "UIBackgroundModes",
			wantKind:     ValueKindStringArray,
		},
		{
			input:        "UIBackgroundModes",
			wantAlias:    "background-modes",
			wantPlistKey: "UIBackgroundModes",
			wantKind:     ValueKindStringArray,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			definition, err := ResolveKey(tc.input)
			if err != nil {
				t.Fatalf("ResolveKey(%q) error = %v", tc.input, err)
			}

			if definition.Alias != tc.wantAlias {
				t.Fatalf("ResolveKey(%q) alias = %q, want %q", tc.input, definition.Alias, tc.wantAlias)
			}
			if definition.PlistKey != tc.wantPlistKey {
				t.Fatalf("ResolveKey(%q) plist key = %q, want %q", tc.input, definition.PlistKey, tc.wantPlistKey)
			}
			if definition.Kind != tc.wantKind {
				t.Fatalf("ResolveKey(%q) kind = %d, want %d", tc.input, definition.Kind, tc.wantKind)
			}
		})
	}
}

func TestResolveKeyUnknownReturnsHelpfulError(t *testing.T) {
	t.Parallel()

	_, err := ResolveKey("unsupported-key")
	if err == nil {
		t.Fatal("ResolveKey(unsupported-key) error = nil, want error")
	}

	message := err.Error()
	for _, expected := range []string{"unsupported entitlement key", "push", "app-groups", "healthkit"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("ResolveKey error missing %q: %s", expected, message)
		}
	}
}

func TestAliasForPlistKey(t *testing.T) {
	t.Parallel()

	if got := AliasForPlistKey("aps-environment"); got != "push" {
		t.Fatalf("AliasForPlistKey(aps-environment) = %q, want push", got)
	}
	if got := AliasForPlistKey("UIBackgroundModes"); got != "background-modes" {
		t.Fatalf("AliasForPlistKey(UIBackgroundModes) = %q, want background-modes", got)
	}
	if got := AliasForPlistKey("unknown.key"); got != "" {
		t.Fatalf("AliasForPlistKey(unknown.key) = %q, want empty", got)
	}
}
