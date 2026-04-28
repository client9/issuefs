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

Mirrors `gh issue list` minus the search-query DSL. Default output is one line per issue, columns: short ID · state · title (truncated) · labels (comma-joined) · created date.

```
7b8aca29  backlog  Implement reconcile: detect and log…  feature, design  2026-04-28
3953e7f3  backlog  Add --editor/-e flag to ifs create…    feature          2026-04-28
```

## Flags

- `-s, --state {backlog|active|done|all}` — repeatable. Default: `backlog,active` (matches gh's "open" default).
- `-l, --label <name>` — repeatable, AND semantics.
- `-a, --assignee <name>` — exact match.
- `-m, --milestone <name>` — exact match.
- `-L, --limit <int>` — default 30.
- `--sort {created|updated}` — default `created` desc. `updated` = ts of last event.
- `--since <date>` — include only issues created (or updated, when paired with `--sort updated`) on or after the given date. Accepts ISO 8601 (`2026-04-01`, `2026-04-28T12:00:00Z`) and ad-hoc human formats (`"last month"`, `"3 days ago"`, `"yesterday"`) via `github.com/client9/nowandlater@v0.9.0`. No short form — `-s` is taken by `--state`. (Requested as `-s, --since` originally; flagged the conflict.)
- `--json` — emit a JSON array, one object per issue. Object shape mirrors the file frontmatter exactly so `ifs list --json | jq` works without a separate schema.

## Skip

- `--search` / query DSL (out of scope per discussion).
- `--web` (no web UI).
- gh's `-q`/`-t` template flags (`--json | jq` covers it).

## Implementation

- Reuse `store.Resolver.All()` to enumerate matches.
- Add `Match.Load() (*issue.Issue, error)` that reads + parses the file (uses lenient `issue.Parse`).
- Predicate chain for filters: state, label-set, assignee, milestone, since.
- Two renderers: text (tab-aligned, terminal-width-aware title truncation) and JSON.
- Default sort: by `created` desc; `updated` uses `Events[len-1].Timestamp`.
- Truncate title to fit `$COLUMNS` (fallback 80) minus other column widths; suffix with `…`.

## Dependencies

- `github.com/client9/nowandlater` pinned to `v0.9.0` for `--since` parsing. Wraps both ISO 8601 and ad-hoc human ("last month", "3 days ago") into `time.Time`. Add via `go get github.com/client9/nowandlater@v0.9.0`. Keep usage isolated to a small `parseSince(string) (time.Time, error)` helper in `cmd/list.go` so the dependency can be swapped if the library proves unmaintained.

## Tests

- Filter combinations (state, label intersection, assignee).
- Sort orders.
- JSON output round-trips through `issue.Parse` for at least one record.
- Limit cuts after sorting, not before.
- Default state filter excludes `done`.
- Empty result set: prints nothing in text mode, `[]` in JSON mode, exit 0.
- `--since` accepts both ISO (`2026-04-01`) and ad-hoc (`"last month"`) forms; rejects unparseable input with a clear error.
- `--since` paired with `--sort updated` filters on the latest event timestamp, not `created`.

## Notes

- Performance: parsing every file on each `list` is fine into the thousands. No cache until measured.
- This is prerequisite for `ifs changelog`, which reuses the filter machinery.
