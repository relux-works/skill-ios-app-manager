package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpShowsAllSubcommands(t *testing.T) {
	output, err := executeRootCommand("--help")
	if err != nil {
		t.Fatalf("executeRootCommand(--help) error = %v", err)
	}

	for _, expected := range []string{
		"init",
		"status",
		"module",
		"dep",
		"entitlements",
		"push",
		"generate",
		"diagram",
		"clean",
		"q",
		"m",
		"ioc",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("help output missing %q:\n%s", expected, output)
		}
	}
}

func TestRootVersionFlag(t *testing.T) {
	SetVersion("0.1.0-test")
	t.Cleanup(func() {
		SetVersion(defaultVersion)
	})

	output, err := executeRootCommand("--version")
	if err != nil {
		t.Fatalf("executeRootCommand(--version) error = %v", err)
	}

	if output != "0.1.0-test\n" {
		t.Fatalf("version output = %q, want %q", output, "0.1.0-test\n")
	}
}

func TestModuleHelpShowsSubcommands(t *testing.T) {
	output, err := executeRootCommand("module", "--help")
	if err != nil {
		t.Fatalf("executeRootCommand(module --help) error = %v", err)
	}

	for _, expected := range []string{"create", "list", "delete"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("module help output missing %q:\n%s", expected, output)
		}
	}
}

func TestDepHelpShowsSubcommands(t *testing.T) {
	output, err := executeRootCommand("dep", "--help")
	if err != nil {
		t.Fatalf("executeRootCommand(dep --help) error = %v", err)
	}

	for _, expected := range []string{"add", "remove", "list"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("dep help output missing %q:\n%s", expected, output)
		}
	}
}

func TestStubCommandsPrintNotImplemented(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{name: "module", args: []string{"module"}},
		{name: "entitlements", args: []string{"entitlements"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := executeRootCommand(tc.args...)
			if err != nil {
				t.Fatalf("executeRootCommand(%v) error = %v", tc.args, err)
			}

			if output != notImplementedMessage+"\n" {
				t.Fatalf("command output = %q, want %q", output, notImplementedMessage+"\n")
			}
		})
	}
}

func executeRootCommand(args ...string) (string, error) {
	return executeRootCommandWithInput("", args...)
}

func executeRootCommandWithInput(input string, args ...string) (string, error) {
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetIn(strings.NewReader(input))
	root.SetArgs(args)

	err := root.Execute()
	return out.String(), err
}
