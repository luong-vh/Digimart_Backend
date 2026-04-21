// util/slug.go
package util

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func GenerateSlug(s string) string {
	// Normalize unicode
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)

	// Convert to lowercase
	result = strings.ToLower(result)

	// Replace spaces with hyphens
	result = strings.ReplaceAll(result, " ", "-")

	// Remove special characters
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	result = reg.ReplaceAllString(result, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	result = reg.ReplaceAllString(result, "-")

	// Trim hyphens from start and end
	result = strings.Trim(result, "-")

	return result
}
