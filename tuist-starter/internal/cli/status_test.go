package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/components"
	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestStatusCommandRendersProjectSummary(t *testing.T) {
	t.Parallel()

	manager := &statusAppManagerStub{
		status: &components.ProjectStatus{
			ConfigPath: "custom.json",
			Config: config.ProjectConfig{
				AppName:          "DemoApp",
				ProductName:      "Demo Product",
				BundleID:         "com.example.demo",
				TeamID:           "ABCDE12345",
				MinTarget:        "16.0",
				SwiftVersion:     "6.2",
				MarketingVersion: "1.2.3",
				ProjectVersion:   "123",
			},
			ModulesPath:           "Packages",
			Modules:               []string{"Auth", "Utilities"},
			DependencyGraphHealth: "unknown",
		},
	}

	output, err := executeRootCommandWithAppManager(manager, "--config", "custom.json", "status")
	if err != nil {
		t.Fatalf("executeRootCommand(status) error = %v", err)
	}

	if manager.initConfigPath != "custom.json" {
		t.Fatalf("initConfigPath = %q, want custom.json", manager.initConfigPath)
	}

	for _, want := range []string{
		"project:",
		"config: custom.json",
		"app: DemoApp",
		"product: Demo Product",
		"bundle: com.example.demo",
		"modules:",
		"count: 2",
		"- Auth",
		"- Utilities",
		"dependency graph: unknown",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("status output missing %q:\n%s", want, output)
		}
	}
}

func TestStatusCommandWrapsConfigErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("missing config")
	manager := &statusAppManagerStub{initErr: wantErr}

	_, err := executeRootCommandWithAppManager(manager, "status")
	if err == nil {
		t.Fatal("executeRootCommand(status) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "load project config") {
		t.Fatalf("error = %q, want load project config wrapper", err.Error())
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("error should wrap %q", wantErr.Error())
	}
}

func executeRootCommandWithAppManager(appManager components.AppManager, args ...string) (string, error) {
	root := NewRootCommandWithAppManager(appManager)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetIn(strings.NewReader(""))
	root.SetArgs(args)

	err := root.Execute()
	return out.String(), err
}

type statusAppManagerStub struct {
	initConfigPath string
	initErr        error
	status         *components.ProjectStatus
	statusErr      error
}

func (s *statusAppManagerStub) Init(_ context.Context, configPath string) error {
	s.initConfigPath = configPath
	return s.initErr
}

func (s *statusAppManagerStub) Status(_ context.Context) (*components.ProjectStatus, error) {
	if s.statusErr != nil {
		return nil, s.statusErr
	}
	return s.status, nil
}

func (s *statusAppManagerStub) CreateModule(_ context.Context, _ string, _ string) error {
	return nil
}

func (s *statusAppManagerStub) DeleteModule(_ context.Context, _ string) error {
	return nil
}
