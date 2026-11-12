package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SchemaFormats lists the names accepted by WriteSchema.
var SchemaFormats = []string{"markdown", "jsonschema"}

// WriteSchema documents s in format: a Markdown table or a JSON Schema
// (draft 2020-12) document.
//
// Defaults of keys that look secret (see IsSecretKey with the built-in
// matchers) are left out of both formats. An example file is meant to hold
// placeholders, but a real credential committed there should not also be
// copied into a README or an editor schema.
func WriteSchema(w io.Writer, s *Schema, format string) error {
	switch format {
	case "markdown", "md":
		return writeSchemaMarkdown(w, s)
	case "jsonschema", "json-schema":
		return writeJSONSchema(w, s)
	default:
		return fmt.Errorf("unknown format %q (want %s)", format, strings.Join(SchemaFormats, "|"))
	}
}

// documentedDefault returns the default to publish for spec, or "" for a
// secret-looking key.
func documentedDefault(spec Spec) string {
	if IsSecretKey(spec.Key, nil) {
		return ""
	}
	return spec.Default
}

// constraints describes @min, @max and @pattern for the Markdown table.
func constraints(spec Spec) []string {
	var out []string
	unit := " characters"
	if spec.Numeric() {
		unit = ""
	}
	if spec.Min != nil {
		out = append(out, fmt.Sprintf("min %d%s", *spec.Min, unit))
	}
	if spec.Max != nil {
		out = append(out, fmt.Sprintf("max %d%s", *spec.Max, unit))
	}
	if spec.Pattern != nil {
		out = append(out, "pattern "+mdCode("/"+spec.Pattern.String()+"/"))
	}
	return out
}

func writeSchemaMarkdown(w io.Writer, s *Schema) error {
	var b bytes.Buffer
	b.WriteString("| Name | Type | Required | Default | Description |\n")
	b.WriteString("|------|------|----------|---------|-------------|\n")
	for _, spec := range s.specs {
		typ := mdCode(spec.TypeLabel())
		if c := constraints(spec); len(c) > 0 {
			typ += ", " + strings.Join(c, ", ")
		}
		req := "no"
		if spec.Required {
			req = "yes"
		}
		def := ""
		if d := documentedDefault(spec); d != "" {
			def = mdCode(d)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			mdCode(spec.Key), typ, req, def, mdCell(spec.Description))
	}
	_, err := w.Write(b.Bytes())
	return err
}

// mdCell makes text safe inside a GFM table cell: a '|' would end the cell
// and a newline would end the row.
func mdCell(s string) string {
	s = strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
	return strings.ReplaceAll(s, "|", `\|`)
}

// mdCode wraps s in a code span long enough not to be closed by a backtick
// inside s. GFM tables still need '|' escaped inside code spans.
func mdCode(s string) string {
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	pad := ""
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		pad = " "
	}
	return fence + pad + mdCell(s) + pad + fence
}

// jsonSchemaProperty is one property of the generated JSON Schema. Every
// dotenv value is a string, so each property is "type": "string" and the
// declared type becomes a pattern or a format on that string.
type jsonSchemaProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     *string  `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Format      string   `json:"format,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	AllOf       []any    `json:"allOf,omitempty"`
	MinLength   *int64   `json:"minLength,omitempty"`
	MaxLength   *int64   `json:"maxLength,omitempty"`
}

// Patterns for the types JSON Schema cannot name directly. They mirror what
// check accepts; numeric @min/@max on int and port have no JSON Schema
// equivalent for a string and are only enforced by check.
const (
	intPattern  = `^[+-]?[0-9]+$`
	portPattern = `^0*([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$`
	// time.ParseDuration: an optional sign, then "0" or one or more
	// number+unit pairs.
	durationPattern = `^[+-]?(0|(([0-9]+(\.[0-9]*)?|\.[0-9]+)(ns|us|µs|μs|ms|s|m|h))+)$`
)

// boolPattern matches the bool spellings case-insensitively without the (?i)
// flag, which JSON Schema's ECMA-262 regex dialect does not support.
func boolPattern() string {
	words := make([]string, len(boolWords))
	for i, w := range boolWords {
		var b strings.Builder
		for _, c := range w {
			if c >= 'a' && c <= 'z' {
				fmt.Fprintf(&b, "[%c%c]", c, c-'a'+'A')
			} else {
				b.WriteRune(c)
			}
		}
		words[i] = b.String()
	}
	return "^(" + strings.Join(words, "|") + ")$"
}

func jsonSchemaFor(spec Spec) jsonSchemaProperty {
	p := jsonSchemaProperty{Type: "string", Description: spec.Description}
	if d := documentedDefault(spec); d != "" {
		p.Default = &d
	}
	var typePattern string
	switch spec.Type {
	case TypeInt:
		typePattern = intPattern
	case TypePort:
		typePattern = portPattern
	case TypeDuration:
		typePattern = durationPattern
	case TypeBool:
		typePattern = boolPattern()
	case TypeURL:
		p.Format = "uri"
	case TypeEmail:
		p.Format = "email"
	case TypeEnum:
		p.Enum = spec.Enum
	}
	// A string can carry one "pattern"; when the type and @pattern both need
	// one, the second goes into allOf so neither is lost.
	switch {
	case typePattern != "" && spec.Pattern != nil:
		p.Pattern = typePattern
		p.AllOf = []any{map[string]string{"pattern": spec.Pattern.String()}}
	case typePattern != "":
		p.Pattern = typePattern
	case spec.Pattern != nil:
		p.Pattern = spec.Pattern.String()
	}
	if !spec.Numeric() {
		p.MinLength, p.MaxLength = spec.Min, spec.Max
	}
	return p
}

func writeJSONSchema(w io.Writer, s *Schema) error {
	// Properties are written by hand to keep the example file's order; a Go
	// map would come out sorted.
	var props bytes.Buffer
	required := []string{}
	props.WriteString("{")
	for i, spec := range s.specs {
		if i > 0 {
			props.WriteString(",")
		}
		key, err := marshalCompact(spec.Key)
		if err != nil {
			return err
		}
		val, err := marshalCompact(jsonSchemaFor(spec))
		if err != nil {
			return err
		}
		props.Write(key)
		props.WriteString(":")
		props.Write(val)
		if spec.Required {
			required = append(required, spec.Key)
		}
	}
	props.WriteString("}")
	doc := struct {
		Schema     string          `json:"$schema"`
		Type       string          `json:"type"`
		Properties json.RawMessage `json:"properties"`
		Required   []string        `json:"required"`
	}{
		Schema:     "https://json-schema.org/draft/2020-12/schema",
		Type:       "object",
		Properties: props.Bytes(),
		Required:   required,
	}
	return writeJSON(w, doc)
}
