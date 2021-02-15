package env

import (
	"reflect"
	"testing"
)

func TestSortedCopy(t *testing.T) {
	d := docOf("ZED", "1", "ALPHA", "2", "MID", "3")
	got := d.SortedCopy()
	if !reflect.DeepEqual(got.Keys(), []string{"ALPHA", "MID", "ZED"}) {
		t.Fatalf("SortedCopy keys = %v; want [ALPHA MID ZED]", got.Keys())
	}
	// original is unchanged
	if !reflect.DeepEqual(d.Keys(), []string{"ZED", "ALPHA", "MID"}) {
		t.Errorf("original mutated: %v", d.Keys())
	}
	if v, _ := got.Get("MID"); v != "3" {
		t.Errorf("value lost in sort: MID=%q", v)
	}
}
