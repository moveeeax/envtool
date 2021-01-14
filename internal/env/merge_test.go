package env

import (
	"reflect"
	"testing"
)

func docOf(pairs ...string) *Doc {
	d := New()
	for i := 0; i+1 < len(pairs); i += 2 {
		d.Set(pairs[i], pairs[i+1])
	}
	return d
}

func TestMergeLaterWins(t *testing.T) {
	base := docOf("A", "1", "B", "2")
	over := docOf("B", "20", "C", "3")
	got := Merge(base, over)
	if v, _ := got.Get("B"); v != "20" {
		t.Errorf("B = %q; want 20", v)
	}
	if !reflect.DeepEqual(got.Keys(), []string{"A", "B", "C"}) {
		t.Errorf("Keys = %v; want [A B C]", got.Keys())
	}
}

func TestMergeIgnoresNil(t *testing.T) {
	got := Merge(nil, docOf("A", "1"), nil)
	if got.Len() != 1 {
		t.Fatalf("Len = %d; want 1", got.Len())
	}
}
