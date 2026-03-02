package entitlements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	} else if entry.Value.StringValue != "development" {
		t.Fatalf("aps-environment entry = %#v, want value development", entry)
	}

	if entry, ok := byKey["com.apple.developer.healthkit"]; !ok {
		t.Fatal("List() missing healthkit")
	} else if !entry.Value.BoolValue {
		t.Fatalf("healthkit entry = %#v, want true", entry)
	}

	if entry, ok := byKey["com.apple.security.application-groups"]; !ok {
		t.Fatal("List() missing app groups")
	} else if len(entry.Value.ArrayValue) != 2 {
		t.Fatalf("app groups entry = %#v, want 2 values", entry)
	}
}

func TestListReturnsEmptyForEmptyPlist(t *testing.T) {
	t.Parallel()

	path := writeTestEntitlementsFile(t, emptyPlistXML)

	entries, err := List(path)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("List() returned %d entries, want 0", len(entries))
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

const emptyPlistXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
</dict>
</plist>
`
