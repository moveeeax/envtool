package env

import (
	"encoding/json"
	"fmt"
	"io"
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

func exportJSON(w io.Writer, doc *Doc) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	m := make(map[string]string, doc.Len())
	for _, e := range doc.entries {
		m[e.Key] = e.Value
	}
	return enc.Encode(m)
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
	return strings.ContainsAny(v, " \t\n\"'#")
}

func escapeDouble(v string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n", "\t", "\\t", "\r", "\\r")
	return r.Replace(v)
}

func yamlScalar(v string) string {
	if v == "" {
		return `""`
	}
	if strings.ContainsAny(v, ":#{}[],&*!|>'\"%@` \t\n") {
		return `"` + escapeDouble(v) + `"`
	}
	return v
}
