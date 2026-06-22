package telemetry

import "regexp"

// Redaction is applied at write time, even on the user's own machine, so
// disk forensics of a stolen ~/.thask/ never recovers credentials.

var (
	// --token X or --token=X argv pattern.
	reCLIToken = regexp.MustCompile(`(--token[=\s]+)\S+`)
	// URL credential https://user:pass@host
	reURLCred = regexp.MustCompile(`(https?://)([^:@/\s]+):([^@/\s]+)@`)
	// HTTP Authorization: Bearer X
	reBearer = regexp.MustCompile(`(?i)(Authorization\s*:\s*Bearer\s+)\S+`)
	// Thask-prefixed token thsk_...
	rePrefixToken = regexp.MustCompile(`thsk_[A-Za-z0-9_-]+`)
	// JWT eyJ... (eyJ is the base64 of {"alg or {"typ)
	reJWT = regexp.MustCompile(`eyJ[A-Za-z0-9._-]+`)
	// Cookie / Set-Cookie header line (mask value, keep header name).
	reCookieLine = regexp.MustCompile(`(?im)^((?:Set-)?Cookie\s*:).*$`)
)

// Redact returns s with token-like substrings masked. Safe to call on
// any free-form string (argv, stdout, response body) — masking only
// recognised secret shapes leaves user content intact.
//
// Capture-group references use ${N} form so adjacent letters in the
// replacement text are not absorbed into the group name by Go's expander.
func Redact(s string) string {
	s = reCLIToken.ReplaceAllString(s, "${1}thsk_***")
	s = reURLCred.ReplaceAllString(s, "${1}***:***@")
	s = reBearer.ReplaceAllString(s, "${1}thsk_***")
	s = rePrefixToken.ReplaceAllString(s, "thsk_***")
	s = reJWT.ReplaceAllString(s, "eyJ***")
	return s
}

// RedactHeader additionally masks Cookie / Set-Cookie lines wholesale.
// Use on HTTP header dumps rather than argv.
func RedactHeader(s string) string {
	s = Redact(s)
	return reCookieLine.ReplaceAllString(s, "${1} ***")
}
