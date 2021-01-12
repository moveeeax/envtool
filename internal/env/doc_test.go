package env

import (
	"reflect"
	"testing"
)

func TestDocSetOrderAndUpdate(t *testing.T) {
	d := New()
	d.Set("A", "1")
	d.Set("B", "2")
	d.Set("A", "3")
	if got := d.Keys(); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Fatalf("Keys = %v; want [A B]", got)
	}
	if v, _ := d.Get("A"); v != "3" {
		t.Errorf("Get(A) = %q; want 3", v)
	}
}

func TestDocDelete(t *testing.T) {
	d := New()
	d.Set("A", "1")
	d.Set("B", "2")
	d.Set("C", "3")
	if !d.Delete("B") {
		t.Fatal("Delete(B) = false")
	}
	if d.Delete("B") {
		t.Fatal("Delete(B) twice = true")
	}
	if got := d.Keys(); !reflect.DeepEqual(got, []string{"A", "C"}) {
		t.Fatalf("Keys = %v; want [A C]", got)
	}
	if v, _ := d.Get("C"); v != "3" {
		t.Errorf("Get(C) after delete = %q; want 3", v)
	}
}
