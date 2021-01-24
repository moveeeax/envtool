package env

import "testing"

func TestValidate(t *testing.T) {
	doc := docOf("PRESENT", "x", "BLANK", "  ")
	rules := RulesFromKeys([]string{"PRESENT", "BLANK", "MISSING"})
	got := Validate(doc, rules)
	if len(got) != 2 {
		t.Fatalf("violations = %v; want 2", got)
	}
	if got[0].Key != "BLANK" || got[1].Key != "MISSING" {
		t.Errorf("violations not sorted by key: %v", got)
	}
}

func TestValidateOK(t *testing.T) {
	doc := docOf("A", "1", "B", "2")
	if got := Validate(doc, RulesFromKeys([]string{"A", "B"})); len(got) != 0 {
		t.Fatalf("violations = %v; want none", got)
	}
}
