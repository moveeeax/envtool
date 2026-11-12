package env

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestSchemaMarkdown(t *testing.T) {
	s := mustSchema(t, "# Listen | port\n# @type=port @required\nPORT=8080\n"+
		"# @min=32\nAPI_TOKEN=changeme\n"+
		"# @pattern=/^(a|b)$/\nAB=a\n")
	var b bytes.Buffer
	if err := WriteSchema(&b, s, "markdown"); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		"| Name | Type | Required | Default | Description |",
		"| `PORT` | `port` | yes | `8080` | Listen \\| port |",
		"| `API_TOKEN` | `string`, min 32 characters | no |  |  |",
		"pattern `/^(a\\|b)$/`",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown =\n%s\nwant to contain %q", out, want)
		}
	}
	if strings.Contains(out, "changeme") {
		t.Errorf("markdown publishes the default of a secret-looking key:\n%s", out)
	}
}

func TestMDCodeBackticks(t *testing.T) {
	if got := mdCode("a`b"); got != "``a`b``" {
		t.Errorf("mdCode = %q", got)
	}
}

func TestJSONSchema(t *testing.T) {
	s := mustSchema(t, "# @type=port @required\nPORT=8080\n"+
		"# @type=enum(dev,prod)\nAPP_ENV=dev\n"+
		"# @type=int @pattern=/^[0-9]{3}$/\nCODE=123\n"+
		"# @min=2 @max=4 desc here\nNAME=ab\n"+
		"# @type=url\nURL=https://x.test\n"+
		"SECRET_KEY=s3cr3t\n")
	var b bytes.Buffer
	if err := WriteSchema(&b, s, "jsonschema"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "s3cr3t") {
		t.Errorf("JSON Schema publishes the default of a secret-looking key")
	}
	if i, j := strings.Index(b.String(), `"PORT"`), strings.Index(b.String(), `"APP_ENV"`); i < 0 || j < i {
		t.Errorf("properties are not in file order:\n%s", b.String())
	}
	var doc struct {
		Schema     string                    `json:"$schema"`
		Type       string                    `json:"type"`
		Required   []string                  `json:"required"`
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(b.Bytes(), &doc); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, b.String())
	}
	if doc.Schema != "https://json-schema.org/draft/2020-12/schema" || doc.Type != "object" {
		t.Errorf("header = %q %q", doc.Schema, doc.Type)
	}
	if strings.Join(doc.Required, ",") != "PORT" {
		t.Errorf("required = %v; want [PORT]", doc.Required)
	}
	p := doc.Properties
	if p["PORT"]["pattern"] != portPattern || p["PORT"]["default"] != "8080" {
		t.Errorf("PORT = %v", p["PORT"])
	}
	if enum, _ := p["APP_ENV"]["enum"].([]any); len(enum) != 2 {
		t.Errorf("APP_ENV = %v", p["APP_ENV"])
	}
	if allOf, _ := p["CODE"]["allOf"].([]any); p["CODE"]["pattern"] != intPattern || len(allOf) != 1 {
		t.Errorf("CODE = %v; want the int pattern plus @pattern in allOf", p["CODE"])
	}
	if p["NAME"]["minLength"] != 2.0 || p["NAME"]["maxLength"] != 4.0 || p["NAME"]["description"] != "desc here" {
		t.Errorf("NAME = %v", p["NAME"])
	}
	if p["URL"]["format"] != "uri" {
		t.Errorf("URL = %v", p["URL"])
	}
	for k, v := range p {
		if v["type"] != "string" {
			t.Errorf("%s type = %v; every dotenv value is a string", k, v["type"])
		}
	}
}

// The JSON Schema patterns stand in for check's type checks, so the two must
// agree on what is valid.
func TestJSONSchemaPatternsMatchCheck(t *testing.T) {
	cases := []struct {
		typ     string
		pattern string
		values  []string
	}{
		{TypeInt, intPattern, []string{"0", "42", "-7", "+3", "1.5", "x", "9223372036854775807"}},
		{TypePort, portPattern, []string{"1", "80", "0080", "65535", "65536", "0", "-1", "99999"}},
		{TypeDuration, durationPattern, []string{"30s", "1h30m", "1.5h", ".5s", "0", "-2m", "300ms", "2µs", "30", "s", "1d", ""}},
		{TypeBool, boolPattern(), []string{"true", "TRUE", "False", "yes", "No", "on", "OFF", "1", "0", "y", "maybe"}},
	}
	for _, c := range cases {
		re := regexp.MustCompile(c.pattern)
		for _, v := range c.values {
			byCheck := v != "" && checkType(v, Spec{Type: c.typ}) == ""
			if byPattern := re.MatchString(v); byPattern != byCheck {
				t.Errorf("%s %q: pattern says %v, check says %v", c.typ, v, byPattern, byCheck)
			}
		}
	}
}

func TestWriteSchemaUnknownFormat(t *testing.T) {
	if err := WriteSchema(&bytes.Buffer{}, mustSchema(t, "A=1\n"), "yaml"); err == nil {
		t.Error("WriteSchema(yaml) = nil error")
	}
}
