package env

import "strings"

// SecretMatchers are the substrings (case-insensitive) that mark a key as
// sensitive by default.
var SecretMatchers = []string{
	"SECRET", "PASSWORD", "PASSWD", "TOKEN", "APIKEY", "API_KEY",
	"PRIVATE", "CREDENTIAL", "AUTH", "ACCESS_KEY",
}

// IsSecretKey reports whether key looks sensitive according to matchers.
// Matchers are compared case-insensitively; surrounding whitespace is ignored
// and blank entries are dropped, so a list such as "TOKEN, SECRET" split on
// commas behaves the same as "TOKEN,SECRET". When no usable matcher remains the
// built-in SecretMatchers list is used, so an unusable list can never cause a
// secret to be emitted in the clear.
func IsSecretKey(key string, matchers []string) bool {
	list := normalizeMatchers(matchers)
	if len(list) == 0 {
		list = normalizeMatchers(SecretMatchers)
	}
	up := strings.ToUpper(key)
	for _, m := range list {
		if strings.Contains(up, m) {
			return true
		}
	}
	return false
}

// normalizeMatchers trims and upper-cases matchers, dropping blank entries.
func normalizeMatchers(matchers []string) []string {
	out := make([]string, 0, len(matchers))
	for _, m := range matchers {
		if m = strings.TrimSpace(m); m != "" {
			out = append(out, strings.ToUpper(m))
		}
	}
	return out
}

// Redact returns a copy of doc with the values of sensitive keys masked. Each
// masked value keeps up to keep leading characters and replaces the rest with
// '*'; short values are fully masked to a fixed width to avoid leaking length.
func Redact(doc *Doc, matchers []string, keep int) *Doc {
	out := New()
	for _, e := range doc.entries {
		if IsSecretKey(e.Key, matchers) {
			out.Set(e.Key, mask(e.Value, keep))
			continue
		}
		out.Set(e.Key, e.Value)
	}
	return out
}

func mask(value string, keep int) string {
	if keep < 0 {
		keep = 0
	}
	if keep == 0 || len(value) <= keep {
		return "********"
	}
	return value[:keep] + strings.Repeat("*", 8)
}
