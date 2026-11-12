package env

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// FingerprintLen is the number of hex characters Fingerprint returns.
const FingerprintLen = 12

// Fingerprint returns a short keyed hash of value: the first FingerprintLen
// hex characters of HMAC-SHA256(salt, value). Two values with the same
// fingerprint under the same salt are, for review purposes, the same value.
//
// The salt is what stops a fingerprint from being a lookup key: an unsalted
// SHA-256 of a PIN or a short password can be reversed by trying every
// candidate. Anyone who has the salt can still test guesses, so the salt
// needs the same care as the secrets it covers.
func Fingerprint(salt []byte, value string) string {
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))[:FingerprintLen]
}

// Shape describes a value without revealing it: its length, the classes of
// characters it uses and what kind of token it parses as.
type Shape struct {
	Length  int      `json:"length"`
	Classes []string `json:"classes"`
	Kind    string   `json:"kind"`
}

func (s Shape) String() string {
	classes := strings.Join(s.Classes, "+")
	if classes == "" {
		classes = "none"
	}
	return fmt.Sprintf("len=%d chars=%s kind=%s", s.Length, classes, s.Kind)
}

// ShapeOf returns the shape of value.
func ShapeOf(value string) Shape {
	return Shape{
		Length:  utf8.RuneCountInString(value),
		Classes: charClasses(value),
		Kind:    tokenKind(value),
	}
}

// charClasses lists, in a fixed order, which of lower, upper, digit, symbol
// (other printable ASCII), space and non-ascii occur in value.
func charClasses(value string) []string {
	var lower, upper, digit, symbol, space, other bool
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			space = true
		case r > ' ' && r < 0x7f:
			symbol = true
		default:
			other = true
		}
	}
	out := []string{}
	for _, c := range []struct {
		on   bool
		name string
	}{{lower, "lower"}, {upper, "upper"}, {digit, "digit"}, {symbol, "symbol"}, {space, "space"}, {other, "non-ascii"}} {
		if c.on {
			out = append(out, c.name)
		}
	}
	return out
}

// tokenKind names the first kind value parses as, from most to least
// specific: empty, int, bool, uuid, jwt, url, hex, base64, text. hex and
// base64 need at least 16 characters, so short words are not reported as
// encoded data.
func tokenKind(value string) string {
	switch {
	case value == "":
		return "empty"
	case isInt(value):
		return "int"
	case isBool(value):
		return "bool"
	case isUUID(value):
		return "uuid"
	case isJWT(value):
		return "jwt"
	case isURL(value):
		return "url"
	case len(value) >= 16 && isHex(value):
		return "hex"
	case len(value) >= 16 && isBase64(value):
		return "base64"
	default:
		return "text"
	}
}

func isInt(v string) bool {
	_, err := strconv.ParseInt(v, 10, 64)
	return err == nil
}

func isHex(v string) bool {
	for _, c := range v {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// isUUID matches the 8-4-4-4-12 hex layout of RFC 9562.
func isUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, c := range v {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !isHex(string(c)) {
			return false
		}
	}
	return true
}

// isJWT matches the compact JWS layout of RFC 7515: three base64url segments
// separated by dots, the first of which decodes to a JSON object.
func isJWT(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		return false
	}
	for _, p := range parts {
		if _, err := base64.RawURLEncoding.DecodeString(p); err != nil {
			return false
		}
	}
	header, _ := base64.RawURLEncoding.DecodeString(parts[0])
	return strings.HasPrefix(strings.TrimSpace(string(header)), "{")
}

// isBase64 accepts standard or URL-safe base64, padded or not.
func isBase64(v string) bool {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		if _, err := enc.DecodeString(v); err == nil {
			return true
		}
	}
	return false
}
