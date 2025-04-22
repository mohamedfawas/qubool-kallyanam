package strings

import (
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
)

var (
	// Common regex patterns
	scriptTagPattern  = regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`)
	multiSpacePattern = regexp.MustCompile(`\s+`)
	htmlTagPattern    = regexp.MustCompile(`<[^>]*>`)

	// HTML sanitizer policy
	strictPolicy = bluemonday.StrictPolicy()

	// Characters often used in SQL injection attempts
	sqlChars = []string{"'", "\"", ";", "--", "/*", "*/", "\\", "%"}
)

// SanitizeHTML removes all HTML tags and potential XSS content from input
func SanitizeHTML(input string) string {
	if input == "" {
		return ""
	}

	// Use bluemonday for comprehensive HTML sanitization
	return strictPolicy.Sanitize(input)
}

// SanitizeBasic performs basic sanitization without external dependencies
// Suitable for simple text inputs where full HTML sanitization is not needed
func SanitizeBasic(input string) string {
	if input == "" {
		return ""
	}

	// HTML escape
	escaped := html.EscapeString(input)

	// Remove script tags completely
	noScripts := scriptTagPattern.ReplaceAllString(escaped, "")

	// Remove all other HTML tags
	noTags := htmlTagPattern.ReplaceAllString(noScripts, "")

	// Normalize whitespace
	normalized := multiSpacePattern.ReplaceAllString(noTags, " ")

	return strings.TrimSpace(normalized)
}

// SanitizeName sanitizes a name input
func SanitizeName(name string) string {
	if name == "" {
		return ""
	}

	// Basic sanitization
	name = SanitizeBasic(name)

	// Remove any non-alphabetic characters except spaces and common punctuation
	var result strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsSpace(r) || r == '.' || r == '-' || r == '\'' {
			result.WriteRune(r)
		}
	}

	// Normalize spaces
	return multiSpacePattern.ReplaceAllString(result.String(), " ")
}

// SanitizeSearchQuery sanitizes a search query
func SanitizeSearchQuery(query string) string {
	if query == "" {
		return ""
	}

	// Basic sanitization
	query = SanitizeBasic(query)

	// Remove SQL injection characters
	for _, char := range sqlChars {
		query = strings.ReplaceAll(query, char, "")
	}

	// Additional search-specific sanitization can be added here

	return strings.TrimSpace(query)
}

// SanitizeUsername sanitizes a username
func SanitizeUsername(username string) string {
	if username == "" {
		return ""
	}

	// Basic sanitization
	username = SanitizeBasic(username)

	// Remove any non-alphanumeric characters except underscores
	var result strings.Builder
	for _, r := range username {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// TruncateString truncates a string to specified length and adds ellipsis if needed
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}
