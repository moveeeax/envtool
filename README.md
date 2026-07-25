# envtool

A small, dependency-light CLI for working with `.env` files: merge, diff,
validate, redact and convert them between formats.

Managing dotenv files across environments gets messy fast — keys drift between
`.env.example` and `.env`, secrets leak into logs, and every service wants the
values in a slightly different shape. `envtool` keeps those chores honest.

## Install

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

Show what changed between two files (`+` added, `-` removed, `~` changed).

```sh
envtool diff .env.example .env
envtool diff --exit-code .env.example .env   # non-zero when they differ
```

`diff` prints values, so treat its output as sensitive. Parse errors, by
contrast, never echo the offending line — only its number — because a malformed
line is often a stray continuation of a multi-line credential.

### validate

Fail if required keys are missing or empty — handy in CI.

```sh
envtool validate --required DB_URL,API_KEY,PORT .env
```

`--required` must name at least one key; `envtool validate .env` on its own is
an error rather than a silent pass, so a misconfigured CI gate fails loudly.

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

## Development

```sh
gofmt -s -l .        # must print nothing
go vet ./...
go test -race ./...
```

## License

MIT — see [LICENSE](LICENSE).
