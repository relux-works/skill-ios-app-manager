package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const runtimeProfileMarker = "IAM_PROFILE"

// RuntimeAnalyzeOptions configures runtime log analysis.
type RuntimeAnalyzeOptions struct {
	SlowThresholdMS int `json:"slow_threshold_ms"`
	RepeatThreshold int `json:"repeat_threshold"`
}

// RuntimeEvent is one structured runtime profiling event.
type RuntimeEvent struct {
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	DurationMS float64 `json:"duration_ms,omitempty"`
	Thread     string  `json:"thread,omitempty"`
	Timestamp  float64 `json:"timestamp,omitempty"`
	File       string  `json:"file,omitempty"`
	Line       int     `json:"line,omitempty"`
}

// RuntimeProfileReport summarizes runtime profiling events.
type RuntimeProfileReport struct {
	EventCount      int              `json:"event_count"`
	SlowThresholdMS int              `json:"slow_threshold_ms"`
	RepeatThreshold int              `json:"repeat_threshold"`
	Startup         *StartupTiming   `json:"startup,omitempty"`
	Groups          []RuntimeGroup   `json:"groups"`
	Warnings        []RuntimeWarning `json:"warnings,omitempty"`
	ParseErrors     []string         `json:"parse_errors,omitempty"`
}

// StartupTiming captures app start to first render latency.
type StartupTiming struct {
	AppStartTimestamp    float64 `json:"app_start_timestamp"`
	FirstRenderTimestamp float64 `json:"first_render_timestamp"`
	FirstRenderName      string  `json:"first_render_name,omitempty"`
	DurationMS           float64 `json:"duration_ms"`
}

// RuntimeGroup aggregates events by kind and name.
type RuntimeGroup struct {
	Kind            string  `json:"kind"`
	Name            string  `json:"name"`
	Count           int     `json:"count"`
	TotalDurationMS float64 `json:"total_duration_ms"`
	AverageMS       float64 `json:"average_ms"`
	MaxMS           float64 `json:"max_ms"`
	MainThreadCount int     `json:"main_thread_count"`
	SlowCount       int     `json:"slow_count"`
}

// RuntimeWarning highlights a suspicious runtime pattern.
type RuntimeWarning struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// ParseRuntimeProfileLog extracts IAM_PROFILE JSON lines.
func ParseRuntimeProfileLog(raw string) ([]RuntimeEvent, []string) {
	events := make([]RuntimeEvent, 0)
	parseErrors := make([]string, 0)

	for lineNumber, line := range strings.Split(raw, "\n") {
		index := strings.Index(line, runtimeProfileMarker)
		if index < 0 {
			continue
		}
		payload := strings.TrimSpace(line[index+len(runtimeProfileMarker):])
		if payload == "" {
			continue
		}
		if strings.HasPrefix(payload, ":") {
			payload = strings.TrimSpace(strings.TrimPrefix(payload, ":"))
		}

		var event RuntimeEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("line %d: %v", lineNumber+1, err))
			continue
		}
		if strings.TrimSpace(event.Kind) == "" {
			event.Kind = "event"
		}
		if strings.TrimSpace(event.Name) == "" {
			parseErrors = append(parseErrors, fmt.Sprintf("line %d: missing event name", lineNumber+1))
			continue
		}
		events = append(events, event)
	}

	return events, parseErrors
}

// AnalyzeRuntimeProfileLog builds a runtime profile report from raw logs.
func AnalyzeRuntimeProfileLog(raw string, opts RuntimeAnalyzeOptions) RuntimeProfileReport {
	if opts.SlowThresholdMS <= 0 {
		opts.SlowThresholdMS = 16
	}
	if opts.RepeatThreshold <= 0 {
		opts.RepeatThreshold = 50
	}

	events, parseErrors := ParseRuntimeProfileLog(raw)
	groups := aggregateRuntimeEvents(events, opts)
	report := RuntimeProfileReport{
		EventCount:      len(events),
		SlowThresholdMS: opts.SlowThresholdMS,
		RepeatThreshold: opts.RepeatThreshold,
		Startup:         detectStartupTiming(events),
		Groups:          groups,
		ParseErrors:     parseErrors,
	}

	for _, group := range groups {
		if group.Count >= opts.RepeatThreshold {
			report.Warnings = append(report.Warnings, RuntimeWarning{
				Kind:    group.Kind,
				Name:    group.Name,
				Message: fmt.Sprintf("called %d times", group.Count),
			})
		}
		if group.SlowCount > 0 {
			report.Warnings = append(report.Warnings, RuntimeWarning{
				Kind:    group.Kind,
				Name:    group.Name,
				Message: fmt.Sprintf("%d main-thread event(s) at or above %dms", group.SlowCount, opts.SlowThresholdMS),
			})
		}
	}

	return report
}

func detectStartupTiming(events []RuntimeEvent) *StartupTiming {
	appStart := 0.0
	for _, event := range events {
		if event.Kind == "app_start" && event.Timestamp > 0 {
			appStart = event.Timestamp
			break
		}
	}
	if appStart <= 0 {
		return nil
	}

	for _, event := range events {
		if event.Kind != "first_render" || event.Timestamp <= 0 || event.Timestamp < appStart {
			continue
		}
		return &StartupTiming{
			AppStartTimestamp:    appStart,
			FirstRenderTimestamp: event.Timestamp,
			FirstRenderName:      event.Name,
			DurationMS:           (event.Timestamp - appStart) * 1000,
		}
	}

	return nil
}

func aggregateRuntimeEvents(events []RuntimeEvent, opts RuntimeAnalyzeOptions) []RuntimeGroup {
	groups := make(map[string]*RuntimeGroup)
	for _, event := range events {
		key := event.Kind + "\x00" + event.Name
		group := groups[key]
		if group == nil {
			group = &RuntimeGroup{Kind: event.Kind, Name: event.Name}
			groups[key] = group
		}
		group.Count++
		group.TotalDurationMS += event.DurationMS
		if event.DurationMS > group.MaxMS {
			group.MaxMS = event.DurationMS
		}
		if strings.EqualFold(event.Thread, "main") {
			group.MainThreadCount++
			if event.DurationMS >= float64(opts.SlowThresholdMS) {
				group.SlowCount++
			}
		}
	}

	out := make([]RuntimeGroup, 0, len(groups))
	for _, group := range groups {
		if group.Count > 0 {
			group.AverageMS = group.TotalDurationMS / float64(group.Count)
		}
		out = append(out, *group)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalDurationMS == out[j].TotalDurationMS {
			if out[i].Count == out[j].Count {
				return out[i].Name < out[j].Name
			}
			return out[i].Count > out[j].Count
		}
		return out[i].TotalDurationMS > out[j].TotalDurationMS
	})
	return out
}

// RuntimeScaffoldOptions configures PerformanceProbe generation.
type RuntimeScaffoldOptions struct {
	ProjectRoot string
	Config      config.ProjectConfig
	OutputPath  string
	Force       bool
}

// RuntimeScaffoldResult reports generated runtime probe location.
type RuntimeScaffoldResult struct {
	Path string `json:"path"`
}

// ScaffoldRuntimeProbe writes the debug-only Swift runtime profiling helper.
func ScaffoldRuntimeProbe(opts RuntimeScaffoldOptions) (RuntimeScaffoldResult, error) {
	root := strings.TrimSpace(opts.ProjectRoot)
	if root == "" {
		root = "."
	}
	appName := strings.TrimSpace(opts.Config.AppName)
	if appName == "" {
		return RuntimeScaffoldResult{}, fmt.Errorf("app_name is required to choose default runtime probe path")
	}

	outputPath := strings.TrimSpace(opts.OutputPath)
	if outputPath == "" {
		outputPath = filepath.Join("Targets", appName, "Sources", "Diagnostics", "PerformanceProbe.swift")
	}
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(root, outputPath)
	}
	outputPath = filepath.Clean(outputPath)

	if !opts.Force {
		if _, err := os.Stat(outputPath); err == nil {
			return RuntimeScaffoldResult{}, fmt.Errorf("runtime probe already exists at %q; pass --force to overwrite", outputPath)
		} else if !os.IsNotExist(err) {
			return RuntimeScaffoldResult{}, fmt.Errorf("stat runtime probe %q: %w", outputPath, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return RuntimeScaffoldResult{}, fmt.Errorf("create runtime probe directory: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(GeneratePerformanceProbeSwift()), 0o644); err != nil {
		return RuntimeScaffoldResult{}, fmt.Errorf("write runtime probe %q: %w", outputPath, err)
	}

	return RuntimeScaffoldResult{Path: outputPath}, nil
}

// GeneratePerformanceProbeSwift returns the Swift helper source.
func GeneratePerformanceProbeSwift() string {
	return performanceProbeSwift
}
