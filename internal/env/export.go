package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Format enumerates the supported export encodings.
type Format string

const (
	// FormatDotenv writes KEY=VALUE lines with minimal quoting.
	FormatDotenv Format = "dotenv"
	// FormatShell writes `export KEY='VALUE'` lines safe for `eval`.
	FormatShell Format = "shell"
	// FormatJSON writes a single JSON object of string values.
	FormatJSON Format = "json"
	// FormatYAML writes a flat YAML mapping of string values.
	FormatYAML Format = "yaml"
)

// ParseFormat resolves a format name, defaulting sensibly on aliases.
func ParseFormat(name string) (Format, error) {
	switch strings.ToLower(name) {
	case "", "dotenv", "env":
		return FormatDotenv, nil
	case "shell", "sh", "bash":
		return FormatShell, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unknown format %q", name)
	}
}

// Export writes doc to w in the requested format.
func Export(w io.Writer, doc *Doc, format Format) error {
	switch format {
	case FormatDotenv:
		return exportDotenv(w, doc)
	case FormatShell:
		return exportShell(w, doc)
	case FormatJSON:
		return exportJSON(w, doc)
	case FormatYAML:
		return exportYAML(w, doc)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func exportDotenv(w io.Writer, doc *Doc) error {
	for _, e := range doc.entries {
		v := e.Value
		if needsQuote(v) {
			v = "\"" + escapeDouble(v) + "\""
		}
		if _, err := fmt.Fprintf(w, "%s=%s\n", e.Key, v); err != nil {
			return err
		}
	}
	return nil
}

func exportShell(w io.Writer, doc *Doc) error {
	for _, e := range doc.entries {
		quoted := "'" + strings.ReplaceAll(e.Value, "'", `'\''`) + "'"
		if _, err := fmt.Fprintf(w, "export %s=%s\n", e.Key, quoted); err != nil {
			return err
		}
	}
	return nil
}

// exportJSON writes the entries in document order. Encoding a map instead would
// re-sort the keys alphabetically, which loses the file's own ordering and makes
// the --sort flag a no-op for this format.
func exportJSON(w io.Writer, doc *Doc) error {
	var b bytes.Buffer
	if len(doc.entries) == 0 {
		b.WriteString("{}\n")
	} else {
		b.WriteString("{\n")
		for i, e := range doc.entries {
			key, err := json.Marshal(e.Key)
			if err != nil {
				return err
			}
			value, err := json.Marshal(e.Value)
			if err != nil {
				return err
			}
			b.WriteString("  ")
			b.Write(key)
			b.WriteString(": ")
			b.Write(value)
			if i < len(doc.entries)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString("}\n")
	}
	_, err := w.Write(b.Bytes())
	return err
}

func exportYAML(w io.Writer, doc *Doc) error {
	for _, e := range doc.entries {
		if _, err := fmt.Fprintf(w, "%s: %s\n", e.Key, yamlScalar(e.Value)); err != nil {
			return err
		}
	}
	return nil
}

func needsQuote(v string) bool {
	if v == "" {
		return false
	}
	// \r must be quoted too: an unquoted value ending in a carriage return is
	// silently swallowed by the line scanner when the output is read back.
	return strings.ContainsAny(v, " \t\n\r\"'#")
}

func escapeDouble(v string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n", "\t", "\\t", "\r", "\\r")
	return r.Replace(v)
}

// yamlPlainWords are values that a YAML loader resolves to a boolean or null
// rather than to a string. YAML 1.1 loaders (still the default in many
// ecosystems) treat all of these as non-strings, so they must be quoted.
var yamlPlainWords = map[string]bool{
	"y": true, "n": true, "yes": true, "no": true,
	"true": true, "false": true, "on": true, "off": true,
	"null": true, "~": true, "": true,
	".inf": true, "+.inf": true, "-.inf": true, ".nan": true,
}

// yamlScalar renders v as a YAML scalar that always loads back as a string.
func yamlScalar(v string) string {
	if needsYAMLQuote(v) {
		return `"` + escapeDouble(v) + `"`
	}
	return v
}

func needsYAMLQuote(v string) bool {
	// Structural characters, quoting indicators and any whitespace.
	if strings.ContainsAny(v, ":#{}[],&*!|>'\"%@`") || strings.ContainsAny(v, " \t\n\r") {
		return true
	}
	// Indicators that are only special in first position.
	if strings.HasPrefix(v, "-") || strings.HasPrefix(v, "?") {
		return true
	}
	if yamlPlainWords[strings.ToLower(v)] {
		return true
	}
	// Anything a loader would resolve to a number rather than a string. This
	// matters in practice: a Kubernetes ConfigMap rejects non-string data, and
	// a version like "1.10" would otherwise round-trip as the float 1.1.
	if looksNumeric(v) {
		return true
	}
	// Unquoted YYYY-MM-DD is a timestamp to a YAML 1.1 loader.
	return looksLikeDate(v)
}

func looksNumeric(v string) bool {
	if _, err := strconv.ParseInt(v, 0, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseUint(v, 0, 64); err == nil {
		return true
	}
	_, err := strconv.ParseFloat(v, 64)
	return err == nil
}

func looksLikeDate(v string) bool {
	if len(v) != 10 || v[4] != '-' || v[7] != '-' {
		return false
	}
	for i, c := range v {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
