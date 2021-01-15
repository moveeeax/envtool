package env

import "testing"

func TestDiff(t *testing.T) {
	left := docOf("SAME", "x", "CHANGED", "old", "GONE", "y")
	right := docOf("SAME", "x", "CHANGED", "new", "NEW", "z")
	got := Diff(left, right)
	if len(got) != 3 {
		t.Fatalf("len(changes) = %d; want 3", len(got))
	}
	byKey := map[string]Change{}
	for _, c := range got {
		byKey[c.Key] = c
	}
	if c := byKey["CHANGED"]; c.Kind != Changed || c.Left != "old" || c.Right != "new" {
		t.Errorf("CHANGED = %+v", c)
	}
	if c := byKey["GONE"]; c.Kind != Removed || c.Left != "y" {
		t.Errorf("GONE = %+v", c)
	}
	if c := byKey["NEW"]; c.Kind != Added || c.Right != "z" {
		t.Errorf("NEW = %+v", c)
	}
}

func TestDiffEqual(t *testing.T) {
	d := docOf("A", "1", "B", "2")
	if got := Diff(d, d); len(got) != 0 {
		t.Fatalf("Diff of equal docs = %v; want empty", got)
	}
}
