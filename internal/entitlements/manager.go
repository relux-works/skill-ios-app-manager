package entitlements

import (
	"fmt"
	"strconv"
	"strings"
)

// ListedEntitlement is one entitlement entry emitted by List.
type ListedEntitlement struct {
	Alias string
	Key   string
	Value Value
}

// Add adds or updates an entitlement in a plist file.
func Add(entitlementsPath string, entitlementKey string, rawValue string) error {
	definition, err := ResolveKey(entitlementKey)
	if err != nil {
		return err
	}

	value, err := parseValueForDefinition(definition, rawValue)
	if err != nil {
		return fmt.Errorf("invalid value for %q: %w", definition.Alias, err)
	}

	doc, err := LoadPlistFile(entitlementsPath)
	if err != nil {
		return err
	}

	doc.Set(definition.PlistKey, value)

	if err := WritePlistFile(entitlementsPath, doc); err != nil {
		return err
	}

	return nil
}

// Remove removes an entitlement from a plist file.
func Remove(entitlementsPath string, entitlementKey string) error {
	definition, err := ResolveKey(entitlementKey)
	if err != nil {
		return err
	}

	doc, err := LoadPlistFile(entitlementsPath)
	if err != nil {
		return err
	}

	if !doc.Remove(definition.PlistKey) {
		return nil
	}

	if err := WritePlistFile(entitlementsPath, doc); err != nil {
		return err
	}

	return nil
}

// List returns all entries from an entitlement plist.
func List(entitlementsPath string) ([]ListedEntitlement, error) {
	doc, err := LoadPlistFile(entitlementsPath)
	if err != nil {
		return nil, err
	}

	keys := doc.Keys()
	entries := make([]ListedEntitlement, 0, len(keys))
	for _, key := range keys {
		value, ok := doc.Get(key)
		if !ok {
			continue
		}
		entries = append(entries, ListedEntitlement{
			Alias: AliasForPlistKey(key),
			Key:   key,
			Value: value,
		})
	}

	return entries, nil
}

func parseValueForDefinition(definition KeyDefinition, rawValue string) (Value, error) {
	switch definition.Kind {
	case ValueKindString:
		return parseStringValue(definition, rawValue)
	case ValueKindBool:
		return parseBoolValue(rawValue)
	case ValueKindStringArray:
		return parseArrayValue(rawValue)
	default:
		return Value{}, fmt.Errorf("unsupported entitlement value kind %d", definition.Kind)
	}
}

func parseStringValue(definition KeyDefinition, rawValue string) (Value, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return Value{}, fmt.Errorf("a non-empty string value is required")
	}

	if definition.PlistKey == "aps-environment" {
		env := strings.ToLower(trimmed)
		if env != "development" && env != "production" {
			return Value{}, fmt.Errorf("aps-environment must be development or production")
		}
		trimmed = env
	}

	return Value{
		Kind:        ValueKindString,
		StringValue: trimmed,
	}, nil
}

func parseBoolValue(rawValue string) (Value, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return Value{Kind: ValueKindBool, BoolValue: true}, nil
	}

	normalized := strings.ToLower(trimmed)
	switch normalized {
	case "yes", "on", "enabled":
		return Value{Kind: ValueKindBool, BoolValue: true}, nil
	case "no", "off", "disabled":
		return Value{Kind: ValueKindBool, BoolValue: false}, nil
	}

	parsed, err := strconv.ParseBool(normalized)
	if err != nil {
		return Value{}, fmt.Errorf("expected boolean value")
	}

	return Value{Kind: ValueKindBool, BoolValue: parsed}, nil
}

func parseArrayValue(rawValue string) (Value, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return Value{}, fmt.Errorf("a comma-separated value list is required")
	}

	parts := strings.Split(trimmed, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return Value{}, fmt.Errorf("a comma-separated value list is required")
	}

	return Value{
		Kind:       ValueKindStringArray,
		ArrayValue: items,
	}, nil
}
