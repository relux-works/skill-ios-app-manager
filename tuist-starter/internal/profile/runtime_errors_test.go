package profile

import "testing"

func TestAnalyzeRuntimeErrorsParsesUnifiedLogAndIAMError(t *testing.T) {
	t.Parallel()

	raw := `
{"timestamp":"2026-05-19 10:00:00","logType":"error","process":"DemoApp","subsystem":"com.example.demo","category":"network","composedMessage":"Request failed with code 500 for id 123"}
{"timestamp":"2026-05-19 10:00:01","logType":"error","process":"DemoApp","subsystem":"com.example.demo","category":"network","composedMessage":"Request failed with code 404 for id 456"}
IAM_ERROR {"severity":"fault","name":"CrashGuard","message":"Uncaught exception NSInvalidArgumentException","process":"DemoApp"}
Thread 1: Fatal error: Index out of range
`

	report := AnalyzeRuntimeErrors(raw, RuntimeErrorAnalyzeOptions{MaxExamples: 2})
	if report.EventCount != 4 {
		t.Fatalf("event count = %d, want 4", report.EventCount)
	}
	if len(report.Groups) != 3 {
		t.Fatalf("group count = %d, want 3: %#v", len(report.Groups), report.Groups)
	}
	if report.Groups[0].Count != 2 || report.Groups[0].Signature != "request failed with code <num> for id <num>" {
		t.Fatalf("first group = %#v", report.Groups[0])
	}
	if len(report.Hints) < 2 {
		t.Fatalf("hints = %#v, want exception/crash hints", report.Hints)
	}
}

func TestParseRuntimeErrorsSkipsDefaultUnifiedLogsByDefault(t *testing.T) {
	t.Parallel()

	raw := `{"logType":"default","process":"DemoApp","composedMessage":"normal lifecycle message"}`
	report := AnalyzeRuntimeErrors(raw, RuntimeErrorAnalyzeOptions{})
	if report.EventCount != 0 {
		t.Fatalf("event count = %d, want 0", report.EventCount)
	}
}
