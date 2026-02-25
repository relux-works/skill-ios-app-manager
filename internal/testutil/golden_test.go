package testutil

import "testing"

func TestAssertGoldenFile(t *testing.T) {
	AssertGoldenFile(t, "sample_output", "hello from golden test\n")
}
