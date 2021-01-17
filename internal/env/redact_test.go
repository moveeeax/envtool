package env

import "testing"

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
