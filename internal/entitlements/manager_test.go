package entitlements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddUpdatesValuesAndPreservesExistingEntries(t *testing.T) {
	t.Parallel()

	path := writeTestEntitlementsFile(t, testPlistXML)

	if err := Add(path, "push", "production"); err != nil {
		t.Fatalf("Add(push) error = %v", err)
	}
	if err := Add(path, "associated-domains", "applinks:example.com, webcredentials:example.com"); err != nil {
		t.Fatalf("Add(associated-domains) error = %v", err)
	}

	doc, err := LoadPlistFile(path)
	if err != nil {
		t.Fatalf("LoadPlistFile() error = %v", err)
	}

	push, ok := doc.Get("aps-environment")
	if !ok {
		t.Fatal("aps-environment missing after add")
	}
	if push.Kind != ValueKindString || push.StringValue != "production" {
		t.Fatalf("aps-environment = %#v, want string production", push)
	}

	healthkit, ok := doc.Get("com.apple.developer.healthkit")
	if !ok || healthkit.Kind != ValueKindBool || !healthkit.BoolValue {
		t.Fatalf("healthkit should remain true, got %#v (exists=%v)", healthkit, ok)
	}

	appGroups, ok := doc.Get("com.apple.security.application-groups")
	if !ok || appGroups.Kind != ValueKindStringArray || len(appGroups.ArrayValue) != 2 {
		t.Fatalf("app groups should be preserved, got %#v (exists=%v)", appGroups, ok)
	}

	domains, ok := doc.Get("com.apple.developer.associated-domains")
	if !ok {
		t.Fatal("associated domains missing after add")
	}
	if domains.Kind != ValueKindStringArray || len(domains.ArrayValue) != 2 {
		t.Fatalf("associated domains = %#v, want 2 values", domains)
	}
}

func TestAddSupportsBooleanDefaultsAndExplicitValues(t *testing.T) {
	t.Parallel()

	path := writeTestEntitlementsFile(t, minimalPlistXML)

	if err := Add(path, "healthkit", ""); err != nil {
		t.Fatalf("Add(healthkit, default) error = %v", err)
	}

	doc, err := LoadPlistFile(path)
	if err != nil {
		t.Fatalf("LoadPlistFile() error = %v", err)
	}

	healthkit, ok := doc.Get("com.apple.developer.healthkit")
	if !ok || healthkit.Kind != ValueKindBool || !healthkit.BoolValue {
		t.Fatalf("healthkit default value = %#v (exists=%v), want true", healthkit, ok)
	}

	if err := Add(path, "healthkit", "false"); err != nil {
		t.Fatalf("Add(healthkit, false) error = %v", err)
	}

	doc, err = LoadPlistFile(path)
	if err != nil {
		t.Fatalf("LoadPlistFile() after false error = %v", err)
	}

	healthkit, ok = doc.Get("com.apple.developer.healthkit")
	if !ok || healthkit.BoolValue {
		t.Fatalf("healthkit after false = %#v (exists=%v), want false", healthkit, ok)
	}
}

func TestAddValidatesValueShape(t *testing.T) {
	t.Parallel()

	path := writeTestEntitlementsFile(t, minimalPlistXML)

	if err := Add(path, "app-groups", ""); err == nil {
		t.Fatal("Add(app-groups, empty) error = nil, want validation error")
	}

	if err := Add(path, "push", "staging"); err == nil {
		t.Fatal("Add(push, staging) error = nil, want validation error")
	}
}

func TestRemoveDeletesEntries(t *testing.T) {
	t.Parallel()

	path := writeTestEntitlementsFile(t, testPlistXML)

	if err := Remove(path, "healthkit"); err != nil {
		t.Fatalf("Remove(healthkit) error = %v", err)
	}

	doc, err := LoadPlistFile(path)
	if err != nil {
		t.Fatalf("LoadPlistFile() error = %v", err)
	}

	if _, ok := doc.Get("com.apple.developer.healthkit"); ok {
		t.Fatal("healthkit key still exists after remove")
	}
	if _, ok := doc.Get("aps-environment"); !ok {
		t.Fatal("aps-environment should remain after remove")
	}

	if err := Remove(path, "healthkit"); err != nil {
		t.Fatalf("Remove(healthkit) second call error = %v", err)
	}
}

func TestListReturnsAllEntries(t *testing.T) {
	t.Parallel()

	path := writeTestEntitlementsFile(t, testPlistXML)

	entries, err := List(path)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("List() returned %d entries, want 3", len(entries))
	}

	byKey := make(map[string]ListedEntitlement, len(entries))
	for _, entry := range entries {
		byKey[entry.Key] = entry
	}

	if entry, ok := byKey["aps-environment"]; !ok {
		t.Fatal("List() missing aps-environment")
	} else if entry.Alias != "push" || entry.Value.StringValue != "development" {
		t.Fatalf("aps-environment entry = %#v, want alias push and value development", entry)
	}

	if entry, ok := byKey["com.apple.developer.healthkit"]; !ok {
		t.Fatal("List() missing healthkit")
	} else if entry.Alias != "healthkit" || !entry.Value.BoolValue {
		t.Fatalf("healthkit entry = %#v, want alias healthkit and true", entry)
	}
}

func writeTestEntitlementsFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "DemoApp.entitlements")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return path
}

const minimalPlistXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>aps-environment</key>
	<string>development</string>
</dict>
</plist>
`
