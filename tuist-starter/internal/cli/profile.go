package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/profile"
	"github.com/spf13/cobra"
)

func newProfileCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Profile build, runtime, and rendered layout diagnostics",
	}
	cmd.AddCommand(
		newProfileBuildCommand(opts),
		newProfileLayoutCommand(opts),
		newProfileRuntimeCommand(opts),
	)
	return cmd
}

func newProfileBuildCommand(opts *RootOptions) *cobra.Command {
	var configPath string
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		configPath = strings.TrimSpace(opts.ConfigPath)
	} else {
		configPath = config.DefaultConfigPath
	}

	var workspace string
	var scheme string
	var configuration string
	var destination string
	var derivedDataPath string
	var resultBundlePath string
	var logPath string
	var graphJSONPath string
	var outputRoot string
	var format string
	var skipGenerate bool
	var skipGraph bool
	var parallelizeTargets bool
	var jobs int
	var xcodeArgs []string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Profile an Xcode build with timing summary and target graph analysis",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			report, err := profile.ProfileBuild(context.Background(), profile.BuildProfileOptions{
				ProjectRoot:        filepath.Dir(selectedConfigPath),
				Config:             cfg,
				Workspace:          workspace,
				Scheme:             scheme,
				Configuration:      configuration,
				Destination:        destination,
				DerivedDataPath:    derivedDataPath,
				ResultBundlePath:   resultBundlePath,
				LogPath:            logPath,
				GraphJSONPath:      graphJSONPath,
				OutputRoot:         outputRoot,
				SkipGenerate:       skipGenerate,
				SkipGraph:          skipGraph,
				ParallelizeTargets: parallelizeTargets,
				Jobs:               jobs,
				ExtraXcodeArgs:     xcodeArgs,
			})
			if err != nil {
				return err
			}

			return writeProfileReport(cmd, format, report, formatBuildProfileReport)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", configPath, "Path to project config JSON file")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace to build (default: <app_name>.xcworkspace)")
	cmd.Flags().StringVar(&scheme, "scheme", "", "Scheme to build (default: product_name or app_name)")
	cmd.Flags().StringVar(&configuration, "configuration", "Debug", "Build configuration")
	cmd.Flags().StringVar(&destination, "destination", "", "xcodebuild destination")
	cmd.Flags().StringVar(&derivedDataPath, "derived-data-path", "", "DerivedData path")
	cmd.Flags().StringVar(&resultBundlePath, "result-bundle-path", "", "Result bundle path")
	cmd.Flags().StringVar(&logPath, "log", "", "Analyze an existing xcodebuild log instead of running a build")
	cmd.Flags().StringVar(&graphJSONPath, "graph-json", "", "Analyze an existing Tuist legacyJSON graph")
	cmd.Flags().StringVar(&outputRoot, "output-root", "", "Directory for generated profile artifacts")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json")
	cmd.Flags().BoolVar(&skipGenerate, "skip-generate", false, "Skip tuist generate before building")
	cmd.Flags().BoolVar(&skipGraph, "skip-graph", false, "Skip Tuist graph analysis")
	cmd.Flags().BoolVar(&parallelizeTargets, "parallelize-targets", true, "Pass -parallelizeTargets to xcodebuild")
	cmd.Flags().IntVar(&jobs, "jobs", 0, "Maximum concurrent xcodebuild jobs")
	cmd.Flags().StringArrayVar(&xcodeArgs, "xcodebuild-arg", nil, "Extra argument passed to xcodebuild; repeat for multiple arguments")

	return cmd
}

func newProfileRuntimeCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runtime",
		Short: "Scaffold and analyze runtime profiling instrumentation",
	}
	cmd.AddCommand(
		newProfileRuntimeScaffoldCommand(opts),
		newProfileRuntimeAnalyzeCommand(),
		newProfileRuntimeErrorsCommand(),
	)
	return cmd
}

func newProfileLayoutCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "layout",
		Short: "Scaffold and analyze rendered UI hierarchy XML",
	}
	cmd.AddCommand(
		newProfileLayoutScaffoldCommand(opts),
		newProfileLayoutAnalyzeCommand(),
	)
	return cmd
}

func newProfileLayoutScaffoldCommand(opts *RootOptions) *cobra.Command {
	var configPath string
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		configPath = strings.TrimSpace(opts.ConfigPath)
	} else {
		configPath = config.DefaultConfigPath
	}

	var outputPath string
	var force bool
	var format string

	cmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Generate XCTest helpers that dump rendered UI hierarchy XML",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			result, err := profile.ScaffoldLayoutProbe(profile.LayoutScaffoldOptions{
				ProjectRoot: filepath.Dir(selectedConfigPath),
				Config:      cfg,
				OutputPath:  outputPath,
				Force:       force,
			})
			if err != nil {
				return err
			}

			return writeProfileReport(cmd, format, result, func(value profile.LayoutScaffoldResult) string {
				return fmt.Sprintf("layout hierarchy helper written to %s\n", value.Path)
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", configPath, "Path to project config JSON file")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output Swift file path")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing layout probe")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json")
	return cmd
}

func newProfileLayoutAnalyzeCommand() *cobra.Command {
	var inputPath string
	var format string
	var minTapSize float64
	var maxElements int
	var includeHidden bool

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze rendered UI hierarchy XML for agent-readable layout diagnostics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(inputPath) == "" {
				return fmt.Errorf("--input is required")
			}
			report, err := profile.AnalyzeLayoutXMLFile(inputPath, profile.LayoutAnalyzeOptions{
				MinTapSize:    minTapSize,
				MaxElements:   maxElements,
				IncludeHidden: includeHidden,
			})
			if err != nil {
				return err
			}

			return writeProfileReport(cmd, format, report, formatLayoutReport)
		},
	}

	cmd.Flags().StringVar(&inputPath, "input", "", "Path to rendered layout XML or log containing IAM_LAYOUT_XML markers")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json")
	cmd.Flags().Float64Var(&minTapSize, "min-tap-size", 44, "Minimum interactive element width/height before warning")
	cmd.Flags().IntVar(&maxElements, "max-elements", 200, "Maximum hierarchy elements included in the report")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "Include explicitly hidden elements in issue detection")
	return cmd
}

func newProfileRuntimeScaffoldCommand(opts *RootOptions) *cobra.Command {
	var configPath string
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		configPath = strings.TrimSpace(opts.ConfigPath)
	} else {
		configPath = config.DefaultConfigPath
	}

	var outputPath string
	var force bool
	var format string

	cmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Generate debug-only Swift runtime profiling helpers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			result, err := profile.ScaffoldRuntimeProbe(profile.RuntimeScaffoldOptions{
				ProjectRoot: filepath.Dir(selectedConfigPath),
				Config:      cfg,
				OutputPath:  outputPath,
				Force:       force,
			})
			if err != nil {
				return err
			}

			return writeProfileReport(cmd, format, result, func(value profile.RuntimeScaffoldResult) string {
				return fmt.Sprintf("runtime profile helper written to %s\n", value.Path)
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", configPath, "Path to project config JSON file")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output Swift file path")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing runtime probe")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json")
	return cmd
}

func newProfileRuntimeAnalyzeCommand() *cobra.Command {
	var inputPath string
	var format string
	var slowMS int
	var repeatThreshold int

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze IAM_PROFILE runtime log lines",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(inputPath) == "" {
				return fmt.Errorf("--input is required")
			}
			data, err := os.ReadFile(inputPath)
			if err != nil {
				return fmt.Errorf("read runtime profile log %q: %w", inputPath, err)
			}

			report := profile.AnalyzeRuntimeProfileLog(string(data), profile.RuntimeAnalyzeOptions{
				SlowThresholdMS: slowMS,
				RepeatThreshold: repeatThreshold,
			})
			return writeProfileReport(cmd, format, report, formatRuntimeProfileReport)
		},
	}

	cmd.Flags().StringVar(&inputPath, "input", "", "Path to a log file containing IAM_PROFILE JSON lines")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json")
	cmd.Flags().IntVar(&slowMS, "slow-ms", 16, "Main-thread duration threshold in milliseconds")
	cmd.Flags().IntVar(&repeatThreshold, "repeat-threshold", 50, "Call count threshold for repeated-call warnings")
	return cmd
}

func newProfileRuntimeErrorsCommand() *cobra.Command {
	var inputPath string
	var format string
	var last string
	var predicate string
	var processName string
	var subsystem string
	var category string
	var simulator bool
	var device string
	var includeDefault bool
	var maxExamples int

	cmd := &cobra.Command{
		Use:   "errors",
		Short: "Analyze runtime errors, faults, crash hints, and IAM_ERROR events",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := profile.RuntimeErrorAnalyzeOptions{
				IncludeDefault: includeDefault,
				MaxExamples:    maxExamples,
			}

			var report profile.RuntimeErrorReport
			var err error
			if strings.TrimSpace(inputPath) != "" {
				report, err = profile.AnalyzeRuntimeErrorsFile(inputPath, opts)
			} else {
				raw, collectErr := profile.CollectRuntimeErrors(cmd.Context(), profile.RuntimeErrorCollectOptions{
					Last:      last,
					Predicate: predicate,
					Process:   processName,
					Subsystem: subsystem,
					Category:  category,
					Simulator: simulator,
					Device:    device,
				})
				if collectErr != nil {
					return collectErr
				}
				report = profile.AnalyzeRuntimeErrors(string(raw), opts)
			}
			if err != nil {
				return err
			}

			return writeProfileReport(cmd, format, report, formatRuntimeErrorReport)
		},
	}

	cmd.Flags().StringVar(&inputPath, "input", "", "Path to a log file containing unified log output or IAM_ERROR lines")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, json")
	cmd.Flags().StringVar(&last, "last", "10m", "Time window for log collection when --input is omitted")
	cmd.Flags().StringVar(&predicate, "predicate", "", "Custom log predicate")
	cmd.Flags().StringVar(&processName, "process", "", "Filter collected logs by process")
	cmd.Flags().StringVar(&subsystem, "subsystem", "", "Filter collected logs by subsystem")
	cmd.Flags().StringVar(&category, "category", "", "Filter collected logs by category")
	cmd.Flags().BoolVar(&simulator, "simulator", false, "Collect logs from a simulator via simctl")
	cmd.Flags().StringVar(&device, "device", "", "Simulator device for --simulator (default: booted)")
	cmd.Flags().BoolVar(&includeDefault, "include-default", false, "Include non-error unified log entries from NDJSON input")
	cmd.Flags().IntVar(&maxExamples, "max-examples", 3, "Maximum example messages per group")
	return cmd
}

func writeProfileReport[T any](cmd *cobra.Command, format string, value T, textRenderer func(T) string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		_, err := fmt.Fprint(cmd.OutOrStdout(), textRenderer(value))
		return err
	case "json":
		data, err := profile.MarshalJSONStable(value)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", data)
		return err
	default:
		return fmt.Errorf("unsupported format %q (supported: text, json)", format)
	}
}

func formatBuildProfileReport(report profile.BuildProfileReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "build profile:\n")
	fmt.Fprintf(&b, "  scheme: %s\n", report.Scheme)
	fmt.Fprintf(&b, "  configuration: %s\n", report.Configuration)
	fmt.Fprintf(&b, "  destination: %s\n", report.Destination)
	fmt.Fprintf(&b, "  timing entries: %d\n", len(report.TimingEntries))
	fmt.Fprintf(&b, "  total target work: %.2fs\n", report.TotalWorkSeconds)
	if report.IdealParallelism > 0 {
		fmt.Fprintf(&b, "  ideal parallelism ceiling: %.2fx\n", report.IdealParallelism)
	}

	if len(report.Artifacts) > 0 {
		fmt.Fprintf(&b, "artifacts:\n")
		for _, key := range []string{"log", "derived_data", "result_bundle"} {
			if value := report.Artifacts[key]; value != "" {
				fmt.Fprintf(&b, "  %s: %s\n", key, value)
			}
		}
	}

	if len(report.TopCommands) > 0 {
		fmt.Fprintf(&b, "top commands:\n")
		for _, entry := range report.TopCommands {
			target := entry.Target
			if target == "" {
				target = "unknown"
			}
			fmt.Fprintf(&b, "  %.2fs  %-18s %s\n", entry.Duration, target, entry.Raw)
		}
	}

	if len(report.TargetWork) > 0 {
		fmt.Fprintf(&b, "target work:\n")
		for _, item := range report.TargetWork {
			fmt.Fprintf(&b, "  %.2fs  %-18s %d command(s)\n", item.Duration, item.Target, item.Commands)
		}
	}

	if len(report.CriticalPath) > 0 {
		fmt.Fprintf(&b, "critical path estimate:\n")
		for _, node := range report.CriticalPath {
			fmt.Fprintf(&b, "  %.2fs  %s\n", node.Duration, node.Target)
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintf(&b, "warnings:\n")
		for _, warning := range report.Warnings {
			fmt.Fprintf(&b, "  - %s\n", warning)
		}
	}

	return b.String()
}

func formatRuntimeProfileReport(report profile.RuntimeProfileReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "runtime profile:\n")
	fmt.Fprintf(&b, "  events: %d\n", report.EventCount)
	fmt.Fprintf(&b, "  slow threshold: %dms\n", report.SlowThresholdMS)
	fmt.Fprintf(&b, "  repeat threshold: %d\n", report.RepeatThreshold)
	if report.Startup != nil {
		fmt.Fprintf(
			&b,
			"  app startup to first render: %.2fms (%s)\n",
			report.Startup.DurationMS,
			report.Startup.FirstRenderName,
		)
	}

	if len(report.Groups) > 0 {
		fmt.Fprintf(&b, "hot groups:\n")
		for _, group := range report.Groups {
			fmt.Fprintf(
				&b,
				"  %-14s %-28s count=%d total=%.2fms avg=%.2fms max=%.2fms main=%d slow=%d\n",
				group.Kind,
				group.Name,
				group.Count,
				group.TotalDurationMS,
				group.AverageMS,
				group.MaxMS,
				group.MainThreadCount,
				group.SlowCount,
			)
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintf(&b, "warnings:\n")
		for _, warning := range report.Warnings {
			fmt.Fprintf(&b, "  - %s/%s: %s\n", warning.Kind, warning.Name, warning.Message)
		}
	}

	if len(report.ParseErrors) > 0 {
		fmt.Fprintf(&b, "parse errors:\n")
		for _, parseError := range report.ParseErrors {
			fmt.Fprintf(&b, "  - %s\n", parseError)
		}
	}

	return b.String()
}

func formatLayoutReport(report profile.LayoutReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "layout hierarchy:\n")
	fmt.Fprintf(&b, "  elements: %d\n", report.ElementCount)
	if report.Screen.Width > 0 || report.Screen.Height > 0 {
		fmt.Fprintf(&b, "  screen: %.2fx%.2f\n", report.Screen.Width, report.Screen.Height)
	}
	fmt.Fprintf(&b, "  max depth: %d\n", report.MaxDepth)
	if report.ReportedElementCount < report.ElementCount {
		fmt.Fprintf(&b, "  displayed: %d\n", report.ReportedElementCount)
	}

	if len(report.TypeCounts) > 0 {
		fmt.Fprintf(&b, "types:\n")
		for _, item := range report.TypeCounts {
			fmt.Fprintf(&b, "  %-18s %d\n", item.Type, item.Count)
		}
	}

	if len(report.Elements) > 0 {
		fmt.Fprintf(&b, "tree:\n")
		for _, element := range report.Elements {
			indent := strings.Repeat("  ", element.Depth+1)
			identity := firstNonEmptyLocal(element.Identifier, element.Name, element.Label, "-")
			frame := "-"
			if element.Frame != nil {
				frame = fmt.Sprintf("%.0f,%.0f %.0fx%.0f", element.Frame.X, element.Frame.Y, element.Frame.Width, element.Frame.Height)
			}
			state := layoutElementState(element)
			if state != "" {
				state = " " + state
			}
			fmt.Fprintf(&b, "%s%s identity=%q frame=%s path=%s%s\n", indent, element.Type, identity, frame, element.Path, state)
		}
	}

	if len(report.DuplicateIdentities) > 0 {
		fmt.Fprintf(&b, "duplicate identities:\n")
		for _, duplicate := range report.DuplicateIdentities {
			fmt.Fprintf(&b, "  %q count=%d paths=%s\n", duplicate.Identity, duplicate.Count, strings.Join(duplicate.Paths, ", "))
		}
	}

	if len(report.Issues) > 0 {
		fmt.Fprintf(&b, "issues:\n")
		for _, issue := range report.Issues {
			identity := firstNonEmptyLocal(issue.Identifier, issue.Label, "-")
			fmt.Fprintf(&b, "  - %s/%s %s identity=%q path=%s\n", issue.Kind, issue.Severity, issue.Message, identity, issue.Path)
		}
	}

	if len(report.ParseErrors) > 0 {
		fmt.Fprintf(&b, "parse errors:\n")
		for _, parseError := range report.ParseErrors {
			fmt.Fprintf(&b, "  - %s\n", parseError)
		}
	}

	return b.String()
}

func layoutElementState(element profile.LayoutElement) string {
	parts := make([]string, 0, 4)
	if element.Enabled != nil && !*element.Enabled {
		parts = append(parts, "disabled")
	}
	if element.Hittable != nil && *element.Hittable {
		parts = append(parts, "hittable")
	}
	if element.Visible != nil && !*element.Visible {
		parts = append(parts, "hidden")
	}
	if element.Accessible != nil && *element.Accessible {
		parts = append(parts, "accessible")
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func firstNonEmptyLocal(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func formatRuntimeErrorReport(report profile.RuntimeErrorReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "runtime errors:\n")
	fmt.Fprintf(&b, "  events: %d\n", report.EventCount)

	if len(report.Groups) > 0 {
		fmt.Fprintf(&b, "groups:\n")
		for _, group := range report.Groups {
			scope := group.Process
			if group.Subsystem != "" {
				if scope != "" {
					scope += " "
				}
				scope += group.Subsystem
			}
			if group.Category != "" {
				if scope != "" {
					scope += "/"
				}
				scope += group.Category
			}
			if scope == "" {
				scope = "-"
			}
			fmt.Fprintf(
				&b,
				"  %-5s count=%d scope=%s signature=%s\n",
				group.Severity,
				group.Count,
				scope,
				group.Signature,
			)
			for _, example := range group.Examples {
				fmt.Fprintf(&b, "    example: %s\n", example)
			}
		}
	}

	if len(report.Hints) > 0 {
		fmt.Fprintf(&b, "hints:\n")
		for _, hint := range report.Hints {
			fmt.Fprintf(&b, "  - %s/%s: %s\n", hint.Kind, hint.Severity, hint.Message)
		}
	}

	if len(report.ParseErrors) > 0 {
		fmt.Fprintf(&b, "parse errors:\n")
		for _, parseError := range report.ParseErrors {
			fmt.Fprintf(&b, "  - %s\n", parseError)
		}
	}

	return b.String()
}
