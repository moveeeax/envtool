package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const checkExample = `# @type=port @required
PORT=8080
# @type=enum(dev,prod) @required
APP_ENV=dev
`

func TestCheckCommandPasses(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	f := writeTemp(t, ".env", "PORT=3000\nAPP_ENV=prod\n")
	out, err := run(t, "check", "--example", ex, "--env", f)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if out != "" {
		t.Errorf("check output = %q; want nothing on a clean pass", out)
	}
}

func TestCheckCommandFails(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	f := writeTemp(t, ".env", "PORT=hunter2\nEXTRA=1\n")
	out, err := run(t, "check", "-e", ex, "--env", f)
	if err == nil {
		t.Fatal("check = nil error; want a failure")
	}
	if !strings.Contains(err.Error(), "2 check issue(s) failed") {
		t.Errorf("error = %q", err)
	}
	for _, want := range []string{"mismatch  PORT:", "missing   APP_ENV:", "extra     EXTRA:"} {
		if !strings.Contains(out, want) {
			t.Errorf("check output = %q; want %q", out, want)
		}
	}
	if strings.Contains(out, "hunter2") {
		t.Errorf("check output leaks a value: %q", out)
	}
}

func TestCheckCommandStrict(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	f := writeTemp(t, ".env", "PORT=1\nAPP_ENV=dev\nEXTRA=1\n")
	if _, err := run(t, "check", "-e", ex, "--env", f); err != nil {
		t.Errorf("non-strict check with an extra key: %v", err)
	}
	if _, err := run(t, "check", "-e", ex, "--env", f, "--strict"); err == nil {
		t.Error("check --strict with an extra key = nil error")
	}
}

func TestCheckCommandJSON(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	f := writeTemp(t, ".env", "PORT=0\nAPP_ENV=dev\n")
	out, _ := run(t, "check", "-e", ex, "--env", f, "--format", "json")
	var res struct {
		OK     bool `json:"ok"`
		Issues []struct {
			Key  string `json:"key"`
			Kind string `json:"kind"`
			Line int    `json:"line"`
		} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if res.OK || len(res.Issues) != 1 || res.Issues[0].Key != "PORT" || res.Issues[0].Kind != "mismatch" || res.Issues[0].Line != 2 {
		t.Errorf("result = %+v", res)
	}
}

func TestCheckCommandGitHubFormat(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	f := writeTemp(t, ".env", "PORT=x\nAPP_ENV=dev\n")
	out, _ := run(t, "check", "-e", ex, "--env", f, "--format", "github")
	if !strings.HasPrefix(out, "::error file=") || !strings.Contains(out, ",line=2,") {
		t.Errorf("github output = %q", out)
	}
}

func TestCheckCommandQuiet(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	f := writeTemp(t, ".env", "PORT=x\n")
	out, err := run(t, "check", "-q", "-e", ex, "--env", f)
	if out != "" {
		t.Errorf("check -q output = %q; want nothing", out)
	}
	ec, ok := err.(interface{ ExitCode() int })
	if !ok || ec.ExitCode() != 1 || err.Error() != "" {
		t.Errorf("check -q error = %v (%T); want a silent exit code 1", err, err)
	}
}

func TestCheckCommandFromEnv(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	t.Setenv("PORT", "8080")
	t.Setenv("APP_ENV", "prod")
	// Variables such as PATH and HOME are always present and must not be
	// reported as extra.
	out, err := run(t, "check", "-e", ex, "--from-env")
	if err != nil || out != "" {
		t.Errorf("check --from-env = %q, %v; want a silent pass", out, err)
	}
	t.Setenv("APP_ENV", "qa")
	if _, err := run(t, "check", "-e", ex, "--from-env"); err == nil {
		t.Error("check --from-env with APP_ENV=qa = nil error")
	}
}

// stubExec replaces execProgram for the test and records what it was asked
// to run.
func stubExec(t *testing.T) *[]string {
	t.Helper()
	var got []string
	old := execProgram
	execProgram = func(argv []string) error {
		got = append([]string{}, argv...)
		return nil
	}
	t.Cleanup(func() { execProgram = old })
	return &got
}

func TestCheckCommandWrapperExecsOnPass(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	t.Setenv("PORT", "8080")
	t.Setenv("APP_ENV", "dev")
	got := stubExec(t)
	out, err := run(t, "check", "-e", ex, "--from-env", "--", "./app", "--flag", "x")
	if err != nil {
		t.Fatalf("check wrapper: %v", err)
	}
	if want := []string{"./app", "--flag", "x"}; !reflect.DeepEqual(*got, want) {
		t.Errorf("exec argv = %v; want %v", *got, want)
	}
	if out != "" {
		t.Errorf("wrapper wrote %q to stdout, which belongs to the app", out)
	}
}

func TestCheckCommandWrapperDoesNotExecOnFailure(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	t.Setenv("PORT", "not-a-port")
	t.Setenv("APP_ENV", "dev")
	got := stubExec(t)
	out, err := run(t, "check", "-e", ex, "--from-env", "--", "./app")
	if err == nil {
		t.Fatal("check wrapper with a bad PORT = nil error")
	}
	if *got != nil {
		t.Errorf("exec ran %v although the check failed", *got)
	}
	if out != "" {
		t.Errorf("wrapper wrote its report to stdout: %q", out)
	}
}

func TestCheckCommandArgumentErrors(t *testing.T) {
	ex := writeTemp(t, ".env.example", checkExample)
	f := writeTemp(t, ".env", "PORT=1\nAPP_ENV=dev\n")
	stubExec(t)
	for _, args := range [][]string{
		{"check", "-e", ex, "--env", f, "--", "./app"},    // a command needs --from-env
		{"check", "-e", ex, "--from-env", "--"},           // empty command
		{"check", "-e", ex, "--from-env", "x", "--", "y"}, // stray argument before --
		{"check", "-e", ex, f},                            // file given positionally
		{"check", "-e", ex, "--from-env", "--strict"},
		{"check", "-e", ex, "--from-env", "--env", f},
		{"check", "-e", ex, "--env", f, "--format", "xml"},
		{"check", "-e", writeTemp(t, "bad.example", "# @type=nope\nA=1\n"), "--env", f},
		{"check", "-e", ex, "--env", f + ".missing"},
	} {
		if _, err := run(t, args...); err == nil {
			t.Errorf("run(%v) = nil error", args)
		}
	}
}

func TestSchemaCommand(t *testing.T) {
	ex := writeTemp(t, ".env.example", "# Listen port\n"+checkExample)
	out, err := run(t, "schema", ex)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	if !strings.Contains(out, "| `PORT` | `port` | yes | `8080` | Listen port |") {
		t.Errorf("schema markdown = %q", out)
	}
	out, err = run(t, "schema", "-f", "jsonschema", ex)
	if err != nil {
		t.Fatalf("schema -f jsonschema: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil || doc["$schema"] == nil {
		t.Errorf("schema -f jsonschema = %q, %v", out, err)
	}
	if _, err := run(t, "schema", "-f", "yaml", ex); err == nil {
		t.Error("schema -f yaml = nil error")
	}
}
