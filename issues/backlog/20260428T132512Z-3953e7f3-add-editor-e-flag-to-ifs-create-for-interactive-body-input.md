{
  "title": "Add --editor/-e flag to 'ifs create' for interactive body input",
  "id": "20260428T132512Z-3953e7f3",
  "state": "backlog",
  "created": "2026-04-28T13:25:12Z",
  "labels": [
    "feature"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T13:25:12Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Motivation

Originally omitted on the rationale that interactive flags don't fit a script-friendly tool. In practice, when using `ifs` from a terminal (not via an AI session), composing a multi-paragraph body inline with `--body "..."` or via heredoc is awkward. `gh issue create -e` opens `$EDITOR` on a temp file; we should match that behavior.

## Design

- New flag: `-e, --editor` — opens `$EDITOR` (fallback `$VISUAL`, fallback `vi`) on a temp file. Uses the file's contents as the body after the editor exits.
- Mutually exclusive with `--body` and `--body-file` — same as those two are with each other.
- Temp file: `os.CreateTemp("", "ifs-*.md")`; delete on exit (success or failure). Pre-populate with a single blank line so syntax highlighting kicks in.
- Editor not found / non-zero exit: error out, do not create the issue.
- Empty body after edit: allowed (consistent with `--body ""` today).
- Title is still required via `-t` — don't try to harvest title from a "first line of body" convention; that's surprising and inconsistent with the rest of the tool.

## Implementation notes

- `cmd/create.go`: add the flag, route through a small `openEditor() (string, error)` helper.
- `MarkFlagsMutuallyExclusive("body", "body-file", "editor")` covers the three-way exclusion.
- Tests: skip the actual editor invocation in unit tests (no TTY); test the helper by setting `EDITOR=true` (writes nothing, exits 0) and `EDITOR=false` (exits non-zero).

## Out of scope

- `--web` flag from `gh` — no web UI here.
- `--recover` — gh-specific recovery from a crashed editor session; not worth the complexity yet.
