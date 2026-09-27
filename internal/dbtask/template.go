package dbtask

import (
	"fmt"
	"regexp"
)

var envPlaceholder = regexp.MustCompile(`\{\{\s*\.([A-Za-z_][A-Za-z_0-9]*)\s*\}\}`)

// Substitute expands {{.NAME}} references from the provided environment
// lookup. SQL templates are trusted configuration, not safe query parameters:
// callers should never substitute untrusted user input into SQL.
func Substitute(query string, lookup func(string) (string, bool)) (string, error) {
	var missing string
	result := envPlaceholder.ReplaceAllStringFunc(query, func(match string) string {
		key := envPlaceholder.FindStringSubmatch(match)[1]
		value, ok := lookup(key)
		if !ok {
			missing = key
			return match
		}
		return value
	})
	if missing != "" {
		return "", fmt.Errorf("SQL template environment variable %q is not set", missing)
	}
	return result, nil
}
