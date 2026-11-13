package notes

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// SearchSnippet returns bounded context around the first body match. It returns
// an empty string when the body does not match. Limit is measured in runes.
func SearchSnippet(body, query string, caseSensitive bool, limit int) string {
	if query == "" || limit <= 0 {
		return ""
	}
	_, body, _ = readMetadata(body)
	if strings.HasPrefix(body, "# ") {
		_, body, _ = strings.Cut(body, "\n")
		body = strings.TrimLeft(body, "\r\n")
	}
	searchBody, searchQuery := body, query
	if !caseSensitive {
		searchBody = strings.ToLower(body)
		searchQuery = strings.ToLower(query)
	}
	byteIndex := strings.Index(searchBody, searchQuery)
	if byteIndex < 0 {
		return ""
	}
	startMatch := utf8.RuneCountInString(searchBody[:byteIndex])
	matchLength := utf8.RuneCountInString(searchQuery)
	runes := []rune(body)
	start := max(0, startMatch-max(0, limit-matchLength)/2)
	end := min(len(runes), start+limit)
	if end-start < limit {
		start = max(0, end-limit)
	}
	excerpt := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, string(runes[start:end]))
	excerpt = strings.Join(strings.Fields(excerpt), " ")
	if start > 0 {
		excerpt = "…" + excerpt
	}
	if end < len(runes) {
		excerpt += "…"
	}
	return excerpt
}
