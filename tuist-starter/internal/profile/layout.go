package profile

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const (
	layoutXMLStartMarker = "IAM_LAYOUT_XML_START"
	layoutXMLEndMarker   = "IAM_LAYOUT_XML_END"
)

// LayoutAnalyzeOptions configures rendered layout XML analysis.
type LayoutAnalyzeOptions struct {
	MinTapSize    float64 `json:"min_tap_size"`
	MaxElements   int     `json:"max_elements"`
	IncludeHidden bool    `json:"include_hidden"`
}

// LayoutFrame is an element frame in screen coordinates.
type LayoutFrame struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// LayoutScreen captures the root rendered screen size.
type LayoutScreen struct {
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
}

// LayoutElement is one parsed rendered UI element.
type LayoutElement struct {
	Path       string       `json:"path"`
	ParentPath string       `json:"parent_path,omitempty"`
	Depth      int          `json:"depth"`
	Type       string       `json:"type"`
	SourceTag  string       `json:"source_tag,omitempty"`
	Identifier string       `json:"identifier,omitempty"`
	Name       string       `json:"name,omitempty"`
	Label      string       `json:"label,omitempty"`
	Value      string       `json:"value,omitempty"`
	Enabled    *bool        `json:"enabled,omitempty"`
	Hittable   *bool        `json:"hittable,omitempty"`
	Visible    *bool        `json:"visible,omitempty"`
	Accessible *bool        `json:"accessible,omitempty"`
	Selected   *bool        `json:"selected,omitempty"`
	Frame      *LayoutFrame `json:"frame,omitempty"`
	ChildCount int          `json:"child_count"`
}

// LayoutTypeCount summarizes element counts by type.
type LayoutTypeCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// LayoutDuplicateIdentity groups duplicate accessibility identities.
type LayoutDuplicateIdentity struct {
	Identity string   `json:"identity"`
	Count    int      `json:"count"`
	Paths    []string `json:"paths"`
}

// LayoutIssue describes one suspicious rendered layout pattern.
type LayoutIssue struct {
	Kind       string       `json:"kind"`
	Severity   string       `json:"severity"`
	Path       string       `json:"path,omitempty"`
	Type       string       `json:"type,omitempty"`
	Identifier string       `json:"identifier,omitempty"`
	Label      string       `json:"label,omitempty"`
	Frame      *LayoutFrame `json:"frame,omitempty"`
	Message    string       `json:"message"`
}

// LayoutReport summarizes a rendered UI hierarchy.
type LayoutReport struct {
	Source               string                    `json:"source,omitempty"`
	ElementCount         int                       `json:"element_count"`
	ReportedElementCount int                       `json:"reported_element_count"`
	MaxDepth             int                       `json:"max_depth"`
	Screen               LayoutScreen              `json:"screen,omitempty"`
	TypeCounts           []LayoutTypeCount         `json:"type_counts"`
	DuplicateIdentities  []LayoutDuplicateIdentity `json:"duplicate_identities,omitempty"`
	Issues               []LayoutIssue             `json:"issues,omitempty"`
	Elements             []LayoutElement           `json:"elements"`
	ParseErrors          []string                  `json:"parse_errors,omitempty"`
}

// LayoutScaffoldOptions configures XCTest layout probe generation.
type LayoutScaffoldOptions struct {
	ProjectRoot string
	Config      config.ProjectConfig
	OutputPath  string
	Force       bool
}

// LayoutScaffoldResult reports generated layout probe location.
type LayoutScaffoldResult struct {
	Path string `json:"path"`
}

type parsedLayoutTree struct {
	source   string
	screen   LayoutScreen
	elements []LayoutElement
}

// AnalyzeLayoutXML builds a report from rendered layout XML or logs containing IAM layout markers.
func AnalyzeLayoutXML(raw string, opts LayoutAnalyzeOptions) LayoutReport {
	if opts.MinTapSize <= 0 {
		opts.MinTapSize = 44
	}
	if opts.MaxElements <= 0 {
		opts.MaxElements = 200
	}

	tree, parseErrors := ParseLayoutXML(raw)
	screen := tree.screen
	if screen.Width <= 0 || screen.Height <= 0 {
		screen = inferLayoutScreen(tree.elements)
	}

	report := LayoutReport{
		Source:               tree.source,
		ElementCount:         len(tree.elements),
		ReportedElementCount: minInt(len(tree.elements), opts.MaxElements),
		MaxDepth:             maxLayoutDepth(tree.elements),
		Screen:               screen,
		TypeCounts:           countLayoutTypes(tree.elements),
		DuplicateIdentities:  findDuplicateLayoutIdentities(tree.elements),
		Issues:               findLayoutIssues(tree.elements, screen, opts),
		ParseErrors:          parseErrors,
	}
	report.Elements = append(report.Elements, tree.elements[:report.ReportedElementCount]...)
	return report
}

// AnalyzeLayoutXMLFile reads and analyzes rendered layout XML.
func AnalyzeLayoutXMLFile(path string, opts LayoutAnalyzeOptions) (LayoutReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LayoutReport{}, fmt.Errorf("read layout XML %q: %w", path, err)
	}
	return AnalyzeLayoutXML(string(data), opts), nil
}

// ParseLayoutXML parses LayoutHierarchyProbe, Appium/WDA page source, or raw XML hierarchy.
func ParseLayoutXML(raw string) (parsedLayoutTree, []string) {
	payload := strings.TrimSpace(extractLayoutXMLPayload(raw))
	if payload == "" {
		return parsedLayoutTree{}, []string{"layout XML payload not found"}
	}

	decoder := xml.NewDecoder(strings.NewReader(payload))
	decoder.Strict = false

	for {
		token, err := decoder.Token()
		if err != nil {
			return parsedLayoutTree{}, []string{fmt.Sprintf("parse XML root: %v", err)}
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		tree, err := parseLayoutRoot(decoder, start)
		if err != nil {
			return parsedLayoutTree{}, []string{err.Error()}
		}
		return tree, nil
	}
}

func extractLayoutXMLPayload(raw string) string {
	if start := strings.Index(raw, layoutXMLStartMarker); start >= 0 {
		afterStart := raw[start+len(layoutXMLStartMarker):]
		if newline := strings.IndexByte(afterStart, '\n'); newline >= 0 {
			afterStart = afterStart[newline+1:]
		}
		if end := strings.Index(afterStart, layoutXMLEndMarker); end >= 0 {
			return afterStart[:end]
		}
		return afterStart
	}

	trimmed := strings.TrimSpace(raw)
	for _, marker := range []string{"<?xml", "<layout", "<AppiumAUT", "<XCUIElementType", "<element"} {
		if index := strings.Index(trimmed, marker); index >= 0 {
			return trimmed[index:]
		}
	}
	return trimmed
}

func parseLayoutRoot(decoder *xml.Decoder, start xml.StartElement) (parsedLayoutTree, error) {
	source := attrString(start.Attr, "source")
	if source == "" && strings.EqualFold(start.Name.Local, "AppiumAUT") {
		source = "Appium/WDA"
	}
	screen := screenFromAttributes(start.Attr)
	if isLayoutContainer(start) {
		elements, err := parseLayoutChildren(decoder, start, 0, "")
		if err != nil {
			return parsedLayoutTree{}, err
		}
		return parsedLayoutTree{source: source, screen: screen, elements: elements}, nil
	}

	element, elements, err := parseLayoutElement(decoder, start, 0, "", 1)
	if err != nil {
		return parsedLayoutTree{}, err
	}
	elements[0] = element
	return parsedLayoutTree{source: source, screen: screen, elements: elements}, nil
}

func parseLayoutChildren(decoder *xml.Decoder, parent xml.StartElement, depth int, parentPath string) ([]LayoutElement, error) {
	children := make([]LayoutElement, 0)
	siblingCounters := map[string]int{}
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("parse XML children for <%s>: %w", parent.Name.Local, err)
		}
		switch current := token.(type) {
		case xml.StartElement:
			tokenName := pathToken(normalizeLayoutType(current))
			siblingCounters[tokenName]++
			child, flattened, err := parseLayoutElement(decoder, current, depth, parentPath, siblingCounters[tokenName])
			if err != nil {
				return nil, err
			}
			flattened[0] = child
			children = append(children, flattened...)
		case xml.EndElement:
			if current.Name.Local == parent.Name.Local {
				return children, nil
			}
		}
	}
}

func parseLayoutElement(decoder *xml.Decoder, start xml.StartElement, depth int, parentPath string, siblingOrdinal int) (LayoutElement, []LayoutElement, error) {
	elementType := normalizeLayoutType(start)
	tokenName := pathToken(elementType)
	path := fmt.Sprintf("/%s[%d]", tokenName, siblingOrdinal)
	if parentPath != "" {
		path = fmt.Sprintf("%s/%s[%d]", parentPath, tokenName, siblingOrdinal)
	}

	element := LayoutElement{
		Path:       path,
		ParentPath: parentPath,
		Depth:      depth,
		Type:       elementType,
		SourceTag:  start.Name.Local,
		Identifier: attrString(start.Attr, "identifier", "accessibilityIdentifier", "accessibility_identifier"),
		Name:       attrString(start.Attr, "name"),
		Label:      attrString(start.Attr, "label"),
		Value:      attrString(start.Attr, "value"),
		Enabled:    attrBool(start.Attr, "enabled"),
		Hittable:   attrBool(start.Attr, "hittable"),
		Visible:    attrBool(start.Attr, "visible"),
		Accessible: attrBool(start.Attr, "accessible"),
		Selected:   attrBool(start.Attr, "selected"),
		Frame:      frameFromAttributes(start.Attr),
	}

	flattened := []LayoutElement{element}
	siblingCounters := map[string]int{}
	for {
		token, err := decoder.Token()
		if err != nil {
			return element, nil, fmt.Errorf("parse XML element <%s>: %w", start.Name.Local, err)
		}
		switch current := token.(type) {
		case xml.StartElement:
			childToken := pathToken(normalizeLayoutType(current))
			siblingCounters[childToken]++
			child, childElements, err := parseLayoutElement(decoder, current, depth+1, path, siblingCounters[childToken])
			if err != nil {
				return element, nil, err
			}
			childElements[0] = child
			element.ChildCount++
			flattened = append(flattened, childElements...)
		case xml.EndElement:
			if current.Name.Local == start.Name.Local {
				flattened[0] = element
				return element, flattened, nil
			}
		}
	}
}

func isLayoutContainer(start xml.StartElement) bool {
	name := strings.ToLower(start.Name.Local)
	return name == "layout" || name == "appiumaut" || name == "hierarchy"
}

func normalizeLayoutType(start xml.StartElement) string {
	value := attrString(start.Attr, "type", "elementType", "element_type")
	if value == "" {
		value = start.Name.Local
	}
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "XCUIElementType")
	value = strings.TrimPrefix(value, "UIA")
	if value == "" {
		return "unknown"
	}
	if value == strings.ToUpper(value) {
		value = strings.ToLower(value)
	}
	value = strings.ReplaceAll(value, "_", " ")
	parts := strings.Fields(value)
	if len(parts) > 1 {
		for i := range parts {
			parts[i] = strings.ToLower(parts[i])
		}
		return strings.Join(parts, "-")
	}
	return lowerFirst(value)
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func pathToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "element"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "element"
	}
	return out
}

func attrString(attrs []xml.Attr, names ...string) string {
	for _, name := range names {
		for _, attr := range attrs {
			if strings.EqualFold(attr.Name.Local, name) {
				return strings.TrimSpace(attr.Value)
			}
		}
	}
	return ""
}

func attrFloat(attrs []xml.Attr, names ...string) (float64, bool) {
	raw := attrString(attrs, names...)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func attrBool(attrs []xml.Attr, names ...string) *bool {
	raw := strings.ToLower(attrString(attrs, names...))
	switch raw {
	case "true", "yes", "1":
		value := true
		return &value
	case "false", "no", "0":
		value := false
		return &value
	default:
		return nil
	}
}

func frameFromAttributes(attrs []xml.Attr) *LayoutFrame {
	x, okX := attrFloat(attrs, "x")
	y, okY := attrFloat(attrs, "y")
	width, okWidth := attrFloat(attrs, "width", "w")
	height, okHeight := attrFloat(attrs, "height", "h")
	if !okX && !okY && !okWidth && !okHeight {
		return nil
	}
	return &LayoutFrame{X: x, Y: y, Width: width, Height: height}
}

func screenFromAttributes(attrs []xml.Attr) LayoutScreen {
	width, _ := attrFloat(attrs, "screenWidth", "screen_width", "width")
	height, _ := attrFloat(attrs, "screenHeight", "screen_height", "height")
	return LayoutScreen{Width: width, Height: height}
}

func inferLayoutScreen(elements []LayoutElement) LayoutScreen {
	for _, element := range elements {
		if element.Frame == nil {
			continue
		}
		if element.Type == "application" || element.Type == "window" {
			return LayoutScreen{Width: element.Frame.Width, Height: element.Frame.Height}
		}
	}
	var maxX, maxY float64
	for _, element := range elements {
		if element.Frame == nil {
			continue
		}
		if right := element.Frame.X + element.Frame.Width; right > maxX {
			maxX = right
		}
		if bottom := element.Frame.Y + element.Frame.Height; bottom > maxY {
			maxY = bottom
		}
	}
	return LayoutScreen{Width: maxX, Height: maxY}
}

func maxLayoutDepth(elements []LayoutElement) int {
	maxDepth := 0
	for _, element := range elements {
		if element.Depth > maxDepth {
			maxDepth = element.Depth
		}
	}
	return maxDepth
}

func countLayoutTypes(elements []LayoutElement) []LayoutTypeCount {
	counts := map[string]int{}
	for _, element := range elements {
		counts[element.Type]++
	}
	out := make([]LayoutTypeCount, 0, len(counts))
	for elementType, count := range counts {
		out = append(out, LayoutTypeCount{Type: elementType, Count: count})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Type < out[j].Type
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func findDuplicateLayoutIdentities(elements []LayoutElement) []LayoutDuplicateIdentity {
	pathsByIdentity := map[string][]string{}
	for _, element := range elements {
		identity := layoutIdentity(element)
		if identity == "" {
			continue
		}
		pathsByIdentity[identity] = append(pathsByIdentity[identity], element.Path)
	}

	out := make([]LayoutDuplicateIdentity, 0)
	for identity, paths := range pathsByIdentity {
		if len(paths) < 2 {
			continue
		}
		out = append(out, LayoutDuplicateIdentity{
			Identity: identity,
			Count:    len(paths),
			Paths:    paths,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Identity < out[j].Identity
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func findLayoutIssues(elements []LayoutElement, screen LayoutScreen, opts LayoutAnalyzeOptions) []LayoutIssue {
	issues := make([]LayoutIssue, 0)
	for _, element := range elements {
		if !opts.IncludeHidden && isExplicitlyHidden(element) {
			continue
		}
		if isInteractiveLayoutType(element.Type) && layoutIdentity(element) == "" && strings.TrimSpace(element.Label) == "" {
			issues = append(issues, LayoutIssue{
				Kind:     "missing-accessibility-identity",
				Severity: "warning",
				Path:     element.Path,
				Type:     element.Type,
				Frame:    element.Frame,
				Message:  "interactive element has no identifier/name/label",
			})
		}
		if isInteractiveLayoutType(element.Type) && element.Frame != nil && element.Frame.Width > 0 && element.Frame.Height > 0 {
			if element.Frame.Width < opts.MinTapSize || element.Frame.Height < opts.MinTapSize {
				issues = append(issues, issueForElement("tiny-tap-target", "warning", element, fmt.Sprintf("interactive frame is smaller than %.0fx%.0f", opts.MinTapSize, opts.MinTapSize)))
			}
		}
		if element.Frame != nil && screen.Width > 0 && screen.Height > 0 && !frameIntersectsScreen(*element.Frame, screen) {
			issues = append(issues, issueForElement("offscreen", "warning", element, "element frame does not intersect the inferred screen bounds"))
		}
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Kind == issues[j].Kind {
			return issues[i].Path < issues[j].Path
		}
		return issues[i].Kind < issues[j].Kind
	})
	return issues
}

func issueForElement(kind string, severity string, element LayoutElement, message string) LayoutIssue {
	return LayoutIssue{
		Kind:       kind,
		Severity:   severity,
		Path:       element.Path,
		Type:       element.Type,
		Identifier: layoutIdentity(element),
		Label:      element.Label,
		Frame:      element.Frame,
		Message:    message,
	}
}

func layoutIdentity(element LayoutElement) string {
	return firstNonEmpty(element.Identifier, element.Name)
}

func isExplicitlyHidden(element LayoutElement) bool {
	return element.Visible != nil && !*element.Visible
}

func isInteractiveLayoutType(elementType string) bool {
	switch strings.ToLower(elementType) {
	case "button", "link", "textfield", "securetextfield", "textview", "switch", "toggle", "slider", "cell", "tab", "menuitem", "picker", "pickerwheel", "segmentedcontrol", "searchfield":
		return true
	default:
		return false
	}
}

func frameIntersectsScreen(frame LayoutFrame, screen LayoutScreen) bool {
	if frame.Width <= 0 || frame.Height <= 0 {
		return true
	}
	return frame.X+frame.Width > 0 && frame.Y+frame.Height > 0 && frame.X < screen.Width && frame.Y < screen.Height
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// ScaffoldLayoutProbe writes an XCTest helper that dumps rendered accessibility hierarchy XML.
func ScaffoldLayoutProbe(opts LayoutScaffoldOptions) (LayoutScaffoldResult, error) {
	root := strings.TrimSpace(opts.ProjectRoot)
	if root == "" {
		root = "."
	}
	appName := strings.TrimSpace(opts.Config.AppName)
	if appName == "" {
		return LayoutScaffoldResult{}, fmt.Errorf("app_name is required to choose default layout probe path")
	}

	outputPath := strings.TrimSpace(opts.OutputPath)
	if outputPath == "" {
		outputPath = filepath.Join("Targets", appName+"UITests", "Sources", "Diagnostics", "LayoutHierarchyProbe.swift")
	}
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(root, outputPath)
	}
	outputPath = filepath.Clean(outputPath)

	if !opts.Force {
		if _, err := os.Stat(outputPath); err == nil {
			return LayoutScaffoldResult{}, fmt.Errorf("layout probe already exists at %q; pass --force to overwrite", outputPath)
		} else if !os.IsNotExist(err) {
			return LayoutScaffoldResult{}, fmt.Errorf("stat layout probe %q: %w", outputPath, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return LayoutScaffoldResult{}, fmt.Errorf("create layout probe directory: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(GenerateLayoutHierarchyProbeSwift()), 0o644); err != nil {
		return LayoutScaffoldResult{}, fmt.Errorf("write layout probe %q: %w", outputPath, err)
	}

	return LayoutScaffoldResult{Path: outputPath}, nil
}
