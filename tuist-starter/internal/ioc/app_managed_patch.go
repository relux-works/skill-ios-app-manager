package ioc

import (
	"fmt"
	"regexp"
	"strings"
)

var registryConfigureCallPattern = regexp.MustCompile(
	`(?m)^([ \t]*)(?:[A-Za-z_][A-Za-z0-9_]*\.)?Registry\.configure\s*\([^\n]*\)[ \t]*$`,
)

// AppManagedBootstrapPatch describes one generator-owned call that must run
// immediately before the app's existing Registry.configure(...) call.
type AppManagedBootstrapPatch struct {
	ID   string
	Call string
}

// ConvergeManagedAppBootstrapContent inserts or updates a byte-idempotent
// bootstrap block without changing the existing Registry.configure(...) call
// or any other app initialization.
func ConvergeManagedAppBootstrapContent(content string, patch AppManagedBootstrapPatch) (string, error) {
	if err := validateManagedPatchID(patch.ID); err != nil {
		return "", err
	}
	call := strings.TrimSpace(patch.Call)
	if call == "" || strings.ContainsAny(call, "\r\n") {
		return "", fmt.Errorf("managed app bootstrap patch %q requires a single-line call", patch.ID)
	}

	begin := managedMarker(patch.ID, "bootstrap", "begin")
	end := managedMarker(patch.ID, "bootstrap", "end")
	hasBegin := strings.Contains(content, begin)
	hasEnd := strings.Contains(content, end)
	if hasBegin != hasEnd {
		return "", fmt.Errorf("managed app bootstrap patch %q has incomplete ownership markers", patch.ID)
	}

	updated := content
	var err error
	if hasBegin {
		updated, err = replaceManagedBlock(updated, begin, end, call)
		if err != nil {
			return "", fmt.Errorf("converge %s app bootstrap: %w", patch.ID, err)
		}
	}

	configureLine, err := uniqueRegistryConfigureCall(updated)
	if err != nil {
		return "", err
	}
	if !hasBegin {
		block := managedBlock(configureLine.indent, begin, end, call)
		updated = updated[:configureLine.start] + block + "\n" + updated[configureLine.start:]
		configureLine, err = uniqueRegistryConfigureCall(updated)
		if err != nil {
			return "", err
		}
	}

	beginIndex := strings.Index(updated, begin)
	endIndex := strings.Index(updated, end)
	if beginIndex < 0 || endIndex < beginIndex || endIndex >= configureLine.start {
		return "", fmt.Errorf("managed app bootstrap patch %q must precede Registry.configure(...) call", patch.ID)
	}
	return updated, nil
}

func uniqueRegistryConfigureCall(content string) (registryLine, error) {
	matches := registryConfigureCallPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) != 1 {
		return registryLine{}, fmt.Errorf(
			"expected exactly one supported Registry.configure(...) call in App.swift, found %d",
			len(matches),
		)
	}
	match := matches[0]
	return registryLine{
		start:  match[0],
		end:    match[1],
		indent: content[match[2]:match[3]],
		text:   content[match[0]:match[1]],
	}, nil
}
