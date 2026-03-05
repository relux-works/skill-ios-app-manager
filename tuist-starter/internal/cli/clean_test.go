package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	cleaner "github.com/relux-works/ios-app-manager/internal/clean"
)

type cleanManagerStub struct {
	quickResult cleaner.Result
	quickErr    error
	deepResult  cleaner.Result
	deepErr     error
	killErr     error

	quickRoots []string
	deepRoots  []string
	killCalls  int
}

func (s *cleanManagerStub) QuickClean(projectRoot string) (cleaner.Result, error) {
	s.quickRoots = append(s.quickRoots, projectRoot)
	return s.quickResult, s.quickErr
}

func (s *cleanManagerStub) DeepClean(projectRoot string) (cleaner.Result, error) {
	s.deepRoots = append(s.deepRoots, projectRoot)
	return s.deepResult, s.deepErr
}

func (s *cleanManagerStub) KillXcode() error {
	s.killCalls++
	return s.killErr
}

func TestCleanCommandRunsQuickCleanByDefault(t *testing.T) {
	stub := &cleanManagerStub{
		quickResult: cleaner.Result{
			CleanedPaths: []string{"/tmp/project/DerivedData", "/tmp/project/.build"},
			FreedBytes:   2048,
		},
	}

	restore := installCleanCommandStubs(t, stub, "/tmp/project")
	defer restore()

	stdout, stderr, err := executeRootCommandWithStreams("clean")
	if err != nil {
		t.Fatalf("executeRootCommand(clean) error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if len(stub.quickRoots) != 1 || stub.quickRoots[0] != "/tmp/project" {
		t.Fatalf("quickRoots = %#v, want [/tmp/project]", stub.quickRoots)
	}
	if len(stub.deepRoots) != 0 {
		t.Fatalf("deepRoots = %#v, want none", stub.deepRoots)
	}
	if stub.killCalls != 0 {
		t.Fatalf("killCalls = %d, want 0", stub.killCalls)
	}
	for _, expected := range []string{
		"quick clean completed",
		"- /tmp/project/DerivedData",
		"- /tmp/project/.build",
		"estimated freed space: 2.0 KB",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("stdout missing %q:\n%s", expected, stdout)
		}
	}
}

func TestCleanCommandRunsDeepCleanWhenFlagSet(t *testing.T) {
	stub := &cleanManagerStub{
		deepResult: cleaner.Result{
			CleanedPaths: []string{
				"/tmp/project/DerivedData",
				"/tmp/project/.build",
				"/Users/test/Library/Developer/Xcode/DerivedData",
			},
			FreedBytes: 3072,
		},
	}

	restore := installCleanCommandStubs(t, stub, "/tmp/project")
	defer restore()

	stdout, stderr, err := executeRootCommandWithStreams("clean", "--deep")
	if err != nil {
		t.Fatalf("executeRootCommand(clean --deep) error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if len(stub.deepRoots) != 1 || stub.deepRoots[0] != "/tmp/project" {
		t.Fatalf("deepRoots = %#v, want [/tmp/project]", stub.deepRoots)
	}
	if len(stub.quickRoots) != 0 {
		t.Fatalf("quickRoots = %#v, want none", stub.quickRoots)
	}
	if stub.killCalls != 0 {
		t.Fatalf("killCalls = %d, want 0", stub.killCalls)
	}
	if !strings.Contains(stdout, "deep clean completed") {
		t.Fatalf("stdout missing deep confirmation:\n%s", stdout)
	}
}

func TestCleanCommandKillXcodeImpliesDeepAndWarnsOnFailure(t *testing.T) {
	stub := &cleanManagerStub{
		killErr: errors.New("pkill failed"),
		deepResult: cleaner.Result{
			CleanedPaths: []string{"/tmp/project/DerivedData"},
			FreedBytes:   1024,
		},
	}

	restore := installCleanCommandStubs(t, stub, "/tmp/project")
	defer restore()

	stdout, stderr, err := executeRootCommandWithStreams("clean", "--kill-xcode")
	if err != nil {
		t.Fatalf("executeRootCommand(clean --kill-xcode) error = %v", err)
	}
	if stub.killCalls != 1 {
		t.Fatalf("killCalls = %d, want 1", stub.killCalls)
	}
	if len(stub.deepRoots) != 1 {
		t.Fatalf("deepRoots = %#v, want exactly one deep clean call", stub.deepRoots)
	}
	if !strings.Contains(stderr, "warning: unable to kill Xcode before clean") {
		t.Fatalf("stderr missing warning:\n%s", stderr)
	}
	if !strings.Contains(stdout, "deep clean completed") {
		t.Fatalf("stdout missing deep clean confirmation:\n%s", stdout)
	}
}

func installCleanCommandStubs(t *testing.T, stub cleanManager, cwd string) func() {
	t.Helper()

	originalFactory := cleanManagerFactory
	originalGetwd := cleanGetwd
	cleanManagerFactory = func() cleanManager {
		return stub
	}
	cleanGetwd = func() (string, error) {
		return cwd, nil
	}

	return func() {
		cleanManagerFactory = originalFactory
		cleanGetwd = originalGetwd
	}
}

func executeRootCommandWithStreams(args ...string) (string, string, error) {
	root := NewRootCommand()
	var out bytes.Buffer
	var errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)

	err := root.Execute()
	return out.String(), errOut.String(), err
}
