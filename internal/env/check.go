package env

import (
	"fmt"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// IssueKind classifies a check finding.
type IssueKind string

const (
	// IssueMissing marks a required key that is absent or empty.
	IssueMissing IssueKind = "missing"
	// IssueMismatch marks a value that fails its declared type or constraint.
	IssueMismatch IssueKind = "mismatch"
	// IssueExtra marks a key that the schema does not declare.
	IssueExtra IssueKind = "extra"
)

// Issue is a single check finding. Line is the line in the example file that
// declares Key, or 0 for an extra key, which the example does not declare.
type Issue struct {
	Key     string    `json:"key"`
	Kind    IssueKind `json:"kind"`
	Message string    `json:"message"`
	Line    int       `json:"line,omitempty"`
	// Fails reports whether this issue fails the check: always for missing and
	// mismatch, for extra only under Strict.
	Fails bool `json:"fails"`
}

// CheckOptions tunes Check.
type CheckOptions struct {
	// Strict makes extra keys fail the check.
	Strict bool
	// IgnoreExtra drops extra keys from the result entirely. It is used for
	// the process environment, where PATH, HOME and friends are always extra.
	IgnoreExtra bool
	// ShowValues puts the offending value into mismatch messages. Off by
	// default: a value that fails its type is often a secret pasted into the
	// wrong variable, and check output lands in CI logs.
	ShowValues bool
}

// CheckResult is the outcome of Check.
type CheckResult struct {
	Issues []Issue `json:"issues"`
	OK     bool    `json:"ok"`
}

// Failures counts the issues that fail the check.
func (r CheckResult) Failures() int {
	n := 0
	for _, is := range r.Issues {
		if is.Fails {
			n++
		}
	}
	return n
}

// Check validates doc against schema. Issues come in schema order, followed by
// extra keys in document order. An empty value counts as missing for a
// required key and passes for an optional one.
func Check(schema *Schema, doc *Doc, opts CheckOptions) CheckResult {
	res := CheckResult{Issues: []Issue{}, OK: true}
	add := func(is Issue) {
		if is.Fails {
			res.OK = false
		}
		res.Issues = append(res.Issues, is)
	}
	for _, spec := range schema.specs {
		value, ok := doc.Get(spec.Key)
		switch {
		case !ok:
			if spec.Required {
				add(Issue{Key: spec.Key, Kind: IssueMissing, Message: "required key is missing", Line: spec.Line, Fails: true})
			}
		case value == "":
			if spec.Required {
				add(Issue{Key: spec.Key, Kind: IssueMissing, Message: "required key is present but empty", Line: spec.Line, Fails: true})
			}
		default:
			if msg, ok := checkValue(value, spec, opts.ShowValues); !ok {
				add(Issue{Key: spec.Key, Kind: IssueMismatch, Message: msg, Line: spec.Line, Fails: true})
			}
		}
	}
	if !opts.IgnoreExtra {
		for _, e := range doc.entries {
			if _, known := schema.index[e.Key]; !known {
				add(Issue{Key: e.Key, Kind: IssueExtra, Message: "key is not declared in the example", Fails: opts.Strict})
			}
		}
	}
	return res
}

// checkValue validates a non-empty value against spec: the type first, then
// @min/@max, then @pattern.
func checkValue(value string, spec Spec, showValues bool) (string, bool) {
	got := describeValue(value, showValues)
	if msg := checkType(value, spec); msg != "" {
		return fmt.Sprintf("expected %s, got %s", msg, got), false
	}
	if spec.Numeric() {
		n, _ := strconv.ParseInt(value, 10, 64) // checkType has accepted it
		if spec.Min != nil && n < *spec.Min {
			return fmt.Sprintf("expected %s >= %d, got %s", spec.Type, *spec.Min, got), false
		}
		if spec.Max != nil && n > *spec.Max {
			return fmt.Sprintf("expected %s <= %d, got %s", spec.Type, *spec.Max, got), false
		}
	} else {
		n := int64(utf8.RuneCountInString(value))
		if spec.Min != nil && n < *spec.Min {
			return fmt.Sprintf("expected at least %d characters, got %s", *spec.Min, got), false
		}
		if spec.Max != nil && n > *spec.Max {
			return fmt.Sprintf("expected at most %d characters, got %s", *spec.Max, got), false
		}
	}
	if spec.Pattern != nil && !spec.Pattern.MatchString(value) {
		// The pattern comes from the committed example file, so it is safe to
		// print; the value is not.
		return fmt.Sprintf("expected a value matching /%s/, got %s", spec.Pattern, got), false
	}
	return "", true
}

// checkType returns what was expected when value does not parse as spec's
// type, or "" when it does.
func checkType(value string, spec Spec) string {
	switch spec.Type {
	case TypeInt:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return "int"
		}
	case TypePort:
		if !isPort(value) {
			return "port (1-65535)"
		}
	case TypeBool:
		if !isBool(value) {
			return "bool"
		}
	case TypeURL:
		if !isURL(value) {
			return "url"
		}
	case TypeDuration:
		if _, err := time.ParseDuration(value); err != nil {
			return "duration (such as 30s or 1h30m)"
		}
	case TypeEmail:
		if !isEmail(value) {
			return "email"
		}
	case TypeEnum:
		// The members come from the committed example file, so they are safe
		// to print.
		if !inList(value, spec.Enum) {
			return fmt.Sprintf("one of [%s]", strings.Join(spec.Enum, ", "))
		}
	}
	return ""
}

// describeValue renders an offending value for a message. Unless values were
// asked for, only the length is disclosed.
func describeValue(value string, showValues bool) string {
	if showValues {
		return strconv.Quote(value)
	}
	return fmt.Sprintf("a %d-character value", utf8.RuneCountInString(value))
}

// isPort accepts decimal digits only (no sign, no spaces) in 1..65535.
func isPort(v string) bool {
	for _, c := range v {
		if c < '0' || c > '9' {
			return false
		}
	}
	n, err := strconv.ParseUint(v, 10, 16)
	return err == nil && n >= 1
}

// boolWords are the spellings TypeBool accepts, compared case-insensitively.
var boolWords = []string{"true", "false", "1", "0", "yes", "no", "on", "off"}

func isBool(v string) bool {
	return inList(strings.ToLower(v), boolWords)
}

func isURL(v string) bool {
	u, err := url.Parse(v)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// isEmail accepts a bare address such as ops@example.com. net/mail also
// accepts "Name <ops@example.com>"; that form is rejected because a program
// reading the variable almost always expects the address alone.
func isEmail(v string) bool {
	a, err := mail.ParseAddress(v)
	return err == nil && a.Name == "" && a.Address == v
}

func inList(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
