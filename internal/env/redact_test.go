package env

import (
	"testing"
	"unicode/utf8"
)

func TestIsSecretKey(t *testing.T) {
	secret := []string{"DB_PASSWORD", "API_KEY", "github_token", "AWS_SECRET_ACCESS_KEY"}
	for _, k := range secret {
		if !IsSecretKey(k, nil) {
			t.Errorf("IsSecretKey(%q) = false; want true", k)
		}
	}
	plain := []string{"HOST", "PORT", "DEBUG", "REGION"}
	for _, k := range plain {
		if IsSecretKey(k, nil) {
			t.Errorf("IsSecretKey(%q) = true; want false", k)
		}
	}
}

func TestRedact(t *testing.T) {
	doc := docOf("HOST", "db.local", "DB_PASSWORD", "supersecret", "API_KEY", "ab")
	got := Redact(doc, nil, 2)
	if v, _ := got.Get("HOST"); v != "db.local" {
		t.Errorf("HOST = %q; want db.local", v)
	}
	if v, _ := got.Get("DB_PASSWORD"); v != "su********" {
		t.Errorf("DB_PASSWORD = %q; want su********", v)
	}
	if v, _ := got.Get("API_KEY"); v != "********" {
		t.Errorf("API_KEY = %q; want ******** (too short to keep)", v)
	}
}

// TestIsSecretKeyIgnoresMatcherWhitespace pins the fix for `--match
// "TOKEN, SECRET"`: the untrimmed " SECRET" matcher never matched any key, so
// secrets the user explicitly asked to hide were printed in the clear.
func TestIsSecretKeyIgnoresMatcherWhitespace(t *testing.T) {
	matchers := []string{"TOKEN", " SECRET", "PIN\t"}
	for _, k := range []string{"API_TOKEN", "MY_SECRET", "CARD_PIN"} {
		if !IsSecretKey(k, matchers) {
			t.Errorf("IsSecretKey(%q, %q) = false; want true", k, matchers)
		}
	}
	if IsSecretKey("HOST", matchers) {
		t.Error("IsSecretKey(HOST) = true; want false")
	}
}

// TestIsSecretKeyBlankMatchersFallBack checks that a list with nothing usable in
// it falls back to the built-in matchers rather than matching nothing.
func TestIsSecretKeyBlankMatchersFallBack(t *testing.T) {
	for _, matchers := range [][]string{nil, {}, {""}, {"  ", "\t"}} {
		if !IsSecretKey("DB_PASSWORD", matchers) {
			t.Errorf("IsSecretKey(DB_PASSWORD, %q) = false; want the built-in matchers to apply", matchers)
		}
		if IsSecretKey("HOST", matchers) {
			t.Errorf("IsSecretKey(HOST, %q) = true; want false", matchers)
		}
	}
}

// TestRedactKeepCountsRunesNotBytes pins the fix for mask() slicing by byte
// index. "détente" has a 2-byte 'é'; --keep 2 used to cut the value after the
// first byte of that rune, leaving invalid UTF-8 in the output that a JSON or
// YAML encoder would silently mangle instead of the two-character prefix the
// caller asked to keep.
func TestRedactKeepCountsRunesNotBytes(t *testing.T) {
	doc := docOf("API_SECRET", "détente")
	got := Redact(doc, nil, 2)
	v, _ := got.Get("API_SECRET")
	if !utf8.ValidString(v) {
		t.Fatalf("masked value %q is not valid UTF-8", v)
	}
	if want := "dé********"; v != want {
		t.Errorf("masked value = %q; want %q", v, want)
	}
}

func TestRedactCustomMatchers(t *testing.T) {
	doc := docOf("PIN", "1234", "NAME", "ok")
	got := Redact(doc, []string{"PIN"}, 0)
	if v, _ := got.Get("PIN"); v != "********" {
		t.Errorf("PIN = %q; want ********", v)
	}
	if v, _ := got.Get("NAME"); v != "ok" {
		t.Errorf("NAME = %q; want ok", v)
	}
}
