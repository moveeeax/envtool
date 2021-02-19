package cmd

import (
	"strings"
	"testing"
)

func TestKeysCommandSorted(t *testing.T) {
	f := writeTemp(t, "k.env", "ZED=1\nALPHA=2\nMID=3\n")
	out, err := run(t, "keys", "--sort", f)
	if err != nil {
		t.Fatalf("keys: %v", err)
	}
	got := strings.Fields(out)
	want := []string{"ALPHA", "MID", "ZED"}
	if len(got) != len(want) {
		t.Fatalf("keys = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("keys[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestGetCommand(t *testing.T) {
	f := writeTemp(t, "g.env", "HOST=db.local\nPORT=5432\n")
	out, err := run(t, "get", "PORT", f)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimSpace(out) != "5432" {
		t.Errorf("get PORT = %q; want 5432", out)
	}
	if _, err := run(t, "get", "MISSING", f); err == nil {
		t.Error("get MISSING expected error")
	}
}
