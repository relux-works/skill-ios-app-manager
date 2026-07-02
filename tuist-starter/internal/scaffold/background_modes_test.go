package scaffold

import (
	"strings"
	"testing"
)

const backgroundModesManifestFixture = `let project = Project(
    targets: [
        .target(
            name: appName,
            infoPlist: .extendingDefault(
                with: [
                    "CFBundleDisplayName": .string(appName),
                    "NSMicrophoneUsageDescription": .string("Mic usage"),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
        ),
    ]
)
`

func TestSyncBackgroundModesManifestInsertsBeforeLaunchScreen(t *testing.T) {
	updated, changed, err := syncBackgroundModesManifest(
		backgroundModesManifestFixture,
		[]string{"audio", "push-to-talk"},
	)
	if err != nil {
		t.Fatalf("syncBackgroundModesManifest() error = %v", err)
	}
	if !changed {
		t.Fatal("expected manifest to change")
	}

	wantLine := `"UIBackgroundModes": .array([.string("audio"), .string("push-to-talk")]),`
	if !strings.Contains(updated, wantLine) {
		t.Fatalf("updated manifest missing %q:\n%s", wantLine, updated)
	}

	modesIndex := strings.Index(updated, `"UIBackgroundModes"`)
	launchIndex := strings.Index(updated, `"UILaunchScreen"`)
	if modesIndex < 0 || launchIndex < 0 || modesIndex > launchIndex {
		t.Fatalf("UIBackgroundModes not inserted before UILaunchScreen:\n%s", updated)
	}
}

func TestSyncBackgroundModesManifestReplacesDriftedEntryInPlace(t *testing.T) {
	seeded, _, err := syncBackgroundModesManifest(backgroundModesManifestFixture, []string{"audio"})
	if err != nil {
		t.Fatalf("seed error = %v", err)
	}

	updated, changed, err := syncBackgroundModesManifest(seeded, []string{"audio", "voip"})
	if err != nil {
		t.Fatalf("syncBackgroundModesManifest() error = %v", err)
	}
	if !changed {
		t.Fatal("expected drifted manifest to change")
	}
	if strings.Count(updated, `"UIBackgroundModes"`) != 1 {
		t.Fatalf("expected exactly one UIBackgroundModes entry:\n%s", updated)
	}
	if !strings.Contains(updated, `.array([.string("audio"), .string("voip")]),`) {
		t.Fatalf("updated manifest missing replaced modes:\n%s", updated)
	}
}

func TestSyncBackgroundModesManifestIsIdempotent(t *testing.T) {
	seeded, _, err := syncBackgroundModesManifest(backgroundModesManifestFixture, []string{"audio"})
	if err != nil {
		t.Fatalf("seed error = %v", err)
	}

	updated, changed, err := syncBackgroundModesManifest(seeded, []string{"audio"})
	if err != nil {
		t.Fatalf("syncBackgroundModesManifest() error = %v", err)
	}
	if changed {
		t.Fatal("expected up-to-date manifest to be a no-op")
	}
	if updated != seeded {
		t.Fatal("expected no-op to preserve content")
	}
}

func TestSyncBackgroundModesManifestRemovesEntryWhenConfigEmpty(t *testing.T) {
	seeded, _, err := syncBackgroundModesManifest(backgroundModesManifestFixture, []string{"audio"})
	if err != nil {
		t.Fatalf("seed error = %v", err)
	}

	updated, changed, err := syncBackgroundModesManifest(seeded, nil)
	if err != nil {
		t.Fatalf("syncBackgroundModesManifest() error = %v", err)
	}
	if !changed {
		t.Fatal("expected removal to change the manifest")
	}
	if strings.Contains(updated, "UIBackgroundModes") {
		t.Fatalf("expected UIBackgroundModes to be removed:\n%s", updated)
	}

	again, changedAgain, err := syncBackgroundModesManifest(updated, nil)
	if err != nil {
		t.Fatalf("second removal error = %v", err)
	}
	if changedAgain || again != updated {
		t.Fatal("expected removal on clean manifest to be a no-op")
	}
}

func TestSyncBackgroundModesManifestSkipsManifestsWithoutAppInfoPlist(t *testing.T) {
	// Extension manifests (widgets, app intents) have no UILaunchScreen anchor;
	// the sync must leave them untouched instead of failing.
	content := "let project = Project()\n"
	updated, changed, err := syncBackgroundModesManifest(content, []string{"audio"})
	if err != nil {
		t.Fatalf("syncBackgroundModesManifest() error = %v", err)
	}
	if changed || updated != content {
		t.Fatal("expected non-app manifest to be skipped")
	}
}
