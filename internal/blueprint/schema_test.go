package blueprint

import (
	"testing"
)

func TestParse(t *testing.T) {
	input := `{
		"name": "Auth",
		"type": "relux-feature",
		"data": {
			"http": true,
			"ws": false
		},
		"ui": {
			"features": ["Login", "Register"],
			"components": ["PasswordField"]
		}
	}`

	bp, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if bp.Name != "Auth" {
		t.Errorf("Name = %q, want %q", bp.Name, "Auth")
	}
	if bp.Type != "relux-feature" {
		t.Errorf("Type = %q, want %q", bp.Type, "relux-feature")
	}
	if !bp.HasHTTP() {
		t.Error("HasHTTP() = false, want true")
	}
	if bp.HasWS() {
		t.Error("HasWS() = true, want false")
	}
	if bp.HasLocal() {
		t.Error("HasLocal() = true, want false")
	}
	if !bp.HasFeatures() {
		t.Error("HasFeatures() = false, want true")
	}
	if !bp.HasComponents() {
		t.Error("HasComponents() = false, want true")
	}

	features := bp.Features()
	if len(features) != 2 {
		t.Fatalf("Features count = %d, want 2", len(features))
	}
	if features[0].Name != "Login" || features[0].NameLower != "login" {
		t.Errorf("Feature[0] = %+v, want Login/login", features[0])
	}
	if features[1].Name != "Register" || features[1].NameLower != "register" {
		t.Errorf("Feature[1] = %+v, want Register/register", features[1])
	}

	components := bp.Components()
	if len(components) != 1 {
		t.Fatalf("Components count = %d, want 1", len(components))
	}
	if components[0].Name != "PasswordField" || components[0].NameLower != "passwordField" {
		t.Errorf("Component[0] = %+v, want PasswordField/passwordField", components[0])
	}
}

func TestParseMinimal(t *testing.T) {
	input := `{"name": "Settings", "type": "relux-feature"}`

	bp, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if bp.HasHTTP() {
		t.Error("HasHTTP() = true, want false")
	}
	if bp.HasUI() {
		t.Error("HasUI() = true, want false")
	}
	if bp.HasFeatures() {
		t.Error("HasFeatures() = true, want false")
	}
}

func TestValidateOK(t *testing.T) {
	bp := &Blueprint{
		Name: "Auth",
		Type: "relux-feature",
		UI: &UIConfig{
			Features:   []string{"Login"},
			Components: []string{"PasswordField"},
		},
	}

	if err := bp.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateErrors(t *testing.T) {
	tests := []struct {
		name string
		bp   Blueprint
		want string
	}{
		{
			name: "empty name",
			bp:   Blueprint{Name: "", Type: "relux-feature"},
			want: "name is required",
		},
		{
			name: "non-PascalCase name",
			bp:   Blueprint{Name: "auth", Type: "relux-feature"},
			want: "must be PascalCase",
		},
		{
			name: "empty type",
			bp:   Blueprint{Name: "Auth", Type: ""},
			want: "type is required",
		},
		{
			name: "unsupported type",
			bp:   Blueprint{Name: "Auth", Type: "feature"},
			want: "unsupported type",
		},
		{
			name: "non-PascalCase feature",
			bp: Blueprint{
				Name: "Auth",
				Type: "relux-feature",
				UI:   &UIConfig{Features: []string{"login"}},
			},
			want: "feature name",
		},
		{
			name: "non-PascalCase component",
			bp: Blueprint{
				Name: "Auth",
				Type: "relux-feature",
				UI:   &UIConfig{Components: []string{"password_field"}},
			},
			want: "component name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bp.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); !contains(got, tt.want) {
				t.Errorf("error = %q, want to contain %q", got, tt.want)
			}
		})
	}
}

func TestEntryPoint(t *testing.T) {
	// Explicit entryPoint
	bp := &Blueprint{
		Name: "Auth",
		Type: "relux-feature",
		UI:   &UIConfig{EntryPoint: "Login", Features: []string{"Login", "Register"}},
	}
	if bp.EntryPoint() != "Login" {
		t.Errorf("EntryPoint() = %q, want %q", bp.EntryPoint(), "Login")
	}
	if !bp.IsEntryPoint("Login") {
		t.Error("IsEntryPoint(Login) = false, want true")
	}
	if bp.IsEntryPoint("Register") {
		t.Error("IsEntryPoint(Register) = true, want false")
	}

	// Default to first feature
	bp2 := &Blueprint{
		Name: "Auth",
		Type: "relux-feature",
		UI:   &UIConfig{Features: []string{"Main", "Settings"}},
	}
	if bp2.EntryPoint() != "Main" {
		t.Errorf("EntryPoint() = %q, want %q (default to first)", bp2.EntryPoint(), "Main")
	}

	// No UI
	bp3 := &Blueprint{Name: "Auth", Type: "relux-feature"}
	if bp3.EntryPoint() != "" {
		t.Errorf("EntryPoint() = %q, want empty", bp3.EntryPoint())
	}
}

func TestValidateEntryPointNotInFeatures(t *testing.T) {
	bp := &Blueprint{
		Name: "Auth",
		Type: "relux-feature",
		UI:   &UIConfig{EntryPoint: "Missing", Features: []string{"Login"}},
	}
	err := bp.Validate()
	if err == nil {
		t.Fatal("expected error for entryPoint not in features")
	}
	if !contains(err.Error(), "entryPoint") {
		t.Errorf("error = %q, want to contain 'entryPoint'", err.Error())
	}
}

func TestDefaultBlueprint(t *testing.T) {
	bp := DefaultBlueprint("Auth")

	if bp.Name != "Auth" {
		t.Errorf("Name = %q, want %q", bp.Name, "Auth")
	}
	if bp.Type != "relux-feature" {
		t.Errorf("Type = %q, want %q", bp.Type, "relux-feature")
	}
	if !bp.HasHTTP() {
		t.Error("HasHTTP() = false, want true")
	}
	if bp.HasWS() {
		t.Error("HasWS() = true, want false")
	}
	if !bp.HasFeatures() {
		t.Error("HasFeatures() = false, want true")
	}
	if bp.EntryPoint() != "Main" {
		t.Errorf("EntryPoint() = %q, want %q", bp.EntryPoint(), "Main")
	}

	if err := bp.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestToJSON(t *testing.T) {
	bp := DefaultBlueprint("Auth")
	data, err := bp.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse roundtrip: %v", err)
	}
	if parsed.Name != bp.Name {
		t.Errorf("roundtrip Name = %q, want %q", parsed.Name, bp.Name)
	}
}

func TestLowerFirst(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Login", "login"},
		{"PasswordField", "passwordField"},
		{"A", "a"},
		{"", ""},
		{"abc", "abc"},
	}

	for _, tt := range tests {
		if got := lowerFirst(tt.input); got != tt.want {
			t.Errorf("lowerFirst(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstr(s, substr)
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
