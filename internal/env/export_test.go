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

// TestExportJSONPreservesOrder pins the fix for JSON being emitted through a Go
// map, which re-sorted keys alphabetically and made --sort a no-op.
func TestExportJSONPreservesOrder(t *testing.T) {
	doc := docOf("ZED", "1", "ALPHA", "2", "MID", "3")
	got := exportString(t, doc, FormatJSON)
	want := "{\n  \"ZED\": \"1\",\n  \"ALPHA\": \"2\",\n  \"MID\": \"3\"\n}\n"
	if got != want {
		t.Errorf("json = %q; want %q", got, want)
	}

	sorted := exportString(t, doc.SortedCopy(), FormatJSON)
	wantSorted := "{\n  \"ALPHA\": \"2\",\n  \"MID\": \"3\",\n  \"ZED\": \"1\"\n}\n"
	if sorted != wantSorted {
		t.Errorf("sorted json = %q; want %q", sorted, wantSorted)
	}
}

func TestExportJSONEmpty(t *testing.T) {
	if got := exportString(t, New(), FormatJSON); got != "{}\n" {
		t.Errorf("json of empty doc = %q; want %q", got, "{}\n")
	}
}

func TestExportYAML(t *testing.T) {
	doc := docOf("A", "plain", "B", "needs: quote")
	got := exportString(t, doc, FormatYAML)
	if !strings.Contains(got, "A: plain\n") {
		t.Errorf("yaml missing plain scalar: %q", got)
	}
	if !strings.Contains(got, "B: \"needs: quote\"\n") {
		t.Errorf("yaml missing quoted scalar: %q", got)
	}
}

// TestExportYAMLQuotesNonStringLookalikes covers values a YAML loader would
// otherwise resolve to a bool, null, number or timestamp instead of a string.
func TestExportYAMLQuotesNonStringLookalikes(t *testing.T) {
	quoted := []string{
		"true", "false", "True", "FALSE", "yes", "no", "on", "off", "y", "n",
		"null", "Null", "~", "",
		"0", "1", "5432", "-1", "1.10", "1e5", "0x1f", "0o17", ".inf", ".nan",
		"2021-03-19",
		"- leading dash", "?query",
	}
	for _, v := range quoted {
		got := exportString(t, docOf("K", v), FormatYAML)
		if !strings.HasPrefix(got, `K: "`) {
			t.Errorf("yamlScalar(%q) = %q; want a double-quoted scalar", v, got)
		}
	}

	plain := []string{"db.local", "postgres", "v1.10.0", "abc123", "1.2.3"}
	for _, v := range plain {
		got := exportString(t, docOf("K", v), FormatYAML)
		if got != "K: "+v+"\n" {
			t.Errorf("yamlScalar(%q) = %q; want an unquoted scalar", v, got)
		}
	}
}

// TestExportDotenvRoundTrip proves that dotenv output re-parses to the same
// document, which is the contract `envtool merge`/`export -f dotenv` relies on.
func TestExportDotenvRoundTrip(t *testing.T) {
	values := []string{
		"plain", "", "has space", "trailing space ", " leading",
		"has\ttab", "has\nnewline", "ends with cr\r", "mid\rcr",
		// No other character here forces quoting, so this is the case that
		// catches a bare trailing \r being eaten by the line scanner.
		"endscr\r",
		"has\"double", "has'single", "has#hash", "has \\ backslash",
		`C:\temp\path`, "value # not a comment", "-----BEGIN KEY-----",
		"true", "1.10", "a=b=c",
	}
	for _, v := range values {
		doc := docOf("K", v)
		out := exportString(t, doc, FormatDotenv)
		back, err := Parse(strings.NewReader(out))
		if err != nil {
			t.Errorf("re-parse of %q (encoded as %q): %v", v, out, err)
			continue
		}
		got, ok := back.Get("K")
		if !ok {
			t.Errorf("key K lost round-tripping %q (encoded as %q)", v, out)
			continue
		}
		if got != v {
			t.Errorf("round trip of %q (encoded as %q) = %q", v, out, got)
		}
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
