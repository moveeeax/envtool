package env

import (
	"fmt"
	"sort"
	"strings"
)

// Rule describes a validation constraint on a single key.
type Rule struct {
	Key      string
	Required bool
	NonEmpty bool
}

// Violation is a single failed rule.
type Violation struct {
	Key     string
	Message string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: %s", v.Key, v.Message)
}

// Validate checks doc against rules and returns any violations sorted by key.
func Validate(doc *Doc, rules []Rule) []Violation {
	var out []Violation
	for _, r := range rules {
		value, ok := doc.Get(r.Key)
		if r.Required && !ok {
			out = append(out, Violation{Key: r.Key, Message: "required key is missing"})
			continue
		}
		if r.NonEmpty && ok && strings.TrimSpace(value) == "" {
			out = append(out, Violation{Key: r.Key, Message: "value must not be empty"})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// RulesFromKeys builds required, non-empty rules from a list of key names.
func RulesFromKeys(keys []string) []Rule {
	rules := make([]Rule, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		rules = append(rules, Rule{Key: k, Required: true, NonEmpty: true})
	}
	return rules
}
