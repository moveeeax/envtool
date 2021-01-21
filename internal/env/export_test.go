package env

import (
	"strings"
	"testing"
)

func exportString(t *testing.T, doc *Doc, f Format) string {
	t.Helper()
	var b strings.Builder
	if err := Export(&b, doc, f); err != nil {
		t.Fatalf("Export(%s): %v", f, err)
	}
	return b.String()
}

func TestExportDotenv(t *testing.T) {
	doc := docOf("A", "1", "B", "has space")
	got := exportString(t, doc, FormatDotenv)
	want := "A=1\nB=\"has space\"\n"
	if got != want {
		t.Errorf("dotenv = %q; want %q", got, want)
	}
}

func TestExportShell(t *testing.T) {
	doc := docOf("A", "it's")
	got := exportString(t, doc, FormatShell)
	want := "export A='it'\\''s'\n"
	if got != want {
		t.Errorf("shell = %q; want %q", got, want)
	}
}

func TestExportJSON(t *testing.T) {
	doc := docOf("A", "1")
	got := exportString(t, doc, FormatJSON)
	if !strings.Contains(got, "\"A\": \"1\"") {
		t.Errorf("json = %q; want it to contain A:1", got)
	}
}

func TestExportYAML(t *testing.T) {
	doc := docOf("A", "1", "B", "needs: quote")
	got := exportString(t, doc, FormatYAML)
	if !strings.Contains(got, "A: 1\n") {
		t.Errorf("yaml missing plain scalar: %q", got)
	}
	if !strings.Contains(got, "B: \"needs: quote\"\n") {
		t.Errorf("yaml missing quoted scalar: %q", got)
	}
}

func TestParseFormat(t *testing.T) {
	cases := map[string]Format{
		"": FormatDotenv, "env": FormatDotenv, "sh": FormatShell,
		"JSON": FormatJSON, "yml": FormatYAML,
	}
	for in, want := range cases {
		got, err := ParseFormat(in)
		if err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %q,%v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseFormat("toml"); err == nil {
		t.Error("ParseFormat(toml) expected error")
	}
}
