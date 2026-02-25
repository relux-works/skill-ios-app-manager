package tuistproj

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestTuistRunnerGenerateForwardsExecutionOptions(t *testing.T) {
	t.Parallel()

	var gotBinary string
	var gotArgs []string
	var gotWorkingDir string
	var gotVerbose bool
	var gotStdout io.Writer
	var gotStderr io.Writer

	stdoutSink := &bytes.Buffer{}
	stderrSink := &bytes.Buffer{}

	runner := NewTuistRunner(
		WithBinary("custom-tuist"),
		WithWorkingDir("/tmp/project"),
		WithVerbose(true),
		WithStdout(stdoutSink),
		WithStderr(stderrSink),
		WithExecFunc(func(
			_ context.Context,
			binary string,
			args []string,
			workingDir string,
			verbose bool,
			stdout io.Writer,
			stderr io.Writer,
		) (ExecOutput, error) {
			gotBinary = binary
			gotArgs = append([]string(nil), args...)
			gotWorkingDir = workingDir
			gotVerbose = verbose
			gotStdout = stdout
			gotStderr = stderr

			return ExecOutput{
				Stdout:   "ok",
				Stderr:   "",
				ExitCode: 0,
			}, nil
		}),
	)

	result, err := runner.Generate(context.Background(), "--path", "Project.swift")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if gotBinary != "custom-tuist" {
		t.Fatalf("binary = %q, want %q", gotBinary, "custom-tuist")
	}

	wantArgs := []string{"generate", "--path", "Project.swift"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}

	if gotWorkingDir != "/tmp/project" {
		t.Fatalf("working dir = %q, want %q", gotWorkingDir, "/tmp/project")
	}

	if !gotVerbose {
		t.Fatalf("verbose = false, want true")
	}

	if gotStdout != stdoutSink {
		t.Fatalf("stdout writer mismatch")
	}

	if gotStderr != stderrSink {
		t.Fatalf("stderr writer mismatch")
	}

	if result.Command != "generate" {
		t.Fatalf("result command = %q, want %q", result.Command, "generate")
	}

	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestTuistRunnerReturnsStructuredRunError(t *testing.T) {
	t.Parallel()

	execErr := errors.New("exit status 2")
	runner := NewTuistRunner(
		WithExecFunc(func(
			_ context.Context,
			_ string,
			_ []string,
			_ string,
			_ bool,
			_ io.Writer,
			_ io.Writer,
		) (ExecOutput, error) {
			return ExecOutput{
				Stdout:   "partial output",
				Stderr:   "something failed",
				ExitCode: 2,
			}, execErr
		}),
	)

	result, err := runner.Clean(context.Background(), "--all")
	if err == nil {
		t.Fatalf("Clean() error = nil, want non-nil")
	}

	var runErr *RunError
	if !errors.As(err, &runErr) {
		t.Fatalf("error type = %T, want *RunError", err)
	}

	if runErr.ExitCode != 2 {
		t.Fatalf("runErr exit code = %d, want 2", runErr.ExitCode)
	}

	if runErr.Command != "clean" {
		t.Fatalf("runErr command = %q, want %q", runErr.Command, "clean")
	}

	if runErr.Stderr != "something failed" {
		t.Fatalf("runErr stderr = %q, want %q", runErr.Stderr, "something failed")
	}

	if !errors.Is(runErr, execErr) {
		t.Fatalf("runErr should wrap original exec error")
	}

	if result.ExitCode != 2 {
		t.Fatalf("result exit code = %d, want 2", result.ExitCode)
	}
}

func TestTuistRunnerRejectsUnsupportedCommand(t *testing.T) {
	t.Parallel()

	runner := NewTuistRunner()
	_, err := runner.Run(context.Background(), "something-else")
	if err == nil {
		t.Fatalf("Run() error = nil, want non-nil")
	}

	var invalid *InvalidCommandError
	if !errors.As(err, &invalid) {
		t.Fatalf("error type = %T, want *InvalidCommandError", err)
	}
}
