package cmd

import (
	"strings"
	"testing"
)

func TestDiffCommand(t *testing.T) {
	a := writeTemp(t, "diffa.env", "SAME=1\nCHANGED=old\nGONE=x\n")
	b := writeTemp(t, "diffb.env", "SAME=1\nCHANGED=new\nNEW=y\n")
	out, err := run(t, "diff", a, b)
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
