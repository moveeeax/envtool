package env

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func sampleReport(t *testing.T) CheckReport {
	t.Helper()
	schema := mustSchema(t, "# @type=int @required\nCOUNT=1\n# @required\nNAME=x\n")
	doc := mustDoc(t, "COUNT=abc\nEXTRA=1\n")
	return CheckReport{
		Result:      Check(schema, doc, CheckOptions{}),
		ExamplePath: "./config/.env.example",
		EnvPath:     ".env",
		Version:     "1.2.3",
	}
}

func render(t *testing.T, r CheckReport, format string) string {
	t.Helper()
	var b bytes.Buffer
	if err := WriteCheckReport(&b, r, format); err != nil {
		t.Fatalf("WriteCheckReport(%s): %v", format, err)
	}
	return b.String()
}

func TestCheckReportText(t *testing.T) {
	out := render(t, sampleReport(t), "text")
	for _, want := range []string{"mismatch  COUNT: expected int", "missing   NAME:", "extra     EXTRA:"} {
		if !strings.Contains(out, want) {
			t.Errorf("text = %q; want %q", out, want)
		}
	}
}

func TestCheckReportGitHub(t *testing.T) {
	lines := strings.Split(strings.TrimSpace(render(t, sampleReport(t), "github")), "\n")
	want := []string{
		"::error file=config/.env.example,line=2,title=envtool check%3A mismatch::COUNT: expected int, got a 3-character value",
		"::error file=config/.env.example,line=4,title=envtool check%3A missing::NAME: required key is missing",
		"::warning file=.env,title=envtool check%3A extra::EXTRA: key is not declared in the example",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("github =\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

// A newline in a message must not start a second workflow command.
func TestGitHubEscaping(t *testing.T) {
	if got := ghEscapeData("a\n::error::x%"); got != "a%0A::error::x%25" {
		t.Errorf("ghEscapeData = %q", got)
	}
	if got := ghEscapeProperty("a,b:c"); got != "a%2Cb%3Ac" {
		t.Errorf("ghEscapeProperty = %q", got)
	}
}

func TestCheckReportGitLab(t *testing.T) {
	var issues []map[string]any
	if err := json.Unmarshal([]byte(render(t, sampleReport(t), "gitlab")), &issues); err != nil {
		t.Fatalf("gitlab output is not JSON: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("len = %d; want 3", len(issues))
	}
	seen := map[string]bool{}
	for _, is := range issues {
		for _, field := range []string{"description", "check_name", "fingerprint", "severity"} {
			if s, _ := is[field].(string); s == "" {
				t.Errorf("issue %v has no %s", is, field)
			}
		}
		loc := is["location"].(map[string]any)
		path := loc["path"].(string)
		if strings.HasPrefix(path, "./") {
			t.Errorf("path %q has a ./ prefix, which GitLab rejects", path)
		}
		if begin := loc["lines"].(map[string]any)["begin"].(float64); begin < 1 {
			t.Errorf("lines.begin = %v; want >= 1", begin)
		}
		fp := is["fingerprint"].(string)
		if seen[fp] {
			t.Errorf("duplicate fingerprint %s", fp)
		}
		seen[fp] = true
	}
	if issues[2]["severity"] != "minor" || issues[0]["severity"] != "major" {
		t.Errorf("severities = %v, %v; want major for failures, minor for a non-failing extra", issues[0]["severity"], issues[2]["severity"])
	}
}

func TestCheckReportSARIF(t *testing.T) {
	var log struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID    string `json:"ruleId"`
				Level     string `json:"level"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region *struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(render(t, sampleReport(t), "sarif")), &log); err != nil {
		t.Fatalf("sarif output is not JSON: %v", err)
	}
	if log.Version != "2.1.0" || len(log.Runs) != 1 || log.Runs[0].Tool.Driver.Name != "envtool" || log.Runs[0].Tool.Driver.Version != "1.2.3" {
		t.Fatalf("sarif header = %+v", log)
	}
	res := log.Runs[0].Results
	if len(res) != 3 {
		t.Fatalf("results = %d; want 3", len(res))
	}
	first := res[0].Locations[0].PhysicalLocation
	if res[0].RuleID != "mismatch" || res[0].Level != "error" || first.ArtifactLocation.URI != "config/.env.example" || first.Region == nil || first.Region.StartLine != 2 {
		t.Errorf("first result = %+v", res[0])
	}
	if res[2].Level != "warning" || res[2].Locations[0].PhysicalLocation.Region != nil {
		t.Errorf("extra result = %+v; want a warning without a region", res[2])
	}
}

// With the process environment there is no env file, so extra issues (if a
// caller reports them at all) carry no file location.
func TestCheckReportWithoutEnvFile(t *testing.T) {
	r := sampleReport(t)
	r.EnvPath = ""
	out := render(t, r, "github")
	if !strings.Contains(out, "::warning title=envtool check%3A extra::EXTRA") {
		t.Errorf("github = %q; want the extra warning without file=", out)
	}
}

func TestCheckReportUnknownFormat(t *testing.T) {
	if err := WriteCheckReport(&bytes.Buffer{}, sampleReport(t), "xml"); err == nil {
		t.Error("WriteCheckReport(xml) = nil error")
	}
}
