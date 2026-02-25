package tuistproj

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	// DefaultBinary is the default tuist executable name used by the runner.
	DefaultBinary = "tuist"

	// Supported tuist commands.
	CommandGenerate = "generate"
	CommandInstall  = "install"
	CommandGraph    = "graph"
	CommandClean    = "clean"
	CommandEdit     = "edit"
	CommandDump     = "dump"
	CommandVersion  = "version"
)

var supportedCommands = map[string]struct{}{
	CommandGenerate: {},
	CommandInstall:  {},
	CommandGraph:    {},
	CommandClean:    {},
	CommandEdit:     {},
	CommandDump:     {},
	CommandVersion:  {},
}

// Runner is the interface consumed by version and graph helpers.
type Runner interface {
	Run(ctx context.Context, command string, extraArgs ...string) (RunResult, error)
}

// ExecOutput describes captured subprocess output.
type ExecOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExecFunc is the execution primitive used by TuistRunner.
// It is injectable for tests.
type ExecFunc func(
	ctx context.Context,
	binary string,
	args []string,
	workingDir string,
	verbose bool,
	stdout io.Writer,
	stderr io.Writer,
) (ExecOutput, error)

// RunnerOption configures TuistRunner.
type RunnerOption func(*TuistRunner)

// RunResult contains a completed tuist invocation result.
type RunResult struct {
	Binary     string
	Command    string
	Args       []string
	WorkingDir string
	Stdout     string
	Stderr     string
	ExitCode   int
}

// RunError captures structured subprocess failure details.
type RunError struct {
	Binary     string
	Command    string
	Args       []string
	WorkingDir string
	Stdout     string
	Stderr     string
	ExitCode   int
	Err        error
}

func (e *RunError) Error() string {
	if e == nil {
		return "<nil>"
	}

	commandParts := make([]string, 0, len(e.Args)+2)
	commandParts = append(commandParts, e.Binary, e.Command)
	commandParts = append(commandParts, e.Args...)
	invocation := strings.Join(commandParts, " ")

	if e.ExitCode >= 0 {
		return fmt.Sprintf("%s failed with exit code %d: %v", invocation, e.ExitCode, e.Err)
	}
	return fmt.Sprintf("%s failed: %v", invocation, e.Err)
}

// Unwrap returns the underlying execution error.
func (e *RunError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// InvalidCommandError is returned when an unsupported tuist command is requested.
type InvalidCommandError struct {
	Command string
}

func (e *InvalidCommandError) Error() string {
	return fmt.Sprintf("unsupported tuist command: %q", e.Command)
}

// TuistRunner executes tuist CLI commands.
type TuistRunner struct {
	binary     string
	workingDir string
	verbose    bool
	stdout     io.Writer
	stderr     io.Writer
	execFn     ExecFunc
}

// NewTuistRunner creates a runner with sane defaults.
func NewTuistRunner(options ...RunnerOption) *TuistRunner {
	runner := &TuistRunner{
		binary: DefaultBinary,
		execFn: defaultExecFunc,
	}

	for _, option := range options {
		if option != nil {
			option(runner)
		}
	}

	return runner
}

// WithBinary overrides the tuist binary path/name.
func WithBinary(binary string) RunnerOption {
	return func(r *TuistRunner) {
		if strings.TrimSpace(binary) != "" {
			r.binary = binary
		}
	}
}

// WithWorkingDir sets the working directory for command execution.
func WithWorkingDir(workingDir string) RunnerOption {
	return func(r *TuistRunner) {
		r.workingDir = strings.TrimSpace(workingDir)
	}
}

// WithVerbose toggles realtime subprocess output streaming.
func WithVerbose(verbose bool) RunnerOption {
	return func(r *TuistRunner) {
		r.verbose = verbose
	}
}

// WithStdout sets the destination used for verbose stdout streaming.
func WithStdout(stdout io.Writer) RunnerOption {
	return func(r *TuistRunner) {
		r.stdout = stdout
	}
}

// WithStderr sets the destination used for verbose stderr streaming.
func WithStderr(stderr io.Writer) RunnerOption {
	return func(r *TuistRunner) {
		r.stderr = stderr
	}
}

// WithExecFunc replaces subprocess execution. Intended for unit tests.
func WithExecFunc(execFn ExecFunc) RunnerOption {
	return func(r *TuistRunner) {
		if execFn != nil {
			r.execFn = execFn
		}
	}
}

// Run executes a supported tuist command.
func (r *TuistRunner) Run(ctx context.Context, command string, extraArgs ...string) (RunResult, error) {
	normalized := strings.ToLower(strings.TrimSpace(command))
	if _, ok := supportedCommands[normalized]; !ok {
		return RunResult{}, &InvalidCommandError{Command: command}
	}

	args := make([]string, 0, len(extraArgs)+1)
	args = append(args, normalized)
	args = append(args, extraArgs...)

	output, err := r.execFn(ctx, r.binary, args, r.workingDir, r.verbose, r.stdout, r.stderr)
	result := RunResult{
		Binary:     r.binary,
		Command:    normalized,
		Args:       append([]string(nil), extraArgs...),
		WorkingDir: r.workingDir,
		Stdout:     output.Stdout,
		Stderr:     output.Stderr,
		ExitCode:   output.ExitCode,
	}

	if err != nil {
		return result, &RunError{
			Binary:     result.Binary,
			Command:    result.Command,
			Args:       append([]string(nil), result.Args...),
			WorkingDir: result.WorkingDir,
			Stdout:     result.Stdout,
			Stderr:     result.Stderr,
			ExitCode:   result.ExitCode,
			Err:        err,
		}
	}

	return result, nil
}

// Generate executes `tuist generate`.
func (r *TuistRunner) Generate(ctx context.Context, extraArgs ...string) (RunResult, error) {
	return r.Run(ctx, CommandGenerate, extraArgs...)
}

// Install executes `tuist install`.
func (r *TuistRunner) Install(ctx context.Context, extraArgs ...string) (RunResult, error) {
	return r.Run(ctx, CommandInstall, extraArgs...)
}

// Graph executes `tuist graph`.
func (r *TuistRunner) Graph(ctx context.Context, extraArgs ...string) (RunResult, error) {
	return r.Run(ctx, CommandGraph, extraArgs...)
}

// Clean executes `tuist clean`.
func (r *TuistRunner) Clean(ctx context.Context, extraArgs ...string) (RunResult, error) {
	return r.Run(ctx, CommandClean, extraArgs...)
}

// Edit executes `tuist edit`.
func (r *TuistRunner) Edit(ctx context.Context, extraArgs ...string) (RunResult, error) {
	return r.Run(ctx, CommandEdit, extraArgs...)
}

// Dump executes `tuist dump`.
func (r *TuistRunner) Dump(ctx context.Context, extraArgs ...string) (RunResult, error) {
	return r.Run(ctx, CommandDump, extraArgs...)
}

// Version executes `tuist version`.
func (r *TuistRunner) Version(ctx context.Context, extraArgs ...string) (RunResult, error) {
	return r.Run(ctx, CommandVersion, extraArgs...)
}

func defaultExecFunc(
	ctx context.Context,
	binary string,
	args []string,
	workingDir string,
	verbose bool,
	stdout io.Writer,
	stderr io.Writer,
) (ExecOutput, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = workingDir

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer

	if verbose {
		verboseStdout := stdout
		if verboseStdout == nil {
			verboseStdout = os.Stdout
		}

		verboseStderr := stderr
		if verboseStderr == nil {
			verboseStderr = os.Stderr
		}

		cmd.Stdout = io.MultiWriter(&stdoutBuffer, verboseStdout)
		cmd.Stderr = io.MultiWriter(&stderrBuffer, verboseStderr)
	} else {
		cmd.Stdout = &stdoutBuffer
		cmd.Stderr = &stderrBuffer
	}

	err := cmd.Run()
	output := ExecOutput{
		Stdout:   stdoutBuffer.String(),
		Stderr:   stderrBuffer.String(),
		ExitCode: 0,
	}

	if err != nil {
		output.ExitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			output.ExitCode = exitErr.ExitCode()
		}
		return output, err
	}

	return output, nil
}
