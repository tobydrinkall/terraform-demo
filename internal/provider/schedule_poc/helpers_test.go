package schedule_poc

import "regexp"

// MustCompileRegex is a test helper that wraps regexp.MustCompile.
func MustCompileRegex(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}
