package env

import (
	"reflect"
	"strings"
	"testing"
)

func mustSchema(t *testing.T, s string) *Schema {
	t.Helper()
	schema, err := ParseSchema(strings.NewReader(s))
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}
	return schema
}

func mustSpec(t *testing.T, s *Schema, key string) Spec {
	t.Helper()
	spec, ok := s.Get(key)
	if !ok {
		t.Fatalf("spec %s not found", key)
	}
	return spec
}

func TestParseSchemaAnnotations(t *testing.T) {
	s := mustSchema(t, "\uFEFF# Listen port\n"+
		"# @type=port @required\n"+
		"PORT=8080\n"+
		"\n"+
		"# @type=enum(dev, staging,prod)\n"+
		"APP_ENV=dev\n"+
		"# @enum=a, b @required\n"+
		"LETTER=a\n"+
		"export TIMEOUT=30s # @type=duration\n"+
		`QUOTED="x # y" # @required @min=1 @max=8`+"\n"+
		"PLAIN=hi\n")

	port := mustSpec(t, s, "PORT")
	if port.Type != TypePort || !port.Required || port.Default != "8080" || port.Line != 3 {
		t.Errorf("PORT = %+v", port)
	}
	if port.Description != "Listen port" {
		t.Errorf("PORT description = %q; want %q", port.Description, "Listen port")
	}
	env := mustSpec(t, s, "APP_ENV")
	if env.Type != TypeEnum || !reflect.DeepEqual(env.Enum, []string{"dev", "staging", "prod"}) || env.Required {
		t.Errorf("APP_ENV = %+v", env)
	}
	letter := mustSpec(t, s, "LETTER")
	if letter.Type != TypeEnum || !reflect.DeepEqual(letter.Enum, []string{"a", "b"}) || !letter.Required {
		t.Errorf("LETTER = %+v", letter)
	}
	if tm := mustSpec(t, s, "TIMEOUT"); tm.Type != TypeDuration || tm.Default != "30s" {
		t.Errorf("TIMEOUT = %+v", tm)
	}
	q := mustSpec(t, s, "QUOTED")
	if q.Default != "x # y" || !q.Required || q.Min == nil || *q.Min != 1 || q.Max == nil || *q.Max != 8 {
		t.Errorf("QUOTED = %+v", q)
	}
	if p := mustSpec(t, s, "PLAIN"); p.Type != TypeString || p.Required {
		t.Errorf("PLAIN = %+v", p)
	}
	if got := len(s.Specs()); got != 6 {
		t.Errorf("len(Specs) = %d; want 6", got)
	}
}

// A blank line ends a comment block, so a section header separated from the
// next key by a blank line does not leak its annotations onto it.
func TestParseSchemaBlankLineResets(t *testing.T) {
	s := mustSchema(t, "# @required @type=int\n\nA=x\n")
	if a := mustSpec(t, s, "A"); a.Required || a.Type != TypeString || a.Description != "" {
		t.Errorf("A = %+v; want an unannotated string", a)
	}
}

func TestParseSchemaPattern(t *testing.T) {
	s := mustSchema(t, `# @pattern=/^[a-z]{2}-[a-z]+-\d$/ @required region
REGION=eu-west-1
# @pattern=/^a\/b$/
SLASH=a/b
# @pattern=/^[a-z ]+ x$/
SPACES=ab x
`)
	r := mustSpec(t, s, "REGION")
	if r.Pattern == nil || r.Pattern.String() != `^[a-z]{2}-[a-z]+-\d$` || !r.Required || r.Description != "region" {
		t.Errorf("REGION = %+v", r)
	}
	if sl := mustSpec(t, s, "SLASH"); sl.Pattern == nil || !sl.Pattern.MatchString("a/b") {
		t.Errorf("SLASH pattern = %v", sl.Pattern)
	}
	if sp := mustSpec(t, s, "SPACES"); sp.Pattern == nil || !sp.Pattern.MatchString("ab x") {
		t.Errorf("SPACES pattern = %v", sp.Pattern)
	}
}

// An '@' inside a word (an email address) or an unknown @word is comment
// text, not an annotation.
func TestParseSchemaDescriptionKeepsText(t *testing.T) {
	s := mustSchema(t, "# mail ops@example.com or ping @oncall\nA=1 # @type=int  counts   things\n")
	a := mustSpec(t, s, "A")
	if a.Type != TypeInt {
		t.Errorf("A type = %q; want int", a.Type)
	}
	if want := "mail ops@example.com or ping @oncall counts things"; a.Description != want {
		t.Errorf("A description = %q; want %q", a.Description, want)
	}
}

func TestParseSchemaLastDeclarationWins(t *testing.T) {
	s := mustSchema(t, "# @type=int\nA=1\n# @type=bool\nA=true\n")
	if a := mustSpec(t, s, "A"); a.Type != TypeBool || a.Line != 4 {
		t.Errorf("A = %+v; want the second declaration", a)
	}
	if got := len(s.Specs()); got != 1 {
		t.Errorf("len(Specs) = %d; want 1", got)
	}
}

// Every malformed annotation is an error with a line number. Falling back to
// "string" would let a typo switch a check off without anyone noticing.
func TestParseSchemaErrors(t *testing.T) {
	cases := map[string]string{
		"unknown type":      "# @type=integer\nA=1\n",
		"type no value":     "# @type\nA=1\n",
		"enum no members":   "# @type=enum\nA=1\n",
		"enum on int":       "# @type=int @enum=1,2\nA=1\n",
		"list on int":       "# @type=int(1,2)\nA=1\n",
		"bad regex":         "# @pattern=/a(/\nA=1\n",
		"unclosed pattern":  "# @pattern=/abc\nA=1\n",
		"empty pattern":     "# @pattern=//\nA=1\n",
		"min not int":       "# @min=x\nA=1\n",
		"min over max":      "# @type=int @min=5 @max=1\nA=1\n",
		"negative length":   "# @min=-1\nA=1\n",
		"bad inline":        "A=1 # @type=nope\n",
		"missing separator": "# @required\nNOEQUALS\n",
	}
	for name, in := range cases {
		_, err := ParseSchema(strings.NewReader(in))
		if err == nil {
			t.Errorf("%s: ParseSchema(%q) = nil error", name, in)
			continue
		}
		if !strings.HasPrefix(err.Error(), "line ") {
			t.Errorf("%s: error %q does not start with a line number", name, err)
		}
	}
}

func TestParseSchemaNegativeNumericBound(t *testing.T) {
	s := mustSchema(t, "# @type=int @min=-5 @max=5\nA=0\n")
	if a := mustSpec(t, s, "A"); a.Min == nil || *a.Min != -5 {
		t.Errorf("A.Min = %v; want -5", a.Min)
	}
}
