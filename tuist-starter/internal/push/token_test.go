package push

import (
	"context"
	"errors"
	"testing"
)

func TestExtractLatestAPNsTokenReturnsMostRecentMatch(t *testing.T) {
	t.Parallel()

	first := "1111111111111111111111111111111111111111111111111111111111111111"
	second := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	nonAPNSToken := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"

	logOutput := `
2026-02-24 11:00:00.000 MyApp[123:456] didRegisterForRemoteNotificationsWithDeviceToken <` + first + `>
2026-02-24 11:00:02.000 MyApp[123:456] Session token refreshed: ` + nonAPNSToken + `
2026-02-24 11:00:05.000 MyApp[123:456] APNS device token updated: ` + second + `
`

	got, ok := extractLatestAPNsToken(logOutput)
	if !ok {
		t.Fatal("extractLatestAPNsToken() = not found, want latest APNs token")
	}

	if got != second {
		t.Fatalf("extractLatestAPNsToken() = %q, want %q", got, second)
	}
}

func TestExtractLatestAPNsTokenSupportsGroupedHexInAngleBrackets(t *testing.T) {
	t.Parallel()

	want := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	grouped := "<12345678 90abcdef 12345678 90abcdef 12345678 90abcdef 12345678 90abcdef>"
	logOutput := "2026-02-24 11:05:00.000 MyApp didRegisterForRemoteNotificationsWithDeviceToken " + grouped

	got, ok := extractLatestAPNsToken(logOutput)
	if !ok {
		t.Fatal("extractLatestAPNsToken() = not found, want grouped token to be parsed")
	}

	if got != want {
		t.Fatalf("extractLatestAPNsToken() = %q, want %q", got, want)
	}
}

func TestExtractLatestHexTokenReturnsLastMatch(t *testing.T) {
	t.Parallel()

	first := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	second := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	input := "prefix " + first + " middle " + second + " suffix"
	got, ok := extractLatestHexToken(input)
	if !ok {
		t.Fatal("extractLatestHexToken() = not found, want last token")
	}

	if got != second {
		t.Fatalf("extractLatestHexToken() = %q, want %q", got, second)
	}
}

func TestTokenExtractorFallsBackToFileWhenSimulatorCommandFails(t *testing.T) {
	t.Parallel()

	want := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	extractor := &TokenExtractor{
		runCommand: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("xcrun missing")
		},
		readFile: func(path string) ([]byte, error) {
			if path != "token.txt" {
				t.Fatalf("readFile path = %q, want %q", path, "token.txt")
			}
			return []byte("token=" + want), nil
		},
	}

	got, err := extractor.LatestDeviceToken(context.Background(), "token.txt")
	if err != nil {
		t.Fatalf("LatestDeviceToken() error = %v", err)
	}
	if got != want {
		t.Fatalf("LatestDeviceToken() = %q, want %q", got, want)
	}
}
