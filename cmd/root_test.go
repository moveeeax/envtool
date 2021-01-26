package cmd

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := ioutil.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	root := newRootCmd()
	root.SetArgs(args)
	err := root.Execute()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestMergeCommand(t *testing.T) {
	a := writeTemp(t, "a.env", "HOST=local\nPORT=1\n")
	b := writeTemp(t, "b.env", "PORT=2\nDEBUG=true\n")
	out, err := run(t, "merge", a, b)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if !strings.Contains(out, "PORT=2") || !strings.Contains(out, "DEBUG=true") {
		t.Errorf("merge output = %q", out)
	}
}

func TestValidateCommandFails(t *testing.T) {
	f := writeTemp(t, "c.env", "A=1\n")
	_, err := run(t, "validate", "--required", "A,B", f)
	if err == nil {
		t.Fatal("validate expected error for missing key B")
	}
}

func TestExportJSONCommand(t *testing.T) {
	f := writeTemp(t, "d.env", "A=1\n")
	out, err := run(t, "export", "-f", "json", f)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(out, "\"A\": \"1\"") {
		t.Errorf("export output = %q", out)
	}
}
