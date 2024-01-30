//go:build windows
// +build windows

package gore

import (
	"path/filepath"
	"strings"
)

func osAwarePathDir(s string) string {
	if strings.Contains(s, "/") {
		return strings.ReplaceAll(filepath.Dir(s), "\\", "/")
	}
	return s
}

func osAwarePathBase(s string) string {
	if strings.Contains(s, "/") {
		return strings.ReplaceAll(filepath.Base(s), "\\", "/")
	}
	return s
}

func osAwarePathClean(s string) string {
	if strings.Contains(s, "/") {
		return strings.ReplaceAll(filepath.Clean(s), "\\", "/")
	}
	return s
}
