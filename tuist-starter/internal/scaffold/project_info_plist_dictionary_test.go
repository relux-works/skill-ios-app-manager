package scaffold

import (
	"strconv"
	"strings"
	"testing"
)

func TestSyncProjectManifestInfoPlistDictionaryIgnoresSchemeTargetReferences(t *testing.T) {
	t.Parallel()

	content := `import ProjectDescription

let appName = "DemoApp"

let project = Project(
    name: appName,
    targets: [
        .target(
            name: appName,
            product: .app,
            bundleId: "com.example.demo",
            sources: ["Sources/**"]
        ),
        .target(
            name: "\(appName)UITests",
            product: .uiTests,
            bundleId: "com.example.demo.uitests",
            sources: ["Tests/**"]
        )
    ],
    schemes: [
        .scheme(
            name: "\(appName)UITests",
            buildAction: BuildAction.buildAction(targets: [
                .target(appName),
                .target("\(appName)UITests"),
            ]),
            testAction: TestAction.targets([
                .testableTarget(
                    target: .target("\(appName)UITests"),
                    parallelization: .disabled
                )
            ])
        )
    ]
)
`

	updated, changed, err := syncProjectManifestInfoPlistDictionaryContent(
		content,
		"ApplicationConfiguration",
		true,
		func(indent string) []string {
			return []string{
				indent + strconv.Quote("ApplicationConfiguration") + ": .dictionary([",
				indent + "]),",
			}
		},
	)
	if err != nil {
		t.Fatalf("syncProjectManifestInfoPlistDictionaryContent() error = %v", err)
	}
	if !changed {
		t.Fatal("syncProjectManifestInfoPlistDictionaryContent() changed = false, want true")
	}

	if count := strings.Count(updated, `"ApplicationConfiguration": .dictionary([`); count != 2 {
		t.Fatalf("ApplicationConfiguration count = %d, want target declarations only:\n%s", count, updated)
	}
	if strings.Contains(updated, `buildAction: BuildAction.buildAction(targets: [
                    infoPlist:`) {
		t.Fatalf("sync inserted Info.plist into buildAction target references:\n%s", updated)
	}
	if strings.Contains(updated, `target: .target("\(appName)UITests"),
                            infoPlist:`) {
		t.Fatalf("sync inserted Info.plist into testAction target reference:\n%s", updated)
	}
}

func TestSyncProjectManifestInfoPlistDictionaryKeepsApplicationConfigBeforeAppGroups(t *testing.T) {
	t.Parallel()

	content := `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            product: .app,
            bundleId: "com.example.demo",
            infoPlist: .extendingDefault(
                with: [
                    "AppGroups": .dictionary([
                        "shared": .string("group.com.example.demo.shared"),
                    ]),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            sources: ["Sources/**"]
        )
    ]
)
`

	updated, changed, err := syncProjectManifestInfoPlistDictionaryContent(
		content,
		"ApplicationConfiguration",
		true,
		func(indent string) []string {
			return []string{
				indent + strconv.Quote("ApplicationConfiguration") + ": .dictionary([",
				indent + "]),",
			}
		},
	)
	if err != nil {
		t.Fatalf("syncProjectManifestInfoPlistDictionaryContent() error = %v", err)
	}
	if !changed {
		t.Fatal("syncProjectManifestInfoPlistDictionaryContent() changed = false, want true")
	}

	if !strings.Contains(updated, `"ApplicationConfiguration": .dictionary([
                    ]),
                    "AppGroups": .dictionary([`) {
		t.Fatalf("ApplicationConfiguration should stay before AppGroups:\n%s", updated)
	}

	secondUpdated, secondChanged, err := syncProjectManifestInfoPlistDictionaryContent(
		updated,
		"ApplicationConfiguration",
		true,
		func(indent string) []string {
			return []string{
				indent + strconv.Quote("ApplicationConfiguration") + ": .dictionary([",
				indent + "]),",
			}
		},
	)
	if err != nil {
		t.Fatalf("second syncProjectManifestInfoPlistDictionaryContent() error = %v", err)
	}
	if secondChanged {
		t.Fatalf("second syncProjectManifestInfoPlistDictionaryContent() changed = true:\n%s", secondUpdated)
	}
}
