package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const runtimeErrorMarker = "IAM_ERROR"

var (
	digitsPattern        = regexp.MustCompile(`\b\d+\b`)
	hexAddressPattern    = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	whitespacePattern    = regexp.MustCompile(`\s+`)
	runtimeErrorKeywords = []string{
		"fatal error",
		"uncaught exception",
		"exception",
		"crash",
		"sigabrt",
		"sigsegv",
		"exc_bad",
		"hang",
		"main thread checker",
		"thread performance checker",
		"publishing changes from background threads",
		"index out of range",
		"could not cast value",
	}
)

// RuntimeErrorCollectOptions configures runtime error log collection.
type RuntimeErrorCollectOptions struct {
	Last      string
	Predicate string
	Process   string
	Subsystem string
	Category  string
	Simulator bool
	Device    string
	Runner    CommandRunner
}

// RuntimeErrorAnalyzeOptions configures runtime error analysis.
type RuntimeErrorAnalyzeOptions struct {
	IncludeDefault bool
	MaxExamples    int
}

// RuntimeErrorEvent is one parsed runtime error signal.
type RuntimeErrorEvent struct {
	Source    string `json:"source"`
	Severity  string `json:"severity"`
	Timestamp string `json:"timestamp,omitempty"`
	Process   string `json:"process,omitempty"`
	Subsystem string `json:"subsystem,omitempty"`
	Category  string `json:"category,omitempty"`
	Name      string `json:"name,omitempty"`
	Message   string `json:"message"`
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Raw       string `json:"raw,omitempty"`
}

// RuntimeErrorReport summarizes runtime errors.
type RuntimeErrorReport struct {
	EventCount  int                 `json:"event_count"`
	Groups      []RuntimeErrorGroup `json:"groups"`
	Hints       []RuntimeErrorHint  `json:"hints,omitempty"`
	ParseErrors []string            `json:"parse_errors,omitempty"`
}

// RuntimeErrorGroup groups similar runtime errors.
type RuntimeErrorGroup struct {
	Severity  string   `json:"severity"`
	Process   string   `json:"process,omitempty"`
	Subsystem string   `json:"subsystem,omitempty"`
	Category  string   `json:"category,omitempty"`
	Signature string   `json:"signature"`
	Count     int      `json:"count"`
	Examples  []string `json:"examples,omitempty"`
}

// RuntimeErrorHint highlights crash/hang/exception-like messages.
type RuntimeErrorHint struct {
	Kind      string `json:"kind"`
	Severity  string `json:"severity"`
	Process   string `json:"process,omitempty"`
	Subsystem string `json:"subsystem,omitempty"`
	Message   string `json:"message"`
}

// CollectRuntimeErrors runs the platform log command and returns raw output.
func CollectRuntimeErrors(ctx context.Context, opts RuntimeErrorCollectOptions) ([]byte, error) {
	runner := opts.Runner
	if runner == nil {
		runner = ExecRunner{}
	}

	last := strings.TrimSpace(opts.Last)
	if last == "" {
		last = "10m"
	}
	predicate := runtimeErrorPredicate(opts)

	args := []string{"show", "--style", "ndjson", "--last", last, "--predicate", predicate}
	if opts.Simulator {
		device := strings.TrimSpace(opts.Device)
		if device == "" {
			device = "booted"
		}
		simctlArgs := append([]string{"simctl", "spawn", device, "log"}, args...)
		return runner.Run(ctx, ".", "xcrun", simctlArgs...)
	}

	return runner.Run(ctx, ".", "log", args...)
}

func runtimeErrorPredicate(opts RuntimeErrorCollectOptions) string {
	predicate := strings.TrimSpace(opts.Predicate)
	if predicate == "" {
		predicate = `(logType == "error" OR logType == "fault")`
	}
	if process := strings.TrimSpace(opts.Process); process != "" {
		predicate = fmt.Sprintf(`(%s) AND process == "%s"`, predicate, escapePredicateString(process))
	}
	if subsystem := strings.TrimSpace(opts.Subsystem); subsystem != "" {
		predicate = fmt.Sprintf(`(%s) AND subsystem == "%s"`, predicate, escapePredicateString(subsystem))
	}
	if category := strings.TrimSpace(opts.Category); category != "" {
		predicate = fmt.Sprintf(`(%s) AND category == "%s"`, predicate, escapePredicateString(category))
	}
	return predicate
}

func escapePredicateString(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

// AnalyzeRuntimeErrorsFile reads and analyzes runtime error logs.
func AnalyzeRuntimeErrorsFile(path string, opts RuntimeErrorAnalyzeOptions) (RuntimeErrorReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RuntimeErrorReport{}, fmt.Errorf("read runtime error log %q: %w", path, err)
	}
	return AnalyzeRuntimeErrors(string(data), opts), nil
}

// AnalyzeRuntimeErrors parses runtime error logs from multiple sources.
func AnalyzeRuntimeErrors(raw string, opts RuntimeErrorAnalyzeOptions) RuntimeErrorReport {
	if opts.MaxExamples <= 0 {
		opts.MaxExamples = 3
	}

	events, parseErrors := ParseRuntimeErrorEvents(raw, opts)
	return BuildRuntimeErrorReport(events, parseErrors, opts)
}

// ParseRuntimeErrorEvents extracts error events from structured and plain logs.
func ParseRuntimeErrorEvents(raw string, opts RuntimeErrorAnalyzeOptions) ([]RuntimeErrorEvent, []string) {
	events := make([]RuntimeErrorEvent, 0)
	parseErrors := make([]string, 0)

	for lineNumber, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if event, ok, err := parseRuntimeErrorMarkerLine(trimmed); ok {
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("line %d: %v", lineNumber+1, err))
				continue
			}
			events = append(events, event)
			continue
		}

		if event, ok, err := parseUnifiedLogNDJSONLine(trimmed, opts); ok {
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("line %d: %v", lineNumber+1, err))
				continue
			}
			events = append(events, event)
			continue
		}

		if event, ok := parsePlainRuntimeErrorLine(trimmed); ok {
			events = append(events, event)
		}
	}

	return events, parseErrors
}

func parseRuntimeErrorMarkerLine(line string) (RuntimeErrorEvent, bool, error) {
	index := strings.Index(line, runtimeErrorMarker)
	if index < 0 {
		return RuntimeErrorEvent{}, false, nil
	}
	payload := strings.TrimSpace(line[index+len(runtimeErrorMarker):])
	if strings.HasPrefix(payload, ":") {
		payload = strings.TrimSpace(strings.TrimPrefix(payload, ":"))
	}
	if payload == "" {
		return RuntimeErrorEvent{}, true, fmt.Errorf("empty IAM_ERROR payload")
	}

	var event RuntimeErrorEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return RuntimeErrorEvent{}, true, err
	}
	event.Source = firstNonEmpty(event.Source, "iam_error")
	event.Severity = normalizeSeverity(firstNonEmpty(event.Severity, "error"))
	event.Message = firstNonEmpty(event.Message, event.Name)
	if strings.TrimSpace(event.Message) == "" {
		return RuntimeErrorEvent{}, true, fmt.Errorf("IAM_ERROR missing message/name")
	}
	return event, true, nil
}

func parseUnifiedLogNDJSONLine(line string, opts RuntimeErrorAnalyzeOptions) (RuntimeErrorEvent, bool, error) {
	if !strings.HasPrefix(line, "{") {
		return RuntimeErrorEvent{}, false, nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		return RuntimeErrorEvent{}, true, err
	}

	severity := normalizeSeverity(firstString(payload, "logType", "messageType", "level"))
	if severity == "" {
		severity = normalizeSeverity(firstString(payload, "type"))
	}
	if severity == "" {
		severity = "default"
	}
	if !opts.IncludeDefault && severity != "error" && severity != "fault" {
		return RuntimeErrorEvent{}, false, nil
	}

	message := firstString(payload, "composedMessage", "eventMessage", "message", "formatString")
	if strings.TrimSpace(message) == "" {
		return RuntimeErrorEvent{}, false, nil
	}

	return RuntimeErrorEvent{
		Source:    "unified_log",
		Severity:  severity,
		Timestamp: firstString(payload, "timestamp", "date"),
		Process:   firstString(payload, "process", "processImagePath"),
		Subsystem: firstString(payload, "subsystem"),
		Category:  firstString(payload, "category"),
		Message:   message,
		Raw:       line,
	}, true, nil
}

func parsePlainRuntimeErrorLine(line string) (RuntimeErrorEvent, bool) {
	lower := strings.ToLower(line)
	severity := ""
	switch {
	case strings.Contains(lower, " fault"):
		severity = "fault"
	case strings.Contains(lower, " fatal"), strings.Contains(lower, "exception"), strings.Contains(lower, "crash"), strings.Contains(lower, "error"):
		severity = "error"
	default:
		for _, keyword := range runtimeErrorKeywords {
			if strings.Contains(lower, keyword) {
				severity = "error"
				break
			}
		}
	}
	if severity == "" {
		return RuntimeErrorEvent{}, false
	}
	return RuntimeErrorEvent{
		Source:   "plain",
		Severity: severity,
		Message:  line,
		Raw:      line,
	}, true
}

// BuildRuntimeErrorReport groups error events and adds hints.
func BuildRuntimeErrorReport(events []RuntimeErrorEvent, parseErrors []string, opts RuntimeErrorAnalyzeOptions) RuntimeErrorReport {
	groupsByKey := make(map[string]*RuntimeErrorGroup)
	hints := make([]RuntimeErrorHint, 0)

	for _, event := range events {
		signature := runtimeErrorSignature(event)
		key := strings.Join([]string{event.Severity, event.Process, event.Subsystem, event.Category, signature}, "\x00")
		group := groupsByKey[key]
		if group == nil {
			group = &RuntimeErrorGroup{
				Severity:  event.Severity,
				Process:   event.Process,
				Subsystem: event.Subsystem,
				Category:  event.Category,
				Signature: signature,
			}
			groupsByKey[key] = group
		}
		group.Count++
		if len(group.Examples) < opts.MaxExamples {
			group.Examples = append(group.Examples, event.Message)
		}

		if hintKind := runtimeErrorHintKind(event.Message); hintKind != "" {
			hints = append(hints, RuntimeErrorHint{
				Kind:      hintKind,
				Severity:  event.Severity,
				Process:   event.Process,
				Subsystem: event.Subsystem,
				Message:   event.Message,
			})
		}
	}

	groups := make([]RuntimeErrorGroup, 0, len(groupsByKey))
	for _, group := range groupsByKey {
		groups = append(groups, *group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Count == groups[j].Count {
			if severityRank(groups[i].Severity) == severityRank(groups[j].Severity) {
				return groups[i].Signature < groups[j].Signature
			}
			return severityRank(groups[i].Severity) > severityRank(groups[j].Severity)
		}
		return groups[i].Count > groups[j].Count
	})

	return RuntimeErrorReport{
		EventCount:  len(events),
		Groups:      groups,
		Hints:       hints,
		ParseErrors: parseErrors,
	}
}

func runtimeErrorSignature(event RuntimeErrorEvent) string {
	base := strings.TrimSpace(firstNonEmpty(event.Name, event.Message))
	base = strings.ToLower(base)
	base = hexAddressPattern.ReplaceAllString(base, "<addr>")
	base = digitsPattern.ReplaceAllString(base, "<num>")
	base = whitespacePattern.ReplaceAllString(base, " ")
	base = strings.TrimSpace(base)
	if len(base) > 180 {
		base = base[:180]
	}
	if base == "" {
		return "unknown"
	}
	return base
}

func runtimeErrorHintKind(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "uncaught exception"), strings.Contains(lower, "exception"):
		return "exception"
	case strings.Contains(lower, "fatal error"), strings.Contains(lower, "sigabrt"), strings.Contains(lower, "sigsegv"), strings.Contains(lower, "exc_bad"), strings.Contains(lower, "crash"):
		return "crash"
	case strings.Contains(lower, "hang"):
		return "hang"
	case strings.Contains(lower, "main thread checker"), strings.Contains(lower, "thread performance checker"):
		return "thread-checker"
	case strings.Contains(lower, "publishing changes from background threads"):
		return "swiftui-threading"
	default:
		return ""
	}
}

func severityRank(severity string) int {
	switch normalizeSeverity(severity) {
	case "fault":
		return 3
	case "error":
		return 2
	case "default":
		return 1
	default:
		return 0
	}
}

func normalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fault":
		return "fault"
	case "error":
		return "error"
	case "default", "release":
		return "default"
	case "info":
		return "info"
	case "debug":
		return "debug"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return typed
			}
		case fmt.Stringer:
			if strings.TrimSpace(typed.String()) != "" {
				return typed.String()
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
