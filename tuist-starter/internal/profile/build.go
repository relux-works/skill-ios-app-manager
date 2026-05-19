package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const (
	defaultBuildConfiguration = "Debug"
	defaultBuildDestination   = "generic/platform=iOS Simulator"
	defaultBuildAction        = "build"
)

// CommandRunner executes external commands for profile collection.
type CommandRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
}

// ExecRunner runs commands through os/exec.
type ExecRunner struct{}

// Run executes one command and returns combined stdout/stderr.
func (ExecRunner) Run(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// BuildProfileOptions configures build profiling.
type BuildProfileOptions struct {
	ProjectRoot        string
	Config             config.ProjectConfig
	Workspace          string
	Scheme             string
	Configuration      string
	Destination        string
	DerivedDataPath    string
	ResultBundlePath   string
	LogPath            string
	GraphJSONPath      string
	OutputRoot         string
	SkipGenerate       bool
	SkipGraph          bool
	ParallelizeTargets bool
	Jobs               int
	ExtraXcodeArgs     []string
	Runner             CommandRunner
}

// BuildProfileReport is the machine-readable build profiling result.
type BuildProfileReport struct {
	ProjectRoot      string             `json:"project_root"`
	Workspace        string             `json:"workspace"`
	Scheme           string             `json:"scheme"`
	Configuration    string             `json:"configuration"`
	Destination      string             `json:"destination"`
	DerivedDataPath  string             `json:"derived_data_path,omitempty"`
	ResultBundlePath string             `json:"result_bundle_path,omitempty"`
	LogPath          string             `json:"log_path,omitempty"`
	ElapsedSeconds   float64            `json:"elapsed_seconds,omitempty"`
	TotalWorkSeconds float64            `json:"total_work_seconds"`
	IdealParallelism float64            `json:"ideal_parallelism,omitempty"`
	TimingEntries    []TimingEntry      `json:"timing_entries"`
	TopCommands      []TimingEntry      `json:"top_commands"`
	TargetWork       []TargetWork       `json:"target_work"`
	CriticalPath     []CriticalPathNode `json:"critical_path,omitempty"`
	GraphNodeCount   int                `json:"graph_node_count,omitempty"`
	GraphEdgeCount   int                `json:"graph_edge_count,omitempty"`
	Warnings         []string           `json:"warnings,omitempty"`
	Artifacts        map[string]string  `json:"artifacts,omitempty"`
}

// TimingEntry describes one xcodebuild timing summary entry.
type TimingEntry struct {
	Command  string  `json:"command"`
	Target   string  `json:"target,omitempty"`
	Project  string  `json:"project,omitempty"`
	Duration float64 `json:"duration_seconds"`
	Raw      string  `json:"raw"`
}

// TargetWork is aggregate timing by target.
type TargetWork struct {
	Target   string  `json:"target"`
	Duration float64 `json:"duration_seconds"`
	Commands int     `json:"commands"`
}

// CriticalPathNode is one node in the estimated target critical path.
type CriticalPathNode struct {
	Target   string  `json:"target"`
	Duration float64 `json:"duration_seconds"`
}

// ProfileBuild collects and analyzes a build profile.
func ProfileBuild(ctx context.Context, opts BuildProfileOptions) (BuildProfileReport, error) {
	normalized, err := normalizeBuildOptions(opts)
	if err != nil {
		return BuildProfileReport{}, err
	}

	runner := normalized.Runner
	if runner == nil {
		runner = ExecRunner{}
	}

	var graph *TargetGraph
	var warnings []string
	if !normalized.SkipGraph {
		graph, err = loadBuildGraph(ctx, normalized, runner)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
	}

	var logData []byte
	start := time.Now()
	if normalized.LogPath != "" {
		logData, err = os.ReadFile(normalized.LogPath)
		if err != nil {
			return BuildProfileReport{}, fmt.Errorf("read build log %q: %w", normalized.LogPath, err)
		}
	} else {
		if !normalized.SkipGenerate {
			if output, err := runner.Run(ctx, normalized.ProjectRoot, "tuist", "generate", "--no-open", "--path", normalized.ProjectRoot); err != nil {
				return BuildProfileReport{}, fmt.Errorf("tuist generate failed: %w\noutput: %s", err, string(output))
			}
		}

		args := normalized.xcodebuildArgs()
		logData, err = runner.Run(ctx, normalized.ProjectRoot, "xcodebuild", args...)
		if err != nil {
			return BuildProfileReport{}, fmt.Errorf("xcodebuild failed: %w\noutput: %s", err, string(logData))
		}

		if normalized.OutputRoot != "" {
			if err := os.MkdirAll(normalized.OutputRoot, 0o755); err != nil {
				return BuildProfileReport{}, fmt.Errorf("create output root %q: %w", normalized.OutputRoot, err)
			}
			normalized.LogPath = filepath.Join(normalized.OutputRoot, "xcodebuild.log")
			if err := os.WriteFile(normalized.LogPath, logData, 0o644); err != nil {
				return BuildProfileReport{}, fmt.Errorf("write build log %q: %w", normalized.LogPath, err)
			}
		}
	}

	report := AnalyzeBuildLog(string(logData), graph)
	report.ProjectRoot = normalized.ProjectRoot
	report.Workspace = normalized.Workspace
	report.Scheme = normalized.Scheme
	report.Configuration = normalized.Configuration
	report.Destination = normalized.Destination
	report.DerivedDataPath = normalized.DerivedDataPath
	report.ResultBundlePath = normalized.ResultBundlePath
	report.LogPath = normalized.LogPath
	report.ElapsedSeconds = time.Since(start).Seconds()
	report.Warnings = append(report.Warnings, warnings...)

	if len(report.TimingEntries) == 0 {
		report.Warnings = append(report.Warnings, "no xcodebuild timing summary entries found; make sure the build log includes -showBuildTimingSummary output")
	}

	report.Artifacts = make(map[string]string)
	for key, value := range map[string]string{
		"log":           report.LogPath,
		"derived_data":  report.DerivedDataPath,
		"result_bundle": report.ResultBundlePath,
	} {
		if strings.TrimSpace(value) != "" {
			report.Artifacts[key] = value
		}
	}
	if len(report.Artifacts) == 0 {
		report.Artifacts = nil
	}

	return report, nil
}

func normalizeBuildOptions(opts BuildProfileOptions) (BuildProfileOptions, error) {
	root := strings.TrimSpace(opts.ProjectRoot)
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return BuildProfileOptions{}, fmt.Errorf("resolve project root %q: %w", root, err)
	}
	opts.ProjectRoot = filepath.Clean(absRoot)

	if strings.TrimSpace(opts.Workspace) == "" {
		appName := strings.TrimSpace(opts.Config.AppName)
		if appName == "" {
			return BuildProfileOptions{}, fmt.Errorf("workspace is required when app_name is unavailable")
		}
		opts.Workspace = appName + ".xcworkspace"
	}
	if strings.TrimSpace(opts.Scheme) == "" {
		opts.Scheme = strings.TrimSpace(opts.Config.ProductName)
		if opts.Scheme == "" {
			opts.Scheme = strings.TrimSpace(opts.Config.AppName)
		}
	}
	if strings.TrimSpace(opts.Scheme) == "" {
		return BuildProfileOptions{}, fmt.Errorf("scheme is required")
	}
	if strings.TrimSpace(opts.Configuration) == "" {
		opts.Configuration = defaultBuildConfiguration
	}
	if strings.TrimSpace(opts.Destination) == "" {
		opts.Destination = defaultBuildDestination
	}

	if strings.TrimSpace(opts.OutputRoot) == "" && strings.TrimSpace(opts.LogPath) == "" {
		opts.OutputRoot = filepath.Join(opts.ProjectRoot, ".temp", "build-profile", time.Now().Format("20060102-150405"))
	}
	if strings.TrimSpace(opts.OutputRoot) != "" {
		outputRoot := opts.OutputRoot
		if !filepath.IsAbs(outputRoot) {
			outputRoot = filepath.Join(opts.ProjectRoot, outputRoot)
		}
		opts.OutputRoot = filepath.Clean(outputRoot)
	}

	if strings.TrimSpace(opts.DerivedDataPath) == "" && opts.OutputRoot != "" {
		opts.DerivedDataPath = filepath.Join(opts.OutputRoot, "DerivedData")
	}
	if strings.TrimSpace(opts.ResultBundlePath) == "" && opts.OutputRoot != "" {
		opts.ResultBundlePath = filepath.Join(opts.OutputRoot, "Build.xcresult")
	}

	return opts, nil
}

func (o BuildProfileOptions) xcodebuildArgs() []string {
	args := []string{
		"-workspace", o.Workspace,
		"-scheme", o.Scheme,
		"-configuration", o.Configuration,
		"-destination", o.Destination,
	}
	if o.DerivedDataPath != "" {
		args = append(args, "-derivedDataPath", o.DerivedDataPath)
	}
	if o.ResultBundlePath != "" {
		args = append(args, "-resultBundlePath", o.ResultBundlePath)
	}
	if o.ParallelizeTargets {
		args = append(args, "-parallelizeTargets")
	}
	if o.Jobs > 0 {
		args = append(args, "-jobs", strconv.Itoa(o.Jobs))
	}
	args = append(args, "-showBuildTimingSummary")
	args = append(args, o.ExtraXcodeArgs...)
	args = append(args, defaultBuildAction)
	return args
}

func loadBuildGraph(ctx context.Context, opts BuildProfileOptions, runner CommandRunner) (*TargetGraph, error) {
	var data []byte
	var err error
	if opts.GraphJSONPath != "" {
		data, err = os.ReadFile(opts.GraphJSONPath)
		if err != nil {
			return nil, fmt.Errorf("read graph JSON %q: %w", opts.GraphJSONPath, err)
		}
	} else {
		graphDir := ""
		if opts.OutputRoot != "" {
			graphDir = filepath.Join(opts.OutputRoot, "graph")
		} else {
			graphDir = filepath.Join(opts.ProjectRoot, ".temp", "build-profile-graph")
		}
		if err := os.MkdirAll(graphDir, 0o755); err != nil {
			return nil, fmt.Errorf("create graph output dir %q: %w", graphDir, err)
		}

		output, err := runner.Run(
			ctx,
			opts.ProjectRoot,
			"tuist",
			"graph",
			"--format", "legacyJSON",
			"--no-open",
			"--path", opts.ProjectRoot,
			"--output-path", graphDir,
		)
		if err != nil {
			return nil, fmt.Errorf("tuist graph failed: %w\noutput: %s", err, string(output))
		}

		graphPath := filepath.Join(graphDir, "graph.json")
		data, err = os.ReadFile(graphPath)
		if err != nil {
			return nil, fmt.Errorf("read tuist graph output %q: %w", graphPath, err)
		}
	}

	graph, err := ParseTuistTargetGraph(data, opts.ProjectRoot)
	if err != nil {
		return nil, err
	}
	return graph, nil
}

var (
	durationAtEndPattern = regexp.MustCompile(`(?i)(?:^|\s)([0-9]+(?:\.[0-9]+)?)\s*(seconds|second|sec|s)\s*$`)
	durationOnlyPattern  = regexp.MustCompile(`(?i)^([0-9]+(?:\.[0-9]+)?)\s*(seconds|second|sec|s)$`)
	targetPattern        = regexp.MustCompile(`(?i)in target '([^']+)'(?: from project '([^']+)')?`)
)

// ParseBuildTimingSummary extracts xcodebuild timing entries from raw output.
func ParseBuildTimingSummary(raw string) []TimingEntry {
	lines := strings.Split(raw, "\n")
	entries := make([]TimingEntry, 0)
	previous := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.EqualFold(trimmed, "Build Timing Summary") {
			previous = ""
			continue
		}

		if match := durationOnlyPattern.FindStringSubmatch(trimmed); match != nil && previous != "" {
			duration, _ := strconv.ParseFloat(match[1], 64)
			entries = append(entries, newTimingEntry(previous, duration))
			previous = ""
			continue
		}

		match := durationAtEndPattern.FindStringSubmatch(trimmed)
		if match == nil {
			previous = trimmed
			continue
		}

		duration, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			previous = trimmed
			continue
		}

		command := strings.TrimSpace(trimmed[:strings.LastIndex(trimmed, match[0])])
		if command == "" {
			command = previous
		}
		if command == "" {
			continue
		}
		entries = append(entries, newTimingEntry(command, duration))
		previous = ""
	}

	return entries
}

func newTimingEntry(command string, duration float64) TimingEntry {
	entry := TimingEntry{
		Command:  commandName(command),
		Duration: duration,
		Raw:      strings.TrimSpace(command),
	}
	if match := targetPattern.FindStringSubmatch(command); match != nil {
		entry.Target = match[1]
		if len(match) > 2 {
			entry.Project = match[2]
		}
	}
	return entry
}

func commandName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "unknown"
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "unknown"
	}
	return strings.Trim(fields[0], ":")
}

// AnalyzeBuildLog creates a report from raw xcodebuild output and an optional graph.
func AnalyzeBuildLog(raw string, graph *TargetGraph) BuildProfileReport {
	entries := ParseBuildTimingSummary(raw)
	report := BuildProfileReport{
		TimingEntries:    entries,
		TopCommands:      topTimingEntries(entries, 15),
		TargetWork:       aggregateTargetWork(entries),
		TotalWorkSeconds: totalEntryDuration(entries),
	}

	if graph != nil {
		report.GraphNodeCount = len(graph.Nodes)
		report.GraphEdgeCount = graph.EdgeCount()
		weights := targetWorkWeights(report.TargetWork)
		report.CriticalPath = graph.CriticalPath(weights)
		criticalDuration := criticalPathDuration(report.CriticalPath)
		if criticalDuration > 0 {
			report.IdealParallelism = report.TotalWorkSeconds / criticalDuration
		}
	}

	return report
}

func topTimingEntries(entries []TimingEntry, limit int) []TimingEntry {
	out := append([]TimingEntry(nil), entries...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Duration > out[j].Duration
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func aggregateTargetWork(entries []TimingEntry) []TargetWork {
	type aggregate struct {
		duration float64
		commands int
	}
	byTarget := make(map[string]aggregate)
	for _, entry := range entries {
		target := strings.TrimSpace(entry.Target)
		if target == "" {
			target = "unknown"
		}
		value := byTarget[target]
		value.duration += entry.Duration
		value.commands++
		byTarget[target] = value
	}

	out := make([]TargetWork, 0, len(byTarget))
	for target, value := range byTarget {
		out = append(out, TargetWork{
			Target:   target,
			Duration: value.duration,
			Commands: value.commands,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Duration == out[j].Duration {
			return out[i].Target < out[j].Target
		}
		return out[i].Duration > out[j].Duration
	})
	return out
}

func totalEntryDuration(entries []TimingEntry) float64 {
	total := 0.0
	for _, entry := range entries {
		total += entry.Duration
	}
	return total
}

func targetWorkWeights(work []TargetWork) map[string]float64 {
	weights := make(map[string]float64, len(work))
	for _, item := range work {
		weights[item.Target] = item.Duration
	}
	return weights
}

func criticalPathDuration(path []CriticalPathNode) float64 {
	total := 0.0
	for _, node := range path {
		total += node.Duration
	}
	return total
}

// MarshalJSONStable returns indented JSON for reports.
func MarshalJSONStable(value any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func seconds(value float64) float64 {
	return math.Round(value*1000) / 1000
}
