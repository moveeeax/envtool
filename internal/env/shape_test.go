package env

import (
	"reflect"
	"testing"
)

func TestFingerprint(t *testing.T) {
	a := Fingerprint([]byte("salt"), "value")
	if len(a) != FingerprintLen {
		t.Fatalf("len = %d; want %d", len(a), FingerprintLen)
	}
	if b := Fingerprint([]byte("salt"), "value"); a != b {
		t.Errorf("same salt and value: %s != %s", a, b)
	}
	if b := Fingerprint([]byte("salt"), "other"); a == b {
		t.Errorf("different values share fingerprint %s", a)
	}
	if b := Fingerprint([]byte("pepper"), "value"); a == b {
		t.Errorf("different salts share fingerprint %s", a)
	}
}

func TestShapeOf(t *testing.T) {
	cases := []struct {
		value string
		want  Shape
	}{
		{"", Shape{0, []string{}, "empty"}},
		{"8080", Shape{4, []string{"digit"}, "int"}},
		{"True", Shape{4, []string{"lower", "upper"}, "bool"}},
		{"123e4567-e89b-12d3-a456-426614174000", Shape{36, []string{"lower", "digit", "symbol"}, "uuid"}},
		{"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.c2ln", Shape{41, []string{"lower", "upper", "digit", "symbol"}, "jwt"}},
		{"https://db.local:5432/app", Shape{25, []string{"lower", "digit", "symbol"}, "url"}},
		{"deadbeefdeadbeef", Shape{16, []string{"lower"}, "hex"}},
		{"c2VjcmV0LXZhbHVlLTEyMw==", Shape{24, []string{"lower", "upper", "digit", "symbol"}, "base64"}},
		{"deadbeef", Shape{8, []string{"lower"}, "text"}},
		{"hello world é", Shape{13, []string{"lower", "space", "non-ascii"}, "text"}},
	}
	for _, c := range cases {
		if got := ShapeOf(c.value); !reflect.DeepEqual(got, c.want) {
			t.Errorf("ShapeOf(%q) = %+v; want %+v", c.value, got, c.want)
		}
	}
}

func TestShapeString(t *testing.T) {
	if got := ShapeOf("ab1").String(); got != "len=3 chars=lower+digit kind=text" {
		t.Errorf("String = %q", got)
	}
	if got := ShapeOf("").String(); got != "len=0 chars=none kind=empty" {
		t.Errorf("String = %q", got)
	}
}
