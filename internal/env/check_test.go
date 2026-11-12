package env

import (
	"strings"
	"testing"
)

const checkExample = `# @type=int @required
COUNT=1
# @type=enum(dev,prod) @required
APP_ENV=dev
# @type=url
API_URL=https://example.com
# @type=bool
DEBUG=false
# @type=port
PORT=8080
# @type=duration
TIMEOUT=30s
# @type=email
OPS=ops@example.com
# @pattern=/^[a-z]{2}-[a-z]+-\d$/
REGION=eu-west-1
# @min=4 @max=6
CODE=abcd
# @type=int @min=1 @max=10
WORKERS=2
OPTIONAL=hi
`

func mustDoc(t *testing.T, s string) *Doc {
	t.Helper()
	d, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

func hasIssue(res CheckResult, key string, kind IssueKind) bool {
	for _, is := range res.Issues {
		if is.Key == key && is.Kind == kind {
			return true
		}
	}
	return false
}

func TestCheckClean(t *testing.T) {
	doc := mustDoc(t, "COUNT=3\nAPP_ENV=prod\nAPI_URL=https://api.test\nDEBUG=YES\nPORT=443\n"+
		"TIMEOUT=1h30m\nOPS=a@b.example\nREGION=us-east-2\nCODE=abcdé\nWORKERS=10\nOPTIONAL=x\n")
	res := Check(mustSchema(t, checkExample), doc, CheckOptions{})
	if !res.OK || len(res.Issues) != 0 {
		t.Fatalf("Check = %+v; want a clean pass", res)
	}
}

func TestCheckMissing(t *testing.T) {
	res := Check(mustSchema(t, checkExample), mustDoc(t, "APP_ENV=\n"), CheckOptions{})
	if res.OK {
		t.Error("OK = true; want failure")
	}
	if !hasIssue(res, "COUNT", IssueMissing) || !hasIssue(res, "APP_ENV", IssueMissing) {
		t.Errorf("issues = %+v; want COUNT and APP_ENV missing", res.Issues)
	}
	// Optional keys may be absent.
	if hasIssue(res, "API_URL", IssueMissing) {
		t.Errorf("optional API_URL reported missing: %+v", res.Issues)
	}
	for _, is := range res.Issues {
		if is.Line == 0 {
			t.Errorf("issue %+v has no declaring line", is)
		}
	}
}

func TestCheckMismatches(t *testing.T) {
	cases := map[string]string{
		"COUNT":   "1.5",
		"APP_ENV": "staging",
		"API_URL": "not a url",
		"DEBUG":   "maybe",
		"PORT":    "65536",
		"TIMEOUT": "30",
		"OPS":     "Ops <ops@example.com>",
		"REGION":  "EU-WEST-1",
		"CODE":    "abcdefg",
		"WORKERS": "11",
	}
	for key, value := range cases {
		doc := mustDoc(t, "COUNT=1\nAPP_ENV=dev\n"+key+"=\""+value+"\"\n")
		res := Check(mustSchema(t, checkExample), doc, CheckOptions{})
		if res.OK || !hasIssue(res, key, IssueMismatch) {
			t.Errorf("%s=%q: issues = %+v; want a mismatch", key, value, res.Issues)
		}
	}
}

func TestCheckPortRejectsSignsAndZero(t *testing.T) {
	for _, v := range []string{"0", "+80", "-1", " 80", "8o"} {
		if isPort(v) {
			t.Errorf("isPort(%q) = true", v)
		}
	}
	for _, v := range []string{"1", "80", "65535", "0080"} {
		if !isPort(v) {
			t.Errorf("isPort(%q) = false", v)
		}
	}
}

func TestCheckLowerBoundOnLength(t *testing.T) {
	doc := mustDoc(t, "COUNT=1\nAPP_ENV=dev\nCODE=abc\n")
	if res := Check(mustSchema(t, checkExample), doc, CheckOptions{}); !hasIssue(res, "CODE", IssueMismatch) {
		t.Errorf("CODE=abc (3 < @min=4) not reported: %+v", res.Issues)
	}
}

// Mismatch messages go to CI logs and must not carry the value unless the
// caller opted in.
func TestCheckDoesNotLeakValues(t *testing.T) {
	const secret = "hunter2-supersecret"
	doc := mustDoc(t, "COUNT="+secret+"\nAPP_ENV="+secret+"\nREGION="+secret+"\nCODE="+secret+"\n")
	res := Check(mustSchema(t, checkExample), doc, CheckOptions{})
	for _, is := range res.Issues {
		if strings.Contains(is.Message, secret) {
			t.Errorf("message leaks the value: %q", is.Message)
		}
	}
	res = Check(mustSchema(t, checkExample), doc, CheckOptions{ShowValues: true})
	if !strings.Contains(res.Issues[0].Message, secret) {
		t.Errorf("ShowValues message = %q; want the value", res.Issues[0].Message)
	}
}

func TestCheckExtra(t *testing.T) {
	doc := mustDoc(t, "COUNT=1\nAPP_ENV=dev\nSURPRISE=1\n")
	schema := mustSchema(t, checkExample)

	res := Check(schema, doc, CheckOptions{})
	if !hasIssue(res, "SURPRISE", IssueExtra) || !res.OK || res.Failures() != 0 {
		t.Errorf("non-strict: %+v; want SURPRISE reported without failing", res)
	}
	res = Check(schema, doc, CheckOptions{Strict: true})
	if res.OK || res.Failures() != 1 {
		t.Errorf("strict: %+v; want SURPRISE to fail", res)
	}
	res = Check(schema, doc, CheckOptions{IgnoreExtra: true})
	if hasIssue(res, "SURPRISE", IssueExtra) {
		t.Errorf("IgnoreExtra: %+v; want no extra issues", res.Issues)
	}
}

func TestFromEnviron(t *testing.T) {
	d := FromEnviron([]string{"A=1", "B=x=y", "=C:=C:\\", "NOEQUALS", "1BAD=2", "EMPTY="})
	if got := d.Keys(); strings.Join(got, ",") != "A,B,EMPTY" {
		t.Fatalf("Keys = %v; want [A B EMPTY]", got)
	}
	if v, _ := d.Get("B"); v != "x=y" {
		t.Errorf("B = %q; want x=y", v)
	}
}
