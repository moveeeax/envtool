# envtool

A small, dependency-light CLI for working with `.env` files: merge, diff,
validate, redact and convert them between formats.

Managing dotenv files across environments gets messy fast — keys drift between
`.env.example` and `.env`, secrets leak into logs, and every service wants the
values in a slightly different shape. `envtool` keeps those chores honest.

## Install

```sh
go install github.com/cybercapybara/envtool@latest
```

Or build from source:

```sh
git clone https://github.com/cybercapybara/envtool
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

### validate

Fail if required keys are missing or empty — handy in CI.

```sh
envtool validate --required DB_URL,API_KEY,PORT .env
```

### redact

Mask sensitive values before sharing a file or printing it in logs.

```sh
envtool redact .env
envtool redact --keep 4 --format yaml .env
envtool redact --match PIN,SEED .env
```

Keys are treated as secret when they contain substrings like `SECRET`,
`PASSWORD`, `TOKEN`, `API_KEY`, `PRIVATE`, `CREDENTIAL` (see `--match` to
override).

### export

Convert a file to another format: `dotenv`, `shell`, `json` or `yaml`.

```sh
envtool export -f shell .env > env.sh && source env.sh
envtool export -f yaml .env
envtool export -f json --sort .env
```

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
| `json`   | flat object of string values                     |
| `yaml`   | flat mapping of string values                    |

## Development

```sh
go test ./...
go vet ./...
```

## License

MIT — see [LICENSE](LICENSE).
