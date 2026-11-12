package cmd

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/moveeeax/envtool/internal/env"
	"github.com/spf13/cobra"
)

// errDifferencesFound is returned by `diff --exit-code` when the two files
// differ. Its message is empty: diff has already written the differences to
// stdout, so main should set the process exit code without printing a
// redundant second "envtool: ..." line to stderr.
//
// Returning this from RunE (rather than calling os.Exit(1) directly, as a
// previous version did) keeps the command testable in-process: os.Exit tears
// down the whole test binary, not just the command under test, so that
// approach could never be exercised by cmd's own test suite.
type errDifferencesFound struct{}

func (errDifferencesFound) Error() string { return "" }

// ExitCode reports the process exit code main should use for this error.
func (errDifferencesFound) ExitCode() int { return 1 }

// saltEnvVar names the variable that holds the --fingerprint salt.
const saltEnvVar = "ENVTOOL_FINGERPRINT_SALT"

// diffOptions selects what diff prints next to each key.
type diffOptions struct {
	showValues  bool
	fingerprint bool
	shape       bool
	salt        []byte
}

// diffEntry is one element of `diff --format json`.
type diffEntry struct {
	Key              string     `json:"key"`
	Kind             string     `json:"kind"`
	Left             *string    `json:"left,omitempty"`
	Right            *string    `json:"right,omitempty"`
	LeftFingerprint  string     `json:"left_fingerprint,omitempty"`
	RightFingerprint string     `json:"right_fingerprint,omitempty"`
	LeftShape        *env.Shape `json:"left_shape,omitempty"`
	RightShape       *env.Shape `json:"right_shape,omitempty"`
}

func newDiffCmd() *cobra.Command {
	var (
		exitCode bool
		format   string
		keysOnly bool
		opts     diffOptions
	)
	cmd := &cobra.Command{
		Use:   "diff [flags] FILE_A FILE_B",
		Short: "Show which keys differ between two .env files, without printing values",
		Long: `Show which keys were added (+), removed (-) or changed (~) between two .env
files. Values are not printed unless --show-values is given, so the output
can go into a pull request or a CI log during a secret rotation.

--fingerprint prints a short HMAC-SHA256 of each value, keyed with the salt
in ` + saltEnvVar + `. Runs with the same salt print the same fingerprint for the
same value. Without the variable a random salt is used, and fingerprints can
only be compared within that one run.

--shape prints the length, character classes and token kind of each value.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" && format != "json" {
				return fmt.Errorf("unknown format %q (want text|json)", format)
			}
			left, err := env.ParseFile(args[0])
			if err != nil {
				return err
			}
			right, err := env.ParseFile(args[1])
			if err != nil {
				return err
			}
			if opts.fingerprint {
				opts.salt, err = fingerprintSalt(cmd.ErrOrStderr())
				if err != nil {
					return err
				}
			}
			changes := env.Diff(left, right)
			if format == "json" {
				err = writeDiffJSON(os.Stdout, changes, opts)
			} else {
				err = writeDiffText(os.Stdout, changes, opts)
			}
			if err != nil {
				return err
			}
			if exitCode && len(changes) > 0 {
				return errDifferencesFound{}
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.BoolVar(&exitCode, "exit-code", false, "exit 1 when differences are found")
	f.StringVarP(&format, "format", "f", "text", "output format: text|json")
	f.BoolVar(&opts.showValues, "show-values", false, "print the old and new values (the output then holds secrets)")
	f.BoolVar(&keysOnly, "keys-only", false, "print keys and change kinds only (the default; rejects the other value flags)")
	f.BoolVar(&opts.fingerprint, "fingerprint", false, "print a salted fingerprint of each value (salt from "+saltEnvVar+")")
	f.BoolVar(&opts.shape, "shape", false, "print the length, character classes and token kind of each value")
	cmd.MarkFlagsMutuallyExclusive("keys-only", "show-values")
	cmd.MarkFlagsMutuallyExclusive("keys-only", "fingerprint")
	cmd.MarkFlagsMutuallyExclusive("keys-only", "shape")
	return cmd
}

// fingerprintSalt returns the salt from saltEnvVar, or a random one with a
// note on stderr explaining that the fingerprints are good for this run only.
func fingerprintSalt(stderr io.Writer) ([]byte, error) {
	if s := os.Getenv(saltEnvVar); s != "" {
		return []byte(s), nil
	}
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	fmt.Fprintf(stderr, "envtool: %s is not set; fingerprints use a one-off salt and only compare within this run\n", saltEnvVar)
	return salt, nil
}

// annotation renders the --fingerprint and --shape details for one side, or
// "" when neither was asked for.
func (o diffOptions) annotation(value string) string {
	var parts []string
	if o.fingerprint {
		parts = append(parts, "fp="+env.Fingerprint(o.salt, value))
	}
	if o.shape {
		parts = append(parts, env.ShapeOf(value).String())
	}
	return strings.Join(parts, " ")
}

func writeDiffText(w io.Writer, changes []env.Change, o diffOptions) error {
	for _, c := range changes {
		var line string
		switch c.Kind {
		case env.Added:
			line = "+ " + c.Key
			if o.showValues {
				line += "=" + c.Right
			}
			if a := o.annotation(c.Right); a != "" {
				line += " [" + a + "]"
			}
		case env.Removed:
			line = "- " + c.Key
			if o.showValues {
				line += "=" + c.Left
			}
			if a := o.annotation(c.Left); a != "" {
				line += " [" + a + "]"
			}
		case env.Changed:
			if o.showValues {
				line = fmt.Sprintf("~ %s: %s -> %s", c.Key, c.Left, c.Right)
			} else {
				line = "~ " + c.Key + " (changed)"
			}
			if a := o.annotation(c.Left); a != "" {
				line += " [" + a + " -> " + o.annotation(c.Right) + "]"
			}
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func writeDiffJSON(w io.Writer, changes []env.Change, o diffOptions) error {
	out := make([]diffEntry, 0, len(changes))
	for _, c := range changes {
		e := diffEntry{Key: c.Key, Kind: c.Kind.String()}
		hasLeft, hasRight := c.Kind != env.Added, c.Kind != env.Removed
		if hasLeft {
			o.fill(c.Left, &e.Left, &e.LeftFingerprint, &e.LeftShape)
		}
		if hasRight {
			o.fill(c.Right, &e.Right, &e.RightFingerprint, &e.RightShape)
		}
		out = append(out, e)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// fill sets the value, fingerprint and shape fields of one side of a JSON
// entry according to the options.
func (o diffOptions) fill(value string, v **string, fp *string, shape **env.Shape) {
	if o.showValues {
		*v = &value
	}
	if o.fingerprint {
		*fp = env.Fingerprint(o.salt, value)
	}
	if o.shape {
		s := env.ShapeOf(value)
		*shape = &s
	}
}
