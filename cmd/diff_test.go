package cmd

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/moveeeax/envtool/internal/env"
)

// TestDiffCommand pins the default output: keys and change kinds only. diff
// used to print both values in full, which leaked old and new secrets into
// any PR or CI log the output was pasted into.
func TestDiffCommand(t *testing.T) {
	a := writeTemp(t, "diffa.env", "SAME=1\nCHANGED=old-secret\nGONE=gone-secret\n")
	b := writeTemp(t, "diffb.env", "SAME=1\nCHANGED=new-secret\nNEW=new-value\n")
	out, err := run(t, "diff", a, b)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	want := "- GONE\n+ NEW\n~ CHANGED (changed)\n"
	if got := sortedLines(out); got != sortedLines(want) {
		t.Errorf("diff output = %q; want %q", out, want)
	}
	for _, v := range []string{"secret", "new-value"} {
		if strings.Contains(out, v) {
			t.Errorf("diff output %q contains a value", out)
		}
	}
	out, err = run(t, "diff", "--keys-only", a, b)
	if err != nil || sortedLines(out) != sortedLines(want) {
		t.Errorf("diff --keys-only = %q, %v; want the default output", out, err)
	}
}

func TestDiffCommandShowValues(t *testing.T) {
	a := writeTemp(t, "diffa.env", "SAME=1\nCHANGED=old\nGONE=x\n")
	b := writeTemp(t, "diffb.env", "SAME=1\nCHANGED=new\nNEW=y\n")
	out, err := run(t, "diff", "--show-values", a, b)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	for _, want := range []string{"+ NEW=y", "- GONE=x", "~ CHANGED: old -> new"} {
		if !strings.Contains(out, want) {
			t.Errorf("diff output = %q; want to contain %q", out, want)
		}
	}
	if strings.Contains(out, "SAME") {
		t.Errorf("diff output = %q; unchanged keys should not appear", out)
	}
}

// With a fixed salt two runs print the same fingerprint for the same value,
// which is how a reviewer confirms that staging and prod now share a value
// without seeing it.
func TestDiffCommandFingerprint(t *testing.T) {
	t.Setenv(saltEnvVar, "review-salt")
	a := writeTemp(t, "a.env", "TOKEN=old\n")
	b := writeTemp(t, "b.env", "TOKEN=new\n")
	c := writeTemp(t, "c.env", "OTHER=1\nTOKEN=new\n")
	first, err := run(t, "diff", "--fingerprint", a, b)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	second, err := run(t, "diff", "--fingerprint", a, c)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	want := "~ TOKEN (changed) [fp=" + env.Fingerprint([]byte("review-salt"), "old") +
		" -> fp=" + env.Fingerprint([]byte("review-salt"), "new") + "]\n"
	if first != want {
		t.Errorf("first run = %q; want %q", first, want)
	}
	if !strings.Contains(second, want) {
		t.Errorf("second run = %q; want the same TOKEN line %q", second, want)
	}
}

func TestDiffCommandShape(t *testing.T) {
	a := writeTemp(t, "a.env", "GONE=abc\n")
	b := writeTemp(t, "b.env", "NEW=12\n")
	out, err := run(t, "diff", "--shape", a, b)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	for _, want := range []string{"- GONE [len=3 chars=lower kind=text]", "+ NEW [len=2 chars=digit kind=int]"} {
		if !strings.Contains(out, want) {
			t.Errorf("diff --shape = %q; want %q", out, want)
		}
	}
}

func TestDiffCommandJSON(t *testing.T) {
	a := writeTemp(t, "a.env", "SAME=1\nCHANGED=old\nGONE=x\n")
	b := writeTemp(t, "b.env", "SAME=1\nCHANGED=new\nNEW=y\n")
	out, err := run(t, "diff", "-f", "json", a, b)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	want := []map[string]any{
		{"key": "CHANGED", "kind": "changed"},
		{"key": "GONE", "kind": "removed"},
		{"key": "NEW", "kind": "added"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("diff -f json = %v; want %v", got, want)
	}

	out, err = run(t, "diff", "-f", "json", "--show-values", "--shape", a, b)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if got[0]["left"] != "old" || got[0]["right"] != "new" || got[0]["right_shape"] == nil {
		t.Errorf("changed entry = %v; want values and shapes", got[0])
	}
	if _, ok := got[1]["right"]; ok {
		t.Errorf("removed entry = %v; has no right side", got[1])
	}

	same := writeTemp(t, "same.env", "A=1\n")
	if out, _ := run(t, "diff", "-f", "json", same, same); strings.TrimSpace(out) != "[]" {
		t.Errorf("diff -f json of equal files = %q; want []", out)
	}
}

func TestDiffCommandFlagErrors(t *testing.T) {
	a := writeTemp(t, "a.env", "A=1\n")
	for _, args := range [][]string{
		{"diff", "--keys-only", "--show-values", a, a},
		{"diff", "--keys-only", "--fingerprint", a, a},
		{"diff", "--keys-only", "--shape", a, a},
		{"diff", "-f", "yaml", a, a},
	} {
		if _, err := run(t, args...); err == nil {
			t.Errorf("run(%v) = nil error", args)
		}
	}
}

func sortedLines(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func TestDiffCommandNoDifferences(t *testing.T) {
	a := writeTemp(t, "diffc.env", "A=1\n")
	b := writeTemp(t, "diffd.env", "A=1\n")
	out, err := run(t, "diff", a, b)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if out != "" {
		t.Errorf("diff output = %q; want empty for identical files", out)
	}
}

// TestDiffExitCodeOnDifference pins the fix that replaced a direct
// os.Exit(1) call inside RunE with a typed error carrying the exit code. The
// old code was impossible to exercise from this test: os.Exit tears down the
// whole test binary, not just the command under test. run() now observes a
// returned error instead of the process disappearing.
func TestDiffExitCodeOnDifference(t *testing.T) {
	a := writeTemp(t, "diffe.env", "A=1\n")
	b := writeTemp(t, "difff.env", "A=2\n")
	_, err := run(t, "diff", "--exit-code", a, b)
	if err == nil {
		t.Fatal("diff --exit-code expected a non-nil error when files differ")
	}
	ec, ok := err.(interface{ ExitCode() int })
	if !ok {
		t.Fatalf("diff --exit-code error %v (%T) does not implement ExitCode()", err, err)
	}
	if got := ec.ExitCode(); got != 1 {
		t.Errorf("ExitCode() = %d; want 1", got)
	}
	if err.Error() != "" {
		t.Errorf("Error() = %q; want empty so main does not print a redundant line", err.Error())
	}
}

func TestDiffExitCodeNoDifference(t *testing.T) {
	a := writeTemp(t, "diffg.env", "A=1\n")
	b := writeTemp(t, "diffh.env", "A=1\n")
	if _, err := run(t, "diff", "--exit-code", a, b); err != nil {
		t.Fatalf("diff --exit-code with no differences: %v", err)
	}
}
