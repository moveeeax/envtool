# envtool

[![ci](https://github.com/moveeeax/envtool/actions/workflows/ci.yml/badge.svg)](https://github.com/moveeeax/envtool/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go)](go.mod)

A small, dependency-light CLI for working with `.env` files: merge, diff,
validate, check against a typed `.env.example`, redact and convert them
between formats.

Managing dotenv files across environments gets messy fast — keys drift between
`.env.example` and `.env`, secrets leak into logs, and every service wants the
values in a slightly different shape. `envtool` keeps those chores honest.

## Install

Requires Go 1.26 or newer.

```sh
go install github.com/moveeeax/envtool@latest
```

Or build from source:

```sh
git clone https://github.com/moveeeax/envtool
cd envtool
go build -o envtool .
```

## Usage

### merge

Overlay several files; later files win on conflicts, order is preserved.

```sh
envtool merge .env.defaults .env.local
envtool merge -f json base.env override.env
```

### diff

Show which keys changed between two files (`+` added, `-` removed, `~`
changed). Values are not printed, so the output can go into a rotation PR or
a CI log.

```sh
envtool diff .env.staging .env.prod
envtool diff --exit-code .env.example .env   # non-zero when they differ
envtool diff -f json .env.old .env.new       # [{"key": ..., "kind": ...}]
```

```text
- OLD_KEY
+ NEW_KEY
~ API_KEY (changed)
```

| Flag            | Prints                                                        |
|-----------------|---------------------------------------------------------------|
| `--keys-only`   | keys and kinds only; the default, spelled out                 |
| `--fingerprint` | a 12-hex-character HMAC-SHA256 of each side (`fp=...`)        |
| `--shape`       | length, character classes and token kind of each side         |
| `--show-values` | the values themselves (`~ KEY: old -> new`); output is secret |

`--fingerprint` keys the hash with `ENVTOOL_FINGERPRINT_SALT`. Two runs with
the same salt print the same fingerprint for the same value, so a reviewer can
confirm that prod now holds what staging holds without seeing either. Without
the variable a random salt is used and fingerprints only compare within that
run. Anyone holding the salt can test guesses against a fingerprint, so keep
the salt as secret as the values.

`--shape` reports, per side, something like
`len=40 chars=lower+digit kind=hex`. The kinds are `empty`, `int`, `bool`,
`uuid`, `jwt`, `url`, `hex`, `base64` (16+ characters for the last two) and
`text`; a rotated token whose kind changed is worth a second look.

Parse errors never echo the offending line, only its number, because a
malformed line is often a stray continuation of a multi-line credential.

### validate

Fail if required keys are missing or empty — handy in CI.

```sh
envtool validate --required DB_URL,API_KEY,PORT .env
```

`--required` must name at least one key; `envtool validate .env` on its own is
an error rather than a silent pass, so a misconfigured CI gate fails loudly.

### check

Validate a `.env` file, or the process environment, against the contract
declared by annotations in `.env.example`. This is the `check` command of the
archived [envsync](https://github.com/moveeeax/envsync).

```sh
envtool check                                  # .env against .env.example
envtool check --example config/.env.example --env .env.ci --strict
envtool check --format github                  # annotations on the PR
envtool check --from-env -- ./app --serve      # container entrypoint
```

Annotations go in comments, either on the lines just above a key (a blank
line ends the block) or after the value:

```dotenv
# Port the HTTP server listens on
# @type=port @required
PORT=8080

# @type=enum(dev,staging,prod) @required
APP_ENV=dev

# @pattern=/^[a-z]{2}-[a-z]+-\d$/
REGION=eu-west-1

SESSION_KEY=changeme # @required @min=32 @max=32
TIMEOUT=30s          # @type=duration
ALERTS_TO=ops@example.com # @type=email
```

| Annotation        | Meaning                                                            |
|-------------------|--------------------------------------------------------------------|
| `@type=NAME`      | `string` (default), `int`, `bool`, `url`, `port` (1-65535), `duration` (Go `time.ParseDuration`, e.g. `1h30m`), `email` (a bare address) |
| `@type=enum(a,b)` | one of the listed values; `@enum=a, b` is the same                 |
| `@required`       | the key must be present and non-empty                              |
| `@pattern=/re/`   | the value must match the Go regular expression; it is not anchored, so write `^...$` for a full match |
| `@min=N` `@max=N` | the number for `int` and `port`, the length in characters otherwise |

An unknown type, a regex that does not compile or `@min` above `@max` is an
error with the line number, not a silent fallback to `string`.

Findings are `missing` (required key absent or empty), `mismatch` (value fails
its type or constraint) and `extra` (key not in the example; fails only with
`--strict`). Messages give the length of an offending value, never the value,
unless `--show-values` is passed.

`--format` takes `text` (default), `json`, `github` (workflow commands that
annotate the declaring line of `.env.example`), `gitlab` (a
[Code Quality](https://docs.gitlab.com/ci/testing/code_quality/) report) or
`sarif` (SARIF 2.1.0, e.g. for GitHub code scanning). `-q` prints nothing and
reports through the exit code alone.

`--from-env` checks `os.Environ()` instead of a file, for containers and
Kubernetes where config arrives as real variables. Extra keys are not
reported there (every process has `PATH` and `HOME`), and `--strict` is
rejected. Put a command after `--` and `check` becomes an entrypoint wrapper:
it prints any problems to stderr and exits 1, or, when the environment
passes, replaces itself with the command via `exec`, so the app keeps PID 1
and receives signals directly. On Windows, which has no `exec`, the command
runs as a child and its exit code is passed through.

```dockerfile
ENTRYPOINT ["envtool", "check", "--example", "/app/.env.example", "--from-env", "--"]
CMD ["/app/server"]
```

Coming from envsync: `-f/--env FILE` is now `--env FILE`, `--json` is
`--format json`. `--fix` was not carried over, because envtool never writes
files; copy the missing keys from `envtool schema` or the example by hand.

### schema

Document the annotated example for a README or an editor.

```sh
envtool schema .env.example                  # Markdown table
envtool schema -f jsonschema .env.example    # JSON Schema draft 2020-12
```

The table has name, type with constraints, required, default and the comment
text as description. The JSON Schema describes the object that `envtool
export -f json` prints: every property is a string, and the declared type
becomes a `pattern` (int, port, bool, duration), a `format` (`uri`, `email`)
or an `enum`. Length bounds map to `minLength`/`maxLength`; numeric bounds on
`int` and `port` have no JSON Schema form for a string and are only enforced
by `check`. Patterns are copied as written, so stick to syntax that Go and
ECMA-262 regexes share (no `(?i)`) if other validators will read the schema.

Defaults of keys that look secret (same matchers as `redact`) are left out of
both formats.

### redact

Mask sensitive values before sharing a file or printing it in logs.

```sh
envtool redact .env
envtool redact --keep 4 --format yaml .env
envtool redact --match PIN,SEED .env
```

Keys are treated as secret when they contain substrings like `SECRET`,
`PASSWORD`, `TOKEN`, `API_KEY`, `PRIVATE`, `CREDENTIAL` (see `--match` to
override). Matching is case-insensitive and whitespace around each `--match`
entry is ignored, so `--match 'TOKEN, SECRET'` behaves like `--match
'TOKEN,SECRET'`. If `--match` ends up with no usable entries the built-in list
applies, so a malformed value never quietly disables redaction.

### export

Convert a file to another format: `dotenv`, `shell`, `json` or `yaml`.

```sh
(umask 077; envtool export -f shell .env > env.sh) && . ./env.sh
envtool export -f yaml .env
envtool export -f json --sort .env
```

`envtool` itself only ever writes to stdout — it never creates or overwrites a
file. That makes your shell's redirection responsible for the permissions of
anything you capture, hence the `umask 077` above: without it `env.sh` lands
world-readable with your secrets in it.

Key order follows the source file for every format (`--sort` overrides it).

### keys / get

Inspect a file without printing every value.

```sh
envtool keys --sort .env
envtool get DATABASE_URL .env
```

## Formats

| Format   | Notes                                            |
|----------|--------------------------------------------------|
| `dotenv` | `KEY=VALUE`, values quoted only when needed      |
| `shell`  | `export KEY='VALUE'`, safe to `eval`             |
| `json`   | flat object of string values, in file order      |
| `yaml`   | flat mapping of string values, quoted when a plain scalar would load as a bool, null, number or date |

A leading UTF-8 byte-order mark on input is stripped rather than parsed as
part of the first key, so files saved by Windows tools (PowerShell's
`Out-File`, Notepad's default "UTF-8") work without editing.

## Development

```sh
gofmt -s -l .        # must print nothing
go vet ./...
go test -race ./...
```

## License

MIT — see [LICENSE](LICENSE).
