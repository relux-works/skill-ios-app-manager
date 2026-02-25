package tuistproj

import (
	"context"
	"errors"
	"testing"
)

func TestParseVersionOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		want   VersionInfo
	}{
		{
			name:   "plain semver",
			output: "4.12.3\n",
			want: VersionInfo{
				Raw:     "4.12.3",
				Version: "4.12.3",
				Major:   4,
				Minor:   12,
				Patch:   3,
			},
		},
		{
			name:   "prefixed output",
			output: "Tuist 4.2.1 (abc123)",
			want: VersionInfo{
				Raw:     "Tuist 4.2.1 (abc123)",
				Version: "4.2.1",
				Major:   4,
				Minor:   2,
				Patch:   1,
			},
		},
		{
			name:   "missing patch defaults to zero",
			output: "version: 4.7",
			want: VersionInfo{
				Raw:     "version: 4.7",
				Version: "4.7.0",
				Major:   4,
				Minor:   7,
				Patch:   0,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseVersionOutput(test.output)
			if err != nil {
				t.Fatalf("ParseVersionOutput() error = %v", err)
			}

			if got != test.want {
				t.Fatalf("ParseVersionOutput() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseVersionOutputError(t *testing.T) {
	t.Parallel()

	_, err := ParseVersionOutput("local-dev-build")
	if err == nil {
		t.Fatalf("ParseVersionOutput() error = nil, want non-nil")
	}

	var parseErr *VersionParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error type = %T, want *VersionParseError", err)
	}
}

func TestCheckVersion(t *testing.T) {
	t.Parallel()

	runner := mockRunner{
		runFn: func(_ context.Context, command string, extraArgs ...string) (RunResult, error) {
			if command != CommandVersion {
				t.Fatalf("command = %q, want %q", command, CommandVersion)
			}
			if len(extraArgs) != 0 {
				t.Fatalf("extra args = %#v, want empty", extraArgs)
			}
			return RunResult{
				Stdout:   "Tuist 4.30.1",
				ExitCode: 0,
			}, nil
		},
	}

	info, err := CheckVersion(context.Background(), runner)
	if err != nil {
		t.Fatalf("CheckVersion() error = %v", err)
	}

	if info.Version != "4.30.1" {
		t.Fatalf("version = %q, want %q", info.Version, "4.30.1")
	}
}

func TestCheckVersionUnsupported(t *testing.T) {
	t.Parallel()

	runner := mockRunner{
		runFn: func(_ context.Context, _ string, _ ...string) (RunResult, error) {
			return RunResult{Stdout: "Tuist 3.44.0"}, nil
		},
	}

	info, err := CheckVersion(context.Background(), runner)
	if err == nil {
		t.Fatalf("CheckVersion() error = nil, want non-nil")
	}

	if info.Version != "3.44.0" {
		t.Fatalf("detected version = %q, want %q", info.Version, "3.44.0")
	}

	var unsupported *VersionUnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error type = %T, want *VersionUnsupportedError", err)
	}
}
