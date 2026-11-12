package env

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Value types a schema can declare with @type=. Every value in a dotenv file
// is a string; the type only decides how check validates it.
const (
	TypeString   = "string"
	TypeInt      = "int"
	TypeBool     = "bool"
	TypeURL      = "url"
	TypeEnum     = "enum"
	TypePort     = "port"
	TypeDuration = "duration"
	TypeEmail    = "email"
)

var knownTypes = map[string]bool{
	TypeString: true, TypeInt: true, TypeBool: true, TypeURL: true,
	TypeEnum: true, TypePort: true, TypeDuration: true, TypeEmail: true,
}

// Spec is the contract an annotated example file declares for one key.
type Spec struct {
	Key string
	// Default is the value written in the example file.
	Default string
	// Line is the 1-based line of the assignment in the example file.
	Line     int
	Type     string
	Enum     []string
	Required bool
	// Pattern, when set, must match the value (unanchored, like JSON Schema's
	// "pattern"; write ^...$ to match the whole value).
	Pattern *regexp.Regexp
	// Min and Max bound the number for int and port, and the length in
	// characters for every other type.
	Min, Max *int64
	// Description is the comment text around the key with the annotations
	// removed.
	Description string
}

// Numeric reports whether Min and Max bound the value itself rather than its
// length.
func (s Spec) Numeric() bool {
	return s.Type == TypeInt || s.Type == TypePort
}

// Schema is the ordered set of specs parsed from an annotated example file.
type Schema struct {
	specs []Spec
	index map[string]int
}

// Specs returns the specs in file order.
func (s *Schema) Specs() []Spec {
	out := make([]Spec, len(s.specs))
	copy(out, s.specs)
	return out
}

// Get returns the spec for key and whether it was declared.
func (s *Schema) Get(key string) (Spec, bool) {
	i, ok := s.index[key]
	if !ok {
		return Spec{}, false
	}
	return s.specs[i], true
}

// ParseSchemaFile reads and parses the annotated example file at path.
func ParseSchemaFile(path string) (*Schema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseSchema(f)
}

// ParseSchema reads an annotated example file. The syntax of the assignments
// is the same as Parse; on top of that, comments carry annotations for the key
// on the next assignment line (standalone comment lines, reset by a blank
// line) or on the same line (a trailing " # ..." comment):
//
//	# @type=port @required
//	PORT=8080
//	APP_ENV=dev # @type=enum(dev,staging,prod)
//
// Annotations are @type=NAME, @type=enum(a,b), @enum=a,b, @required,
// @pattern=/regex/, @min=N and @max=N. A malformed annotation, an unknown
// type or a regex that does not compile is an error rather than a silent
// fallback to "string", so a typo cannot quietly disable a check.
func ParseSchema(r io.Reader) (*Schema, error) {
	s := &Schema{index: make(map[string]int)}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		pending  Spec
		descs    []string
		lastLine int // line of the last annotation seen, for error messages
	)
	reset := func() { pending, descs, lastLine = Spec{}, nil, 0 }

	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if line == 1 {
			text = strings.TrimPrefix(text, "\uFEFF")
		}
		trimmed := strings.TrimSpace(strings.TrimRight(text, "\r\n"))
		if trimmed == "" {
			reset()
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			desc, found, err := parseAnnotations(trimmed[1:], &pending)
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", line, err)
			}
			if found {
				lastLine = line
			}
			if desc != "" {
				descs = append(descs, desc)
			}
			continue
		}
		trimmed = trimExportPrefix(trimmed)
		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			return nil, fmt.Errorf("line %d: missing '=' separator", line)
		}
		key := strings.TrimSpace(trimmed[:eq])
		if err := validateKey(key); err != nil {
			return nil, fmt.Errorf("line %d: %v", line, err)
		}
		value, rest, err := parseValueRest(strings.TrimLeft(trimmed[eq+1:], " \t"))
		if err != nil {
			return nil, fmt.Errorf("line %d: %v", line, err)
		}
		spec := pending
		if rest = strings.TrimSpace(rest); strings.HasPrefix(rest, "#") {
			desc, found, err := parseAnnotations(rest[1:], &spec)
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", line, err)
			}
			if found {
				lastLine = line
			}
			if desc != "" {
				descs = append(descs, desc)
			}
		}
		spec.Key, spec.Default, spec.Line = key, value, line
		spec.Description = strings.Join(descs, " ")
		if err := finishSpec(&spec); err != nil {
			errLine := lastLine
			if errLine == 0 {
				errLine = line
			}
			return nil, fmt.Errorf("line %d: %s: %v", errLine, key, err)
		}
		if i, ok := s.index[key]; ok {
			s.specs[i] = spec // last declaration wins, as in Parse
		} else {
			s.index[key] = len(s.specs)
			s.specs = append(s.specs, spec)
		}
		reset()
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return s, nil
}

var (
	annWordRe  = regexp.MustCompile(`^@([A-Za-z]+)`)
	annTypeRe  = regexp.MustCompile(`^@type=([A-Za-z]+)(?:\(([^)]*)\))?`)
	annEnumRe  = regexp.MustCompile(`^@enum=([^\s,]+(?:\s*,\s*[^\s,]+)*)`)
	annBoundRe = regexp.MustCompile(`^@(?:min|max)=(\S+)`)
)

// parseAnnotations applies the annotations found in comment to spec and
// returns the remaining comment text, whitespace-collapsed, as a description.
// found reports whether any annotation was present. An '@' only starts an
// annotation at the beginning of the comment or after whitespace, so an email
// address in a comment stays text; an unknown @word is also kept as text.
func parseAnnotations(comment string, spec *Spec) (desc string, found bool, err error) {
	var text strings.Builder
	for i := 0; i < len(comment); {
		rest := comment[i:]
		if comment[i] != '@' || (i > 0 && !isSpace(comment[i-1])) {
			text.WriteByte(comment[i])
			i++
			continue
		}
		m := annWordRe.FindStringSubmatch(rest)
		if m == nil {
			text.WriteByte(comment[i])
			i++
			continue
		}
		name := strings.ToLower(m[1])
		var n int
		switch name {
		case "required":
			spec.Required = true
			n = len(m[0])
		case "type":
			tm := annTypeRe.FindStringSubmatch(rest)
			if tm == nil {
				return "", false, fmt.Errorf("@type needs a value, as in @type=int")
			}
			t := strings.ToLower(tm[1])
			if !knownTypes[t] {
				return "", false, fmt.Errorf("unknown type %q", tm[1])
			}
			spec.Type = t
			if tm[2] != "" {
				if t != TypeEnum {
					return "", false, fmt.Errorf("only enum takes a member list, got @type=%s(...)", tm[1])
				}
				spec.Enum = splitList(tm[2])
			}
			n = len(tm[0])
		case "enum":
			em := annEnumRe.FindStringSubmatch(rest)
			if em == nil {
				return "", false, fmt.Errorf("@enum needs a comma-separated list, as in @enum=a,b")
			}
			spec.Enum = splitList(em[1])
			n = len(em[0])
		case "pattern":
			re, used, perr := parsePatternAnnotation(rest)
			if perr != nil {
				return "", false, perr
			}
			spec.Pattern = re
			n = used
		case "min", "max":
			bm := annBoundRe.FindStringSubmatch(rest)
			if bm == nil {
				return "", false, fmt.Errorf("@%s needs an integer, as in @%s=1", name, name)
			}
			v, perr := strconv.ParseInt(bm[1], 10, 64)
			if perr != nil {
				return "", false, fmt.Errorf("@%s=%s is not an integer", name, bm[1])
			}
			if name == "min" {
				spec.Min = &v
			} else {
				spec.Max = &v
			}
			n = len(bm[0])
		default:
			text.WriteString(m[0])
			i += len(m[0])
			continue
		}
		found = true
		i += n
	}
	return strings.Join(strings.Fields(text.String()), " "), found, nil
}

// parsePatternAnnotation parses "@pattern=/regex/" at the start of s and
// returns the compiled regex and the number of bytes consumed. The regex ends
// at the first '/' that is not escaped and is followed by whitespace or the
// end of the comment, so slashes inside the regex work as long as they are
// written \/ or are not followed by a space.
func parsePatternAnnotation(s string) (*regexp.Regexp, int, error) {
	const prefix = "@pattern=/"
	if !strings.HasPrefix(s, prefix) {
		return nil, 0, fmt.Errorf("@pattern needs a regex between slashes, as in @pattern=/^[a-z]+$/")
	}
	for i := len(prefix); i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '/':
			if i+1 == len(s) || isSpace(s[i+1]) {
				src := s[len(prefix):i]
				if src == "" {
					return nil, 0, fmt.Errorf("@pattern is empty")
				}
				re, err := regexp.Compile(src)
				if err != nil {
					return nil, 0, fmt.Errorf("@pattern does not compile: %v", err)
				}
				return re, i + 1, nil
			}
		}
	}
	return nil, 0, fmt.Errorf("@pattern is missing its closing '/'")
}

// finishSpec fills in the default type and rejects combinations that cannot
// be checked.
func finishSpec(s *Spec) error {
	if s.Type == "" {
		if len(s.Enum) > 0 {
			s.Type = TypeEnum
		} else {
			s.Type = TypeString
		}
	}
	if s.Type == TypeEnum && len(s.Enum) == 0 {
		return fmt.Errorf("enum declares no members")
	}
	if s.Type != TypeEnum && len(s.Enum) > 0 {
		return fmt.Errorf("@enum given for type %s", s.Type)
	}
	if s.Min != nil && s.Max != nil && *s.Min > *s.Max {
		return fmt.Errorf("@min=%d is greater than @max=%d", *s.Min, *s.Max)
	}
	if !s.Numeric() && ((s.Min != nil && *s.Min < 0) || (s.Max != nil && *s.Max < 0)) {
		return fmt.Errorf("@min and @max bound the length for type %s and cannot be negative", s.Type)
	}
	return nil
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

// TypeLabel renders the type with its member list for enums, for example
// "enum(dev, staging, prod)".
func (s Spec) TypeLabel() string {
	if s.Type == TypeEnum {
		return fmt.Sprintf("enum(%s)", strings.Join(s.Enum, ", "))
	}
	return s.Type
}
