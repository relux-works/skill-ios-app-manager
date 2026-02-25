package tuistproj

import "context"

type mockRunner struct {
	runFn func(ctx context.Context, command string, extraArgs ...string) (RunResult, error)
}

func (m mockRunner) Run(ctx context.Context, command string, extraArgs ...string) (RunResult, error) {
	if m.runFn == nil {
		return RunResult{}, nil
	}
	return m.runFn(ctx, command, extraArgs...)
}
