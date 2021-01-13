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
SPACES="  padded  "`
	doc, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := map[string]string{
		"DQ":     "line1\nline2",
		"SQ":     `raw\nnot escaped`,
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
