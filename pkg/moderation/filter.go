package moderation

import (
	"strings"
	"unicode"
)

func ContainsFlag(content string, keywords []string) bool {
	lower := strings.ToLower(content)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	wordSet := make(map[string]bool, len(words))
	for _, w := range words {
		wordSet[w] = true
	}
	for _, kw := range keywords {
		if wordSet[strings.ToLower(kw)] {
			return true
		}
	}
	return false
}
