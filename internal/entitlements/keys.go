package entitlements

import (
	"fmt"
	"sort"
	"strings"
)

// KeyDefinition describes one supported entitlement alias and plist key.
type KeyDefinition struct {
	Alias    string
	PlistKey string
	Kind     ValueKind
}

var supportedKeyDefinitions = []KeyDefinition{
	{
		Alias:    "push",
		PlistKey: "aps-environment",
		Kind:     ValueKindString,
	},
	{
		Alias:    "app-groups",
		PlistKey: "com.apple.security.application-groups",
		Kind:     ValueKindStringArray,
	},
	{
		Alias:    "keychain",
		PlistKey: "keychain-access-groups",
		Kind:     ValueKindStringArray,
	},
	{
		Alias:    "icloud",
		PlistKey: "com.apple.developer.icloud-container-identifiers",
		Kind:     ValueKindStringArray,
	},
	{
		Alias:    "healthkit",
		PlistKey: "com.apple.developer.healthkit",
		Kind:     ValueKindBool,
	},
	{
		Alias:    "associated-domains",
		PlistKey: "com.apple.developer.associated-domains",
		Kind:     ValueKindStringArray,
	},
	{
		Alias:    "background-modes",
		PlistKey: "UIBackgroundModes",
		Kind:     ValueKindStringArray,
	},
}

var (
	keyLookupByAlias     = buildKeyLookupByAlias()
	canonicalAliasByKey  = buildCanonicalAliasLookup()
	supportedAliasValues = buildSupportedAliasValues()
)

// SupportedKeys returns all supported entitlement aliases.
func SupportedKeys() []KeyDefinition {
	copied := make([]KeyDefinition, len(supportedKeyDefinitions))
	copy(copied, supportedKeyDefinitions)
	return copied
}

// ResolveKey resolves a user-provided alias or plist key into a supported key definition.
func ResolveKey(input string) (KeyDefinition, error) {
	normalized := normalizeLookupKey(input)
	if normalized == "" {
		return KeyDefinition{}, fmt.Errorf("entitlement key is required")
	}

	if definition, ok := keyLookupByAlias[normalized]; ok {
		return definition, nil
	}

	return KeyDefinition{}, fmt.Errorf(
		"unsupported entitlement key %q (supported aliases: %s)",
		input,
		strings.Join(supportedAliasValues, ", "),
	)
}

// AliasForPlistKey returns the canonical alias for a plist key, if known.
func AliasForPlistKey(plistKey string) string {
	return canonicalAliasByKey[normalizeLookupKey(plistKey)]
}

func buildKeyLookupByAlias() map[string]KeyDefinition {
	lookup := make(map[string]KeyDefinition, len(supportedKeyDefinitions)*3)

	register := func(definition KeyDefinition, aliases ...string) {
		for _, alias := range aliases {
			normalized := normalizeLookupKey(alias)
			if normalized == "" {
				continue
			}
			lookup[normalized] = definition
		}
	}

	for _, definition := range supportedKeyDefinitions {
		synonyms := []string{definition.Alias, definition.PlistKey}
		switch definition.Alias {
		case "push":
			synonyms = append(synonyms, "push-notifications")
		case "app-groups":
			synonyms = append(synonyms, "application-groups")
		case "keychain":
			synonyms = append(synonyms, "keychain-groups")
		case "icloud":
			synonyms = append(synonyms, "icloud-containers")
		case "associated-domains":
			synonyms = append(synonyms, "domains")
		case "background-modes":
			synonyms = append(synonyms, "background")
		}
		register(definition, synonyms...)
	}

	return lookup
}

func buildCanonicalAliasLookup() map[string]string {
	lookup := make(map[string]string, len(supportedKeyDefinitions))
	for _, definition := range supportedKeyDefinitions {
		lookup[normalizeLookupKey(definition.PlistKey)] = definition.Alias
	}
	return lookup
}

func buildSupportedAliasValues() []string {
	aliases := make([]string, 0, len(supportedKeyDefinitions))
	for _, definition := range supportedKeyDefinitions {
		aliases = append(aliases, definition.Alias)
	}
	sort.Strings(aliases)
	return aliases
}

func normalizeLookupKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
