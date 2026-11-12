package env

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// CheckReport is what WriteCheckReport needs to place findings in files.
type CheckReport struct {
	Result CheckResult
	// ExamplePath is where missing and mismatched keys are declared.
	ExamplePath string
	// EnvPath is where extra keys come from; empty for the process
	// environment.
	EnvPath string
	// Version is the envtool version, recorded in SARIF output.
	Version string
}

// CheckFormats lists the names accepted by WriteCheckReport.
var CheckFormats = []string{"text", "json", "github", "gitlab", "sarif"}

// WriteCheckReport writes r to w in format: text, json, github (workflow
// commands), gitlab (Code Quality report) or sarif (SARIF 2.1.0).
func WriteCheckReport(w io.Writer, r CheckReport, format string) error {
	switch format {
	case "text":
		return writeCheckText(w, r)
	case "json":
		return writeJSON(w, r.Result)
	case "github":
		return writeCheckGitHub(w, r)
	case "gitlab":
		return writeCheckGitLab(w, r)
	case "sarif":
		return writeCheckSARIF(w, r)
	default:
		return fmt.Errorf("unknown format %q (want %s)", format, strings.Join(CheckFormats, "|"))
	}
}

// writeJSON writes v indented, without escaping <, > and & so regex
// patterns stay readable.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// marshalCompact is json.Marshal without HTML escaping.
func marshalCompact(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

func writeCheckText(w io.Writer, r CheckReport) error {
	for _, is := range r.Result.Issues {
		if _, err := fmt.Fprintf(w, "%-9s %s: %s\n", is.Kind, is.Key, is.Message); err != nil {
			return err
		}
	}
	return nil
}

// issueFile returns the file an issue points at, the example for keys it
// declares and the env file for extra keys, as a slash-separated path without
// a leading "./" (GitLab rejects that prefix). It is "" for the process
// environment.
func (r CheckReport) issueFile(is Issue) string {
	p := r.ExamplePath
	if is.Kind == IssueExtra {
		p = r.EnvPath
	}
	if p == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(p))
}

// writeCheckGitHub prints GitHub Actions workflow commands, which the runner
// turns into annotations on the file and line:
// https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands
func writeCheckGitHub(w io.Writer, r CheckReport) error {
	for _, is := range r.Result.Issues {
		level := "error"
		if !is.Fails {
			level = "warning"
		}
		props := []string{}
		if f := r.issueFile(is); f != "" {
			props = append(props, "file="+ghEscapeProperty(f))
			if is.Line > 0 {
				props = append(props, fmt.Sprintf("line=%d", is.Line))
			}
		}
		props = append(props, "title="+ghEscapeProperty("envtool check: "+string(is.Kind)))
		msg := ghEscapeData(is.Key + ": " + is.Message)
		if _, err := fmt.Fprintf(w, "::%s %s::%s\n", level, strings.Join(props, ","), msg); err != nil {
			return err
		}
	}
	return nil
}

// ghEscapeData and ghEscapeProperty follow the escaping in @actions/core's
// command.ts, so a message cannot end the command early or inject another.
func ghEscapeData(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
}

func ghEscapeProperty(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C").Replace(s)
}

// gitlabIssue is one entry of a GitLab Code Quality report:
// https://docs.gitlab.com/ci/testing/code_quality/#code-quality-report-format
type gitlabIssue struct {
	Description string         `json:"description"`
	CheckName   string         `json:"check_name"`
	Fingerprint string         `json:"fingerprint"`
	Severity    string         `json:"severity"`
	Location    gitlabLocation `json:"location"`
}

type gitlabLocation struct {
	Path  string      `json:"path"`
	Lines gitlabLines `json:"lines"`
}

type gitlabLines struct {
	Begin int `json:"begin"`
}

func writeCheckGitLab(w io.Writer, r CheckReport) error {
	out := make([]gitlabIssue, 0, len(r.Result.Issues))
	for _, is := range r.Result.Issues {
		path := r.issueFile(is)
		if path == "" {
			path = "environment"
		}
		line := is.Line
		if line == 0 {
			line = 1 // lines.begin is required; extra keys have no declaring line
		}
		severity := "major"
		if !is.Fails {
			severity = "minor"
		}
		// The fingerprint identifies the finding across pipelines so GitLab
		// can tell new issues from fixed ones. It is built from the path, key
		// and kind only, never from the value.
		sum := sha256.Sum256([]byte(path + "\x00" + is.Key + "\x00" + string(is.Kind)))
		out = append(out, gitlabIssue{
			Description: is.Key + ": " + is.Message,
			CheckName:   "envtool-" + string(is.Kind),
			Fingerprint: hex.EncodeToString(sum[:16]),
			Severity:    severity,
			Location:    gitlabLocation{Path: path, Lines: gitlabLines{Begin: line}},
		})
	}
	return writeJSON(w, out)
}

// Minimal SARIF 2.1.0 types, enough for GitHub code scanning and other
// consumers: https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html
type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	ShortDescription sarifMessage `json:"shortDescription"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func writeCheckSARIF(w io.Writer, r CheckReport) error {
	results := make([]sarifResult, 0, len(r.Result.Issues))
	for _, is := range r.Result.Issues {
		level := "error"
		if !is.Fails {
			level = "warning"
		}
		res := sarifResult{
			RuleID:  string(is.Kind),
			Level:   level,
			Message: sarifMessage{Text: is.Key + ": " + is.Message},
		}
		if f := r.issueFile(is); f != "" {
			loc := sarifLocation{PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: f},
			}}
			if is.Line > 0 {
				loc.PhysicalLocation.Region = &sarifRegion{StartLine: is.Line}
			}
			res.Locations = []sarifLocation{loc}
		}
		results = append(results, res)
	}
	log := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "envtool",
				Version:        r.Version,
				InformationURI: "https://github.com/moveeeax/envtool",
				Rules: []sarifRule{
					{ID: string(IssueMissing), ShortDescription: sarifMessage{Text: "A required key is missing or empty"}},
					{ID: string(IssueMismatch), ShortDescription: sarifMessage{Text: "A value does not match its declared type or constraint"}},
					{ID: string(IssueExtra), ShortDescription: sarifMessage{Text: "A key is not declared in the example"}},
				},
			}},
			Results: results,
		}},
	}
	return writeJSON(w, log)
}
