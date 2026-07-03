package notificationservice

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/extensions"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	defaultBundleIDSuffix       = "notification-service"
	extensionTargetSuffix       = "NotificationServiceExtension"
	extensionCoreSuffix         = "Core"
	extensionsDirectoryName     = "Extensions"
	notificationExtensionPoint  = "com.apple.usernotifications.service"
	userNotificationsDependency = "UserNotifications"
)

var extensionTargetNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// SetupInput holds parameters for notification-service setup.
type SetupInput struct {
	ProjectRoot     string
	AppName         string
	ExtensionTarget string
	BundleIDSuffix  string
}

// Plan is the normalized notification-service setup plan.
type Plan struct {
	ProjectRoot       string
	AppName           string
	ExtensionTarget   string
	CorePackageName   string
	BundleIDSuffix    string
	ExtensionTypeName string
}

// NewPlan validates and normalizes setup input.
func NewPlan(input SetupInput) (Plan, error) {
	if strings.TrimSpace(input.ProjectRoot) == "" {
		return Plan{}, fmt.Errorf("project root is required")
	}
	appName := strings.TrimSpace(input.AppName)
	if appName == "" {
		return Plan{}, fmt.Errorf("app name is required")
	}

	extensionTarget := strings.TrimSpace(input.ExtensionTarget)
	if extensionTarget == "" {
		extensionTarget = scaffold.SwiftTypeName(appName + extensionTargetSuffix)
	} else {
		extensionTarget = scaffold.SwiftTypeName(extensionTarget)
	}
	if err := validateExtensionTargetName(extensionTarget); err != nil {
		return Plan{}, err
	}

	bundleIDSuffix := strings.Trim(strings.TrimSpace(input.BundleIDSuffix), ".")
	if bundleIDSuffix == "" {
		bundleIDSuffix = defaultBundleIDSuffix
	}

	return Plan{
		ProjectRoot:       strings.TrimSpace(input.ProjectRoot),
		AppName:           appName,
		ExtensionTarget:   extensionTarget,
		CorePackageName:   extensionTarget + extensionCoreSuffix,
		BundleIDSuffix:    bundleIDSuffix,
		ExtensionTypeName: scaffold.SwiftTypeName(extensionTarget),
	}, nil
}

// Setup creates a Notification Service Extension target and Core handler package.
func Setup(input SetupInput) error {
	plan, err := NewPlan(input)
	if err != nil {
		return err
	}

	cfg, err := loadProjectConfig(plan.ProjectRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := extensions.MakeAppExtensionProject(extensions.ExtensionProjectInput{
		ProjectRoot:              plan.ProjectRoot,
		ExtensionName:            plan.ExtensionTarget,
		CorePackageName:          plan.CorePackageName,
		BundleIDSuffix:           plan.BundleIDSuffix,
		ExtensionPointIdentifier: notificationExtensionPoint,
		PrincipalClass:           "$(PRODUCT_MODULE_NAME)." + plan.ExtensionTypeName,
		HostBundleID:             strings.TrimSpace(cfg.BundleID),
	}); err != nil {
		return fmt.Errorf("create notification service extension project: %w", err)
	}

	extensionRoot := filepath.Join(plan.ProjectRoot, extensionsDirectoryName, plan.ExtensionTarget)
	wrapperPath := filepath.Join(extensionRoot, "Sources", plan.ExtensionTarget+".swift")
	if err := os.WriteFile(wrapperPath, []byte(notificationServiceWrapperSource(plan)), 0o644); err != nil {
		return fmt.Errorf("write notification service wrapper: %w", err)
	}

	handlerPath := filepath.Join(
		extensionRoot,
		plan.CorePackageName,
		"Sources",
		plan.CorePackageName+"NotificationServiceHandler.swift",
	)
	if err := os.MkdirAll(filepath.Dir(handlerPath), 0o755); err != nil {
		return fmt.Errorf("create notification Core sources directory: %w", err)
	}
	if err := os.WriteFile(handlerPath, []byte(notificationServiceHandlerSource(plan)), 0o644); err != nil {
		return fmt.Errorf("write notification service handler: %w", err)
	}

	projectSwiftPath := filepath.Join(extensionRoot, "Project.swift")
	if err := addUserNotificationsDependency(projectSwiftPath); err != nil {
		return fmt.Errorf("add UserNotifications dependency: %w", err)
	}

	return nil
}

func validateExtensionTargetName(name string) error {
	if !extensionTargetNamePattern.MatchString(name) {
		return fmt.Errorf("extension target name %q is invalid; use letters, digits, or underscore and start with a letter", name)
	}
	return nil
}

func loadProjectConfig(projectRoot string) (config.ProjectConfig, error) {
	cfgPath := filepath.Join(projectRoot, config.DefaultConfigPath)
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return config.ProjectConfig{}, err
	}
	return cfg, nil
}

func addUserNotificationsDependency(projectSwiftPath string) error {
	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    userNotificationsDependency,
		Content: `.sdk(name: "UserNotifications", type: .framework)`,
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func notificationServiceWrapperSource(plan Plan) string {
	return fmt.Sprintf(`import UserNotifications
import %s

final class %s: UNNotificationServiceExtension {
    private let handler = %sNotificationServiceHandler()
    private var contentHandler: ((UNNotificationContent) -> Void)?
    private var bestAttemptContent: UNMutableNotificationContent?

    override func didReceive(
        _ request: UNNotificationRequest,
        withContentHandler contentHandler: @escaping (UNNotificationContent) -> Void
    ) {
        self.contentHandler = contentHandler
        bestAttemptContent = handler.process(request: request, contentHandler: contentHandler)
    }

    override func serviceExtensionTimeWillExpire() {
        handler.serviceExtensionTimeWillExpire(
            bestAttemptContent: bestAttemptContent,
            contentHandler: contentHandler
        )
    }
}
`, plan.CorePackageName, plan.ExtensionTypeName, plan.CorePackageName)
}

func notificationServiceHandlerSource(plan Plan) string {
	return fmt.Sprintf(`import Foundation
import UserNotifications

public final class %sNotificationServiceHandler {
    public init() {}

    @discardableResult
    public func process(
        request: UNNotificationRequest,
        contentHandler: @escaping (UNNotificationContent) -> Void
    ) -> UNMutableNotificationContent {
        let bestAttemptContent = (
            request.content.mutableCopy() as? UNMutableNotificationContent
        ) ?? UNMutableNotificationContent()

        contentHandler(bestAttemptContent)
        return bestAttemptContent
    }

    public func serviceExtensionTimeWillExpire(
        bestAttemptContent: UNMutableNotificationContent?,
        contentHandler: ((UNNotificationContent) -> Void)?
    ) {
        guard let bestAttemptContent, let contentHandler else {
            return
        }
        contentHandler(bestAttemptContent)
    }
}
`, plan.CorePackageName)
}
