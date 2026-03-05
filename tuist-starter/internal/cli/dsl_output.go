package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	outputFormatPretty  = "pretty"
	outputFormatCompact = "compact"
)

func writeDSLResult(out io.Writer, format string, result any) error {
	mode := strings.ToLower(strings.TrimSpace(format))
	if mode == "" {
		mode = outputFormatPretty
	}

	var (
		payload []byte
		err     error
	)

	switch mode {
	case outputFormatPretty:
		payload, err = json.MarshalIndent(result, "", "  ")
	case outputFormatCompact:
		payload, err = json.Marshal(result)
	default:
		return fmt.Errorf("unsupported format %q (supported: pretty, compact)", format)
	}
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}

	_, err = fmt.Fprintln(out, string(payload))
	return err
}
