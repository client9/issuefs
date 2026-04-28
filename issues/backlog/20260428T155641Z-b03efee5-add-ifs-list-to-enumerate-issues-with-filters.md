{
  "title": "Add 'ifs list' to enumerate issues with filters",
  "id": "20260428T155641Z-b03efee5",
  "state": "backlog",
  "created": "2026-04-28T15:56:41Z",
  "labels": [
    "feature"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T15:56:41Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Behavior

Mirrors `gh issue list` minus the search-query DSL. Output is a markdown table assembled in-memory and rendered via Glamour (`ascii` style for plaintext / piped output, themed style for TTYs). Columns: short ID, state, title, labels (comma-joined), created date.

Glamour handles wrapping, width-awareness, and color. No per-verb text formatter; no `tablewriter`; no manual title truncation. See the rendering convention in `CLAUDE.md`.

Sample (rendered via `ascii` style):

```
| ID        | State    | Title                                          | Labels          | Created    |
|-----------|----------|------------------------------------------------|-----------------|------------|
| 7b8aca29  | backlog  | Implement reconcile: detect and log hand-edits | feature, design | 2026-04-28 |
| 3953e7f3  | backlog  | Add --editor/-e flag to ifs create             | feature         | 2026-04-28 |
```

## Flags

- `-s, --state {backlog|active|done|all}` — repeatable. Default: `backlog,active` (matches gh's "open" default).
- `-l, --label <name>` — repeatable, AND semantics.
- `-a, --assignee <name>` — exact match.
- `-m, --milestone <name>` — exact match.
- `-L, --limit <int>` — default 30.
- `--sort {created|updated}` — default `created` desc. `updated` = ts of last event.
- `--since <date>` — include only issues created (or updated, when paired with `--sort updated`) on or after the given date. Accepts ISO 8601 (`2026-04-01`, `2026-04-28T12:00:00Z`) and ad-hoc human formats (`"last month"`, `"3 days ago"`, `"yesterday"`) via `github.com/client9/nowandlater@v0.9.0`. No short form — `-s` is taken by `--state`. (Requested as `-s, --since` originally; flagged the conflict.)
- `--format {auto|ansi|plain|json|raw-md}` — default `auto` (ANSI if stdout is a TTY, else `plain`). `json` emits the JSON array described below; `raw-md` emits the assembled markdown table without sending it through Glamour (useful for debugging / piping into other markdown tools).
- `--json` — convenience alias for `--format json`. Emits a JSON array, one object per issue. Object shape mirrors the file frontmatter exactly so `ifs list --json | jq` works without a separate schema.

## Skip

- `--search` / query DSL (out of scope per discussion).
- `--web` (no web UI).
- gh's `-q`/`-t` template flags (`--json | jq` covers it).

## Implementation

- Reuse `store.Resolver.All()` to enumerate matches.
- Add `Match.Load() (*issue.Issue, error)` that reads + parses the file (uses lenient `issue.Parse`).
- Predicate chain for filters: state, label-set, assignee, milestone, since.
- Default sort: by `created` desc; `updated` uses `Events[len-1].Timestamp`.
- **Rendering**: assemble a markdown table from the filtered+sorted matches, then send to Glamour with the style chosen by `--format` (`ascii` for plain, `auto` for ansi). No truncation, no `$COLUMNS` math — Glamour handles it. Cell values must be escaped for `|` and newlines; if `view` (`aee0cbb9`) lands first, reuse its escape helper from `internal/md/`. Otherwise put the helper there during this implementation.
- JSON output bypasses Glamour entirely.

## Dependencies

- `github.com/client9/nowandlater` pinned to `v0.9.0` for `--since` parsing. Wraps both ISO 8601 and ad-hoc human ("last month", "3 days ago") into `time.Time`. Add via `go get github.com/client9/nowandlater@v0.9.0`. Keep usage isolated to a small `parseSince(string) (time.Time, error)` helper in `cmd/list.go` so the dependency can be swapped if the library proves unmaintained.
- `github.com/charmbracelet/glamour` for rendering. Likely already in the module by the time this is implemented (added by `view`, `aee0cbb9`); if not, add it here.
- `github.com/mattn/go-isatty` for TTY detection (`auto` format). Tiny dep.

## Tests

- Filter combinations (state, label intersection, assignee).
- Sort orders.
- JSON output round-trips through `issue.Parse` for at least one record.
- Limit cuts after sorting, not before.
- Default state filter excludes `done`.
- Empty result set: prints nothing in text mode, `[]` in JSON mode, exit 0.
- `--since` accepts both ISO (`2026-04-01`) and ad-hoc (`"last month"`) forms; rejects unparseable input with a clear error.
- `--since` paired with `--sort updated` filters on the latest event timestamp, not `created`.
- `--format plain` output contains zero ANSI escapes (assert byte-level absence of `\x1b[`).
- `--format raw-md` output is valid markdown (parse-able by goldmark/glamour without errors); table cell counts match the filtered match count.
- Title containing `|` is escaped (renders correctly, doesn't split the cell).

## Notes

- Performance: parsing every file on each `list` is fine into the thousands. No cache until measured.
- This is prerequisite for `ifs changelog`, which reuses the filter machinery.
- Implement *after* `view` (`aee0cbb9`). View establishes the `internal/md/` helpers (table assembly, cell escaping); list is the second consumer that confirms the right shape. If list lands first, the helpers get built here and view inherits them.
- **At implementation time, consider adding a `--verbose` mode that includes a one-line summary per issue.** If pursued, adopt the convention "first paragraph under the first body heading is the issue's summary" (codify in SKILL.md as a new filing convention). Most existing issues already follow this informally — a `## Concept`/`## Motivation`/`## Behavior` heading followed by a 1-3 sentence pitch. A small extractor (~15 lines) gets this from the body without schema changes.
