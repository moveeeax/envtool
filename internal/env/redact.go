package env

import "strings"

// SecretMatchers are the substrings (case-insensitive) that mark a key as
// sensitive by default.
var SecretMatchers = []string{
	"SECRET", "PASSWORD", "PASSWD", "TOKEN", "APIKEY", "API_KEY",
	"PRIVATE", "CREDENTIAL", "AUTH", "ACCESS_KEY",
}

// IsSecretKey reports whether key looks sensitive according to matchers. When
// matchers is empty the built-in SecretMatchers list is used.
func IsSecretKey(key string, matchers []string) bool {
	if len(matchers) == 0 {
		matchers = SecretMatchers
	}
	up := strings.ToUpper(key)
	for _, m := range matchers {
		if strings.Contains(up, strings.ToUpper(m)) {
			return true
		}
	}
	return false
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
