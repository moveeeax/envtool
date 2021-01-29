# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.1.0] - 2021-02-26

### Added
- `merge` command to overlay multiple `.env` files (later files win).
- `diff` command with `--exit-code` for CI use.
- `validate` command to enforce required, non-empty keys.
- `redact` command to mask sensitive values with configurable matchers.
- `export` command to convert between dotenv, shell, JSON and YAML.
