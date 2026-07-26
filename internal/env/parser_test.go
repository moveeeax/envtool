package env

import (
	"strings"
	"testing"
)

func TestParseBasic(t *testing.T) {
	in := `
# a comment
FOO=bar
export BAZ=qux
EMPTY=
`
	doc, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := doc.Len(); got != 3 {
		t.Fatalf("Len = %d, want 3", got)
	}
	cases := map[string]string{"FOO": "bar", "BAZ": "qux", "EMPTY": ""}
	for k, want := range cases {
		if v, ok := doc.Get(k); !ok || v != want {
			t.Errorf("Get(%q) = %q,%v; want %q", k, v, ok, want)
		}
	}
}

func TestParseQuoted(t *testing.T) {
	in := `DQ="line1\nline2"
SQ='raw\nnot escaped'
INLINE=value # trailing comment
SPACES="  padded  "`
	doc, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := map[string]string{
		"DQ":     "line1\nline2",
		"SQ":     `raw\nnot escaped`,
		"INLINE": "value",
		"SPACES": "  padded  ",
	}
	for k, w := range want {
		if v, _ := doc.Get(k); v != w {
			t.Errorf("Get(%q) = %q; want %q", k, v, w)
		}
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{
		"NOEQUALS",
		"1BAD=x",
		"BAD KEY=x",
		`UNTERM="oops`,
	}
	for _, in := range cases {
		if _, err := Parse(strings.NewReader(in)); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", in)
		}
	}
}

// TestParseErrorsDoNotLeakLineContent pins the fix for parse errors echoing the
// raw line. A line without '=' is very often a stray continuation of a
// multi-line credential, and the message ends up in stderr and CI logs.
func TestParseErrorsDoNotLeakLineContent(t *testing.T) {
	const secret = "MIIEowIBAAKCAQEAsuperSecretKeyMaterial"
	in := "PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----\n" + secret + "\n"
	_, err := Parse(strings.NewReader(in))
	if err == nil {
		t.Fatal("Parse expected an error for a line without '='")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error %q leaks the line content", err)
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error %q should still identify the line number", err)
	}
}

func TestParseExportPrefix(t *testing.T) {
	in := "export A=1\nexport\tB=2\nexport   C=3\nexport=4\nD=5\n"
	doc, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := map[string]string{"A": "1", "B": "2", "C": "3", "export": "4", "D": "5"}
	if doc.Len() != len(want) {
		t.Fatalf("Len = %d, want %d (keys %v)", doc.Len(), len(want), doc.Keys())
	}
	for k, w := range want {
		if v, ok := doc.Get(k); !ok || v != w {
			t.Errorf("Get(%q) = %q,%v; want %q", k, v, ok, w)
		}
	}
}

// TestParseStripsLeadingBOM pins the fix for a UTF-8 byte-order mark on the
// first line. Editors and tools on Windows commonly write one (PowerShell's
// `Out-File`, Notepad's default "UTF-8"), and without stripping it the BOM
// glued itself onto the first key, failing every command on an otherwise
// valid file.
func TestParseStripsLeadingBOM(t *testing.T) {
	in := "\xEF\xBB\xBFFOO=bar\nBAZ=qux\n"
	doc, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if v, ok := doc.Get("FOO"); !ok || v != "bar" {
		t.Errorf("Get(FOO) = %q,%v; want \"bar\",true", v, ok)
	}
	if v, ok := doc.Get("BAZ"); !ok || v != "qux" {
		t.Errorf("Get(BAZ) = %q,%v; want \"qux\",true", v, ok)
	}
	if doc.Has("\uFEFFFOO") {
		t.Error("the BOM should be stripped, not become part of the key")
	}
}

func TestParseLastWins(t *testing.T) {
	doc, err := Parse(strings.NewReader("K=1\nK=2\nK=3"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if doc.Len() != 1 {
		t.Fatalf("Len = %d, want 1", doc.Len())
	}
	if v, _ := doc.Get("K"); v != "3" {
		t.Errorf("Get(K) = %q; want 3", v)
	}
}
