# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Fixed
- The parser strips a leading UTF-8 byte-order mark instead of gluing it onto
  the first key. Windows tools (PowerShell's `Out-File`, Notepad's default
  "UTF-8") commonly write one, and every command used to fail on line 1 of an
  otherwise valid file.
- `redact --keep` slices by rune instead of byte. A secret containing a
  multi-byte character (accented letters, non-Latin scripts, emoji) could be
  cut mid-character, producing invalid UTF-8 that a JSON or YAML encoder would
  silently mangle instead of the readable prefix that was asked for.
- `diff --exit-code` now returns a typed error instead of calling `os.Exit(1)`
  from inside the command. Behavior at the command line is unchanged, but the
  command is now covered by tests: the previous approach tore down the whole
  process, so it could never be exercised in-process by the test suite.
- `redact --match` no longer ignores entries with surrounding whitespace.
  `--match 'TOKEN, SECRET'` previously left every `SECRET` key unredacted
  because the matcher was compared as `" SECRET"`. A `--match` list with no
  usable entries now falls back to the built-in matchers instead of matching
  nothing.
- Parse errors no longer echo the content of the offending line. A line without
  `=` is commonly a stray continuation of a multi-line credential, so the raw
  text could reach stderr and CI logs; only the line number is reported now.
- `validate` requires `--required` to name at least one key. Running
  `envtool validate .env` with no (or an empty) `--required` previously exited
  0 without checking anything, silently passing a CI gate.
- YAML export quotes values that a loader would resolve to a bool, null, number
  or date. `DEBUG=true`, `PORT=5432` and `VERSION=1.10` used to emit non-string
  scalars (and `1.10` came back as `1.1`), which also made the output invalid
  as Kubernetes ConfigMap data.
- JSON export keeps the document's key order instead of alphabetising it, which
  had made `--sort` a no-op for that format.
- dotenv export quotes values containing a carriage return; a value ending in
  `\r` was silently truncated when the output was read back.
- The parser accepts `export` followed by tabs or multiple spaces, not just a
  single space.

### Changed
- CI runs on `actions/checkout@v4` and `actions/setup-go@v5` with a current Go
  release, and now also enforces `gofmt -s`. The pinned Go 1.16 and the v2
  actions were long out of support.
- `go.mod` declares Go 1.22 as the minimum toolchain.

## [0.2.0] - 2021-03-19

### Added
- `keys` command to list the keys in a file, with an optional `--sort`.
- `get` command to print the value of a single key.
- `--sort` flag on `merge` and `export` to emit keys alphabetically.

## [0.1.0] - 2021-02-26

### Added
- `merge` command to overlay multiple `.env` files (later files win).
- `diff` command with `--exit-code` for CI use.
- `validate` command to enforce required, non-empty keys.
- `redact` command to mask sensitive values with configurable matchers.
- `export` command to convert between dotenv, shell, JSON and YAML.
