package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestStrictBuildFlagsAllowCleanSendableCode(t *testing.T) {
	t.Parallel()

	args := strictBuildFlagSwiftcArgs(t)
	output, err := runSwiftTypecheck(t, append(args, writeSwiftFixture(t, "good.swift", `struct Payload: Sendable {
    let value: Int
}

func acceptSendable(_ operation: @Sendable () -> Void) {}

func good(payload: Payload) {
    acceptSendable {
        print(payload.value)
    }
}
`))...)
	if err != nil {
		t.Fatalf("runSwiftTypecheck(clean) error = %v\n%s", err, output)
	}
}

func TestStrictBuildFlagsRejectNonSendableCapture(t *testing.T) {
	t.Parallel()

	args := strictBuildFlagSwiftcArgs(t)
	output, err := runSwiftTypecheck(t, append(args, writeSwiftFixture(t, "bad.swift", `final class Box {
    var value = 0
}

func acceptSendable(_ operation: @Sendable () -> Void) {}

func bad(box: Box) {
    acceptSendable {
        print(box.value)
    }
}
`))...)
	if err == nil {
		t.Fatal("runSwiftTypecheck(non-sendable capture) error = nil, want strict concurrency failure")
	}

	if !strings.Contains(strings.ToLower(output), "non-sendable type 'box'") {
		t.Fatalf("swiftc output missing non-Sendable Box diagnostic:\n%s", output)
	}
	if !strings.Contains(output, "@Sendable") {
		t.Fatalf("swiftc output missing @Sendable diagnostic:\n%s", output)
	}
}

func strictBuildFlagSwiftcArgs(t *testing.T) []string {
	t.Helper()

	if runtime.GOOS != "darwin" {
		t.Skip("strict Swift compiler build-flag tests require macOS")
	}
	if _, err := exec.LookPath("xcrun"); err != nil {
		t.Skip("strict Swift compiler build-flag tests require xcrun")
	}

	args := []string{
		"swiftc",
		"-typecheck",
		"-suppress-warnings",
	}

	for _, setting := range effectiveBuildFlagSettings(config.ProjectConfig{}) {
		switch setting.Key {
		case "SWIFT_VERSION":
			args = append(args, "-swift-version", strings.TrimSuffix(setting.Value, ".0"))
		case "SWIFT_APPROACHABLE_CONCURRENCY":
			if setting.Value != "NO" {
				t.Fatalf("unexpected %s value %q", setting.Key, setting.Value)
			}
		case "SWIFT_DEFAULT_ACTOR_ISOLATION":
			switch setting.Value {
			case "nonisolated":
			case "MainActor":
				args = append(args, "-default-isolation=MainActor")
			default:
				t.Fatalf("unexpected %s value %q", setting.Key, setting.Value)
			}
		case "SWIFT_STRICT_CONCURRENCY":
			if setting.Value != "complete" {
				t.Fatalf("unexpected %s value %q", setting.Key, setting.Value)
			}
			args = append(args, "-strict-concurrency=complete")
		case "SWIFT_UPCOMING_FEATURE_CONCISE_MAGIC_FILE":
			args = appendUpcomingFeatureArg(t, args, setting, "ConciseMagicFile")
		case "SWIFT_UPCOMING_FEATURE_DISABLE_OUTWARD_ACTOR_ISOLATION":
			args = appendUpcomingFeatureArg(t, args, setting, "DisableOutwardActorInference")
		case "SWIFT_UPCOMING_FEATURE_GLOBAL_ACTOR_ISOLATED_TYPES_USABILITY":
			args = appendUpcomingFeatureArg(t, args, setting, "GlobalActorIsolatedTypesUsability")
		case "SWIFT_UPCOMING_FEATURE_INFER_ISOLATED_CONFORMANCES":
			args = appendUpcomingFeatureArg(t, args, setting, "InferIsolatedConformances")
		case "SWIFT_UPCOMING_FEATURE_INFER_SENDABLE_FROM_CAPTURES":
			args = appendUpcomingFeatureModeArg(t, args, setting, "InferSendableFromCaptures")
		case "SWIFT_UPCOMING_FEATURE_GLOBAL_CONCURRENCY":
			args = appendUpcomingFeatureModeArg(t, args, setting, "GlobalConcurrency")
		case "SWIFT_UPCOMING_FEATURE_MEMBER_IMPORT_VISIBILITY":
			args = appendUpcomingFeatureModeArg(t, args, setting, "MemberImportVisibility")
		case "SWIFT_UPCOMING_FEATURE_NONFROZEN_ENUM_EXHAUSTIVITY":
			args = appendUpcomingFeatureModeArg(t, args, setting, "NonfrozenEnumExhaustivity")
		case "SWIFT_UPCOMING_FEATURE_REGION_BASED_ISOLATION":
			args = appendUpcomingFeatureModeArg(t, args, setting, "RegionBasedIsolation")
		case "SWIFT_UPCOMING_FEATURE_EXISTENTIAL_ANY":
			args = appendUpcomingFeatureModeArg(t, args, setting, "ExistentialAny")
		case "SWIFT_UPCOMING_FEATURE_NONISOLATED_NONSENDING_BY_DEFAULT":
			args = appendUpcomingFeatureModeArg(t, args, setting, "NonisolatedNonsendingByDefault")
		default:
			t.Fatalf("unmapped strict build flag %s", setting.Key)
		}
	}

	return args
}

func appendUpcomingFeatureArg(t *testing.T, args []string, setting config.SwiftBuildSetting, feature string) []string {
	t.Helper()

	if setting.Value != "YES" {
		t.Fatalf("unexpected %s value %q", setting.Key, setting.Value)
	}
	return append(args, "-enable-upcoming-feature", feature)
}

func appendUpcomingFeatureModeArg(t *testing.T, args []string, setting config.SwiftBuildSetting, feature string) []string {
	t.Helper()

	switch setting.Value {
	case "YES":
		return append(args, "-enable-upcoming-feature", feature)
	case "MIGRATE":
		return append(args, "-enable-upcoming-feature", feature+":migrate")
	case "NO":
		return args
	default:
		t.Fatalf("unexpected %s value %q", setting.Key, setting.Value)
		return nil
	}
}

func writeSwiftFixture(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return path
}

func runSwiftTypecheck(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command("xcrun", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("xcrun %s: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}
