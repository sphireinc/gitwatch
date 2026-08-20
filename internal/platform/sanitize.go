package platform

import (
	"regexp"
	"strings"
)

var secretPattern = regexp.MustCompile(`(?i)(token|password|passwd|secret|authorization|x-access-token)([=:][^\s,;]+)`)
var urlUserPattern = regexp.MustCompile(`(?i)(https?://)[^/\s@]+@`)

// SafeText removes terminal control and escape bytes from untrusted Git text.
func SafeText(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r == '\n' || r == '\t' || (r >= 0x20 && r != 0x7f) {
			b.WriteRune(r)
		} else {
			b.WriteRune('�')
		}
	}
	return b.String()
}

func RedactSecrets(value string) string {
	value = secretPattern.ReplaceAllString(value, "$1=[REDACTED]")
	if strings.Contains(value, "https://") || strings.Contains(value, "http://") {
		value = redactURLUserInfo(value)
	}
	return value
}

func redactURLUserInfo(value string) string {
	return urlUserPattern.ReplaceAllString(value, "$1[REDACTED]@")
}
