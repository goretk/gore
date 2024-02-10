// Package extern Copied from src/go/version
package extern

import "strings"

// StripGo converts from a "go1.21-bigcorp" version to a "1.21" version.
// If v does not start with "go", stripGo returns the empty string (a known invalid version).
func StripGo(v string) string {
	v, _, _ = strings.Cut(v, "-") // strip -bigcorp suffix.
	if len(v) < 2 || v[:2] != "go" {
		return ""
	}
	return v[2:]
}
