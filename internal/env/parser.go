package env

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Parse reads a dotenv document from r. It understands blank lines, comments
// beginning with '#', an optional leading "export ", and single- or
// double-quoted values. Double-quoted values support the common backslash
// escapes (\n, \r, \t, \\, \"). Unquoted values are trimmed of surrounding
// whitespace and may carry an inline comment introduced by " #".
func Parse(r io.Reader) (*Doc, error) {
	doc := New()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimRight(sc.Text(), "\r\n")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		trimmed = strings.TrimPrefix(trimmed, "export ")
		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			return nil, fmt.Errorf("line %d: missing '=' in %q", line, raw)
		}
		key := strings.TrimSpace(trimmed[:eq])
		if err := validateKey(key); err != nil {
			return nil, fmt.Errorf("line %d: %v", line, err)
		}
		value, err := parseValue(strings.TrimLeft(trimmed[eq+1:], " \t"))
		if err != nil {
			return nil, fmt.Errorf("line %d: %v", line, err)
		}
		doc.Set(key, value)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return doc, nil
}

// ParseFile reads and parses the dotenv file at path.
func ParseFile(path string) (*Doc, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("empty key")
	}
	for i, r := range key {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return fmt.Errorf("invalid character %q in key %q", r, key)
		}
	}
	return nil
}

func parseValue(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	switch s[0] {
	case '"':
		return parseDoubleQuoted(s)
	case '\'':
		return parseSingleQuoted(s)
	default:
		if i := strings.Index(s, " #"); i >= 0 {
			s = s[:i]
		}
		return strings.TrimRight(s, " \t"), nil
	}
}

func parseSingleQuoted(s string) (string, error) {
	end := strings.IndexByte(s[1:], '\'')
	if end < 0 {
		return "", fmt.Errorf("unterminated single-quoted value")
	}
	return s[1 : 1+end], nil
}

func parseDoubleQuoted(s string) (string, error) {
	var b strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i+1])
			}
			i += 2
			continue
		}
		if c == '"' {
			return b.String(), nil
		}
		b.WriteByte(c)
		i++
	}
	return "", fmt.Errorf("unterminated double-quoted value")
}
