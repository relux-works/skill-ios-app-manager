package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/push"
)

func TestPushHelpShowsSendAndTokenSubcommands(t *testing.T) {
	t.Parallel()

	output, err := executeRootCommand("push", "--help")
	if err != nil {
		t.Fatalf("executeRootCommand(push --help) error = %v", err)
	}

	for _, expected := range []string{"send", "token", "Push notification tools"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("push --help output missing %q:\n%s", expected, output)
		}
	}
}

func TestPushSendRequiresToken(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	cfg.PushKeyPath = "certs/AuthKey_TEST.p8"
	cfg.PushKeyID = "ABC123DEF4"
	configPath := writeTestConfig(t, cfg)

	_, err := executeRootCommand("push", "--config", configPath, "send")
	if err == nil {
		t.Fatal("executeRootCommand(push send) error = nil, want required token error")
	}

	if !strings.Contains(err.Error(), `required flag(s) "token" not set`) {
		t.Fatalf("push send error = %q, want required token message", err.Error())
	}
}

func TestPushSendUsesConfigAndPassesRequestToSender(t *testing.T) {
	cfg := testProjectConfig()
	cfg.PushKeyPath = "keys/AuthKey_TEST.p8"
	cfg.PushKeyID = "ABC123DEF4"
	configPath := writeTestConfig(t, cfg)

	payloadPath := filepath.Join(t.TempDir(), "payload.json")
	payloadContent := []byte(`{"aps":{"alert":"Hello from test"}}`)
	if err := os.WriteFile(payloadPath, payloadContent, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", payloadPath, err)
	}

	var gotConfig push.SenderConfig
	var gotRequest push.SendRequest

	originalFactory := newPushSender
	newPushSender = func(cfg push.SenderConfig) pushSender {
		gotConfig = cfg
		return &stubPushSender{
			sendFunc: func(_ context.Context, request push.SendRequest) (push.SendResponse, error) {
				gotRequest = request
				return push.SendResponse{
					StatusCode: 200,
					APNSID:     "apns-id-ok",
				}, nil
			},
		}
	}
	t.Cleanup(func() {
		newPushSender = originalFactory
	})

	output, err := executeRootCommand(
		"push",
		"--config",
		configPath,
		"send",
		"--token",
		"001122",
		"--env",
		"prod",
		"--payload",
		payloadPath,
	)
	if err != nil {
		t.Fatalf("executeRootCommand(push send) error = %v", err)
	}

	if !strings.Contains(output, "status=200") {
		t.Fatalf("push send output = %q, want status=200", output)
	}

	expectedKeyPath := filepath.Clean(filepath.Join(filepath.Dir(configPath), cfg.PushKeyPath))
	if gotConfig.KeyPath != expectedKeyPath {
		t.Fatalf("sender config KeyPath = %q, want %q", gotConfig.KeyPath, expectedKeyPath)
	}
	if gotConfig.KeyID != cfg.PushKeyID {
		t.Fatalf("sender config KeyID = %q, want %q", gotConfig.KeyID, cfg.PushKeyID)
	}
	if gotConfig.TeamID != cfg.TeamID {
		t.Fatalf("sender config TeamID = %q, want %q", gotConfig.TeamID, cfg.TeamID)
	}
	if gotConfig.BundleID != cfg.BundleID {
		t.Fatalf("sender config BundleID = %q, want %q", gotConfig.BundleID, cfg.BundleID)
	}

	if gotRequest.DeviceToken != "001122" {
		t.Fatalf("request DeviceToken = %q, want %q", gotRequest.DeviceToken, "001122")
	}
	if gotRequest.Environment != push.EnvironmentProduction {
		t.Fatalf("request Environment = %q, want %q", gotRequest.Environment, push.EnvironmentProduction)
	}
	if string(gotRequest.Payload) != string(payloadContent) {
		t.Fatalf("request Payload = %q, want %q", string(gotRequest.Payload), string(payloadContent))
	}
	if gotRequest.AppName != cfg.AppName {
		t.Fatalf("request AppName = %q, want %q", gotRequest.AppName, cfg.AppName)
	}
}

func TestPushTokenPrintsTokenOnly(t *testing.T) {
	stub := &stubPushTokenExtractor{
		token: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	restorePushTokenExtractorFactory(t, stub)

	output, err := executeRootCommand("push", "token")
	if err != nil {
		t.Fatalf("executeRootCommand(push token) error = %v", err)
	}

	if output != stub.token+"\n" {
		t.Fatalf("push token output = %q, want %q", output, stub.token+"\n")
	}
	if stub.gotFallbackPath != "" {
		t.Fatalf("fallback path = %q, want empty", stub.gotFallbackPath)
	}
}

func TestPushTokenUsesExplicitFallbackFileFlag(t *testing.T) {
	stub := &stubPushTokenExtractor{
		token: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	restorePushTokenExtractorFactory(t, stub)

	output, err := executeRootCommand("push", "token", "--token-file", "./tmp/device-token.txt")
	if err != nil {
		t.Fatalf("executeRootCommand(push token --token-file) error = %v", err)
	}

	if output != stub.token+"\n" {
		t.Fatalf("push token output = %q, want %q", output, stub.token+"\n")
	}
	if stub.gotFallbackPath != "tmp/device-token.txt" {
		t.Fatalf("fallback path = %q, want %q", stub.gotFallbackPath, "tmp/device-token.txt")
	}
}

func TestPushTokenUsesFallbackPathFromConfig(t *testing.T) {
	stub := &stubPushTokenExtractor{
		token: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	restorePushTokenExtractorFactory(t, stub)

	cfg := testProjectConfig()
	cfg.PushTokenPath = "/tmp/app/device-token.log"
	configPath := writeTestConfig(t, cfg)

	output, err := executeRootCommand("push", "--config", configPath, "token")
	if err != nil {
		t.Fatalf("executeRootCommand(push --config token) error = %v", err)
	}

	if output != stub.token+"\n" {
		t.Fatalf("push token output = %q, want %q", output, stub.token+"\n")
	}
	if stub.gotFallbackPath != cfg.PushTokenPath {
		t.Fatalf("fallback path = %q, want %q", stub.gotFallbackPath, cfg.PushTokenPath)
	}
}

func TestPushTokenReturnsExtractorError(t *testing.T) {
	stub := &stubPushTokenExtractor{
		err: errors.New("no APNs device token found"),
	}
	restorePushTokenExtractorFactory(t, stub)

	_, err := executeRootCommand("push", "token")
	if err == nil {
		t.Fatal("executeRootCommand(push token) error = nil, want extractor error")
	}
}

type stubPushSender struct {
	sendFunc func(ctx context.Context, request push.SendRequest) (push.SendResponse, error)
}

func (s *stubPushSender) Send(ctx context.Context, request push.SendRequest) (push.SendResponse, error) {
	return s.sendFunc(ctx, request)
}

type stubPushTokenExtractor struct {
	token           string
	err             error
	gotFallbackPath string
}

func (s *stubPushTokenExtractor) LatestDeviceToken(
	_ context.Context,
	fallbackTokenFilePath string,
) (string, error) {
	s.gotFallbackPath = fallbackTokenFilePath
	if s.err != nil {
		return "", s.err
	}

	return s.token, nil
}

func restorePushTokenExtractorFactory(t *testing.T, stub *stubPushTokenExtractor) {
	t.Helper()

	originalFactory := newPushTokenExtractor
	newPushTokenExtractor = func() pushTokenExtractor {
		return stub
	}

	t.Cleanup(func() {
		newPushTokenExtractor = originalFactory
	})
}
