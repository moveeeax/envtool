package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/moveeeax/envtool/internal/env"
	"github.com/spf13/cobra"
)

// exitError makes main exit with code without printing anything further: the
// command has already reported, or was asked to stay quiet.
type exitError struct{ code int }

func (exitError) Error() string { return "" }

// ExitCode reports the process exit code main should use for this error.
func (e exitError) ExitCode() int { return e.code }

// execProgram replaces the current process with argv. It is a variable so
// tests can observe the call instead of losing the test binary to exec.
var execProgram = execArgv

func newCheckCmd() *cobra.Command {
	var (
		examplePath string
		envPath     string
		fromEnv     bool
		format      string
		strict      bool
		showValues  bool
		quiet       bool
	)
	cmd := &cobra.Command{
		Use:   "check [flags] [-- COMMAND [ARG...]]",
		Short: "Validate a .env file or the process environment against an annotated .env.example",
		Long: `Validate a .env file, or with --from-env the process environment, against the
contract declared by annotations in an example file:

  # @type=port @required
  PORT=8080
  # @pattern=/^[a-z]{2}-[a-z]+-\d$/
  REGION=eu-west-1

Types: string (default), int, bool, url, enum(a,b), port, duration, email.
Constraints: @required, @enum=a,b, @pattern=/regex/, @min=N, @max=N (the
number for int and port, the length for other types).

Messages never contain the value, only its length, unless --show-values is
given. Extra keys (in the env but not the example) are reported but only
fail the check under --strict; with --from-env they are not reported.

With --from-env and a command after --, check runs as an entrypoint wrapper:
it reports problems on stderr and exits 1, or replaces itself with the
command when the environment passes.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var argv []string
			if dash := cmd.ArgsLenAtDash(); dash >= 0 {
				if dash > 0 {
					return fmt.Errorf("unexpected argument %q before --", args[0])
				}
				argv = args
				if len(argv) == 0 {
					return fmt.Errorf("no command after --")
				}
				if !fromEnv {
					return fmt.Errorf("running a command after -- needs --from-env")
				}
			} else if len(args) > 0 {
				return fmt.Errorf("unexpected argument %q; pass the file with --env, or put a command after --", args[0])
			}
			if fromEnv && strict {
				return fmt.Errorf("--strict cannot be combined with --from-env: every process has variables such as PATH that no example declares")
			}
			if fromEnv && cmd.Flags().Changed("env") {
				return fmt.Errorf("--env and --from-env are mutually exclusive")
			}
			if !inList(format, env.CheckFormats) {
				return fmt.Errorf("unknown format %q (want %s)", format, strings.Join(env.CheckFormats, "|"))
			}

			schema, err := env.ParseSchemaFile(examplePath)
			if err != nil {
				return fmt.Errorf("example %s: %w", examplePath, err)
			}
			var doc *env.Doc
			reportEnvPath := envPath
			if fromEnv {
				doc = env.FromEnviron(os.Environ())
				reportEnvPath = ""
			} else if doc, err = env.ParseFile(envPath); err != nil {
				return fmt.Errorf("env %s: %w", envPath, err)
			}

			res := env.Check(schema, doc, env.CheckOptions{
				Strict:      strict,
				IgnoreExtra: fromEnv,
				ShowValues:  showValues,
			})

			// In wrapper mode stdout belongs to the wrapped program, so the
			// report goes to stderr.
			var out io.Writer = os.Stdout
			if argv != nil {
				out = os.Stderr
			}
			if !quiet {
				err := env.WriteCheckReport(out, env.CheckReport{
					Result:      res,
					ExamplePath: examplePath,
					EnvPath:     reportEnvPath,
					Version:     version,
				}, format)
				if err != nil {
					return err
				}
			}
			if !res.OK {
				if quiet {
					return exitError{code: 1}
				}
				return fmt.Errorf("%d check issue(s) failed", res.Failures())
			}
			if argv != nil {
				return execProgram(argv)
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&examplePath, "example", "e", ".env.example", "annotated example file that declares the contract")
	f.StringVar(&envPath, "env", ".env", ".env file to validate")
	f.BoolVar(&fromEnv, "from-env", false, "validate the process environment instead of a file")
	f.StringVar(&format, "format", "text", "output format: "+strings.Join(env.CheckFormats, "|"))
	f.BoolVar(&strict, "strict", false, "fail on keys the example does not declare")
	f.BoolVar(&showValues, "show-values", false, "include offending values in messages (may print secrets)")
	f.BoolVarP(&quiet, "quiet", "q", false, "print nothing; report only through the exit code")
	return cmd
}

func inList(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
