package format

import (
	"strings"
	"unicode"
)

// NameToKebab derives a folder name from an artifact heading, as the naming
// conventions define it: words are split on whitespace, hyphens, and
// underscores and at PascalCase boundaries, lowercased, and joined with
// hyphens. Runs of capitals are kept together as acronyms.
//
//	"SnapshotWorkspace" → "snapshot-workspace"
//	"Happy Path"        → "happy-path"
//	"HTTPServer"        → "http-server"
//	"AASDD CLI"         → "aasdd-cli"
func NameToKebab(name string) string {
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, strings.ToLower(string(cur)))
			cur = nil
		}
	}
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case unicode.IsSpace(r) || r == '-' || r == '_':
			flush()
		case unicode.IsUpper(r):
			if len(cur) > 0 {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
					flush()
				}
			}
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return strings.Join(words, "-")
}
