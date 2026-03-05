package entitlements

import (
	"strings"
	"testing"
)

func TestParsePlistXMLSupportsStringBoolAndArrayValues(t *testing.T) {
	t.Parallel()

	doc, err := ParsePlistXML([]byte(testPlistXML))
	if err != nil {
		t.Fatalf("ParsePlistXML() error = %v", err)
	}

	apsEnvironment, ok := doc.Get("aps-environment")
	if !ok {
		t.Fatal("plist missing aps-environment")
	}
	if apsEnvironment.Kind != ValueKindString || apsEnvironment.StringValue != "development" {
		t.Fatalf("aps-environment = %#v, want string development", apsEnvironment)
	}

	healthkit, ok := doc.Get("com.apple.developer.healthkit")
	if !ok {
		t.Fatal("plist missing com.apple.developer.healthkit")
	}
	if healthkit.Kind != ValueKindBool || !healthkit.BoolValue {
		t.Fatalf("com.apple.developer.healthkit = %#v, want bool true", healthkit)
	}

	appGroups, ok := doc.Get("com.apple.security.application-groups")
	if !ok {
		t.Fatal("plist missing com.apple.security.application-groups")
	}
	if appGroups.Kind != ValueKindStringArray {
		t.Fatalf("app groups kind = %d, want %d", appGroups.Kind, ValueKindStringArray)
	}
	if len(appGroups.ArrayValue) != 2 {
		t.Fatalf("app groups count = %d, want 2", len(appGroups.ArrayValue))
	}
	if appGroups.ArrayValue[0] != "group.com.example.demo" || appGroups.ArrayValue[1] != "group.com.example.shared" {
		t.Fatalf("app groups = %#v, want expected values", appGroups.ArrayValue)
	}
}

func TestMarshalPlistXMLRoundTrip(t *testing.T) {
	t.Parallel()

	doc, err := ParsePlistXML([]byte(testPlistXML))
	if err != nil {
		t.Fatalf("ParsePlistXML() error = %v", err)
	}

	doc.Set("UIBackgroundModes", Value{
		Kind:       ValueKindStringArray,
		ArrayValue: []string{"remote-notification", "fetch"},
	})

	payload, err := MarshalPlistXML(doc)
	if err != nil {
		t.Fatalf("MarshalPlistXML() error = %v", err)
	}

	roundTrip, err := ParsePlistXML(payload)
	if err != nil {
		t.Fatalf("ParsePlistXML(round trip) error = %v", err)
	}

	for _, key := range []string{
		"aps-environment",
		"com.apple.developer.healthkit",
		"com.apple.security.application-groups",
		"UIBackgroundModes",
	} {
		if _, ok := roundTrip.Get(key); !ok {
			t.Fatalf("round trip plist missing key %q", key)
		}
	}

	serialized := string(payload)
	for _, snippet := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`,
		`<plist version="1.0">`,
		`<dict>`,
		`<key>UIBackgroundModes</key>`,
		`<string>remote-notification</string>`,
		`</plist>`,
	} {
		if !strings.Contains(serialized, snippet) {
			t.Fatalf("serialized plist missing %q:\n%s", snippet, serialized)
		}
	}
}

func TestParsePlistXMLUnsupportedValueTypeReturnsError(t *testing.T) {
	t.Parallel()

	input := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>some-key</key>
	<integer>1</integer>
</dict>
</plist>`

	_, err := ParsePlistXML([]byte(input))
	if err == nil {
		t.Fatal("ParsePlistXML() error = nil, want unsupported type error")
	}
	if !strings.Contains(err.Error(), "unsupported plist value type") {
		t.Fatalf("ParsePlistXML() error = %v, want unsupported type message", err)
	}
}

const testPlistXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>aps-environment</key>
	<string>development</string>
	<key>com.apple.developer.healthkit</key>
	<true/>
	<key>com.apple.security.application-groups</key>
	<array>
		<string>group.com.example.demo</string>
		<string>group.com.example.shared</string>
	</array>
</dict>
</plist>
`
