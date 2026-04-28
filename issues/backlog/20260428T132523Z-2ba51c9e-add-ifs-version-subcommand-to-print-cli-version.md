{
  "title": "Add 'ifs version' subcommand to print CLI version",
  "id": "20260428T132523Z-2ba51c9e",
  "state": "backlog",
  "created": "2026-04-28T13:25:23Z",
  "labels": [
    "feature"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T13:25:23Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Motivation

No way today to know which `ifs` build is installed. Useful for bug reports, for users with multiple installs on PATH, and for CI to assert a minimum version.

## Design

- New verb: `ifs version`. Prints a single line to stdout: `ifs <version> (<commit>, <date>)`.
  - Example: `ifs 0.2.0 (a1b2c3d, 2026-05-01)`
- For `--short` flag (or `ifs version --short`): prints just the bare semver, one line, no parens. Scriptable.
- Exit code 0 on success.

## Where the version comes from

Two complementary sources, in priority order:

1. **Build-time injection** via `-ldflags`: `go build -ldflags "-X github.com/nickg/issuefs/cmd.version=0.2.0 -X .commit=$(git rev-parse --short HEAD) -X .date=$(date -u +%Y-%m-%d)"`. Authoritative when set.
2. **Runtime fallback** via `debug.ReadBuildInfo()`: works for `go install github.com/nickg/issuefs@latest` users, returns the module version (`v0.2.0`) and the VCS revision/time stamped by `go install`. Use this when ldflags vars are empty.
3. **Last resort**: print `ifs (devel)` if neither is populated (typical for `go run .`).

## Implementation notes

- `cmd/version.go`: declare three package-level vars (`version`, `commit`, `date`), wire the verb, prefer ldflags then `debug.ReadBuildInfo`.
- Add a `Makefile` target or shell snippet so `make build` produces a stamped binary; document in `CLAUDE.md` so future sessions don't ship unstamped builds.

## Out of scope

- JSON output (`--json`). Add only if a consumer asks for it.
- Update-check ("a newer version is available"). Surveillance-y; skip.
