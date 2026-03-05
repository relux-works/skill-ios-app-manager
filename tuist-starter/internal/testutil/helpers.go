package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// CaptureOutput captures stdout produced while fn runs.
func CaptureOutput(fn func()) string {
	originalStdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		panic("failed to create stdout pipe: " + err.Error())
	}
	defer func() {
		_ = r.Close()
	}()

	os.Stdout = w
	defer func() {
		os.Stdout = originalStdout
	}()

	outputCh := make(chan string, 1)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		outputCh <- b.String()
	}()

	fn()
	_ = w.Close()

	return <-outputCh
}

// TempDir creates a temporary directory that is automatically cleaned up.
func TempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}
