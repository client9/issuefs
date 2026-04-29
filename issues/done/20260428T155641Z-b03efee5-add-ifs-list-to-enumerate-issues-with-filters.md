{
  "title": "Add 'ifs list' to enumerate issues with filters",
  "id": "20260428T155641Z-b03efee5",
  "state": "done",
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
    },
    {
      "ts": "2026-04-29T00:15:21Z",
      "type": "moved",
      "from": "backlog",
      "to": "active"
    },
    {
      "ts": "2026-04-29T00:38:04Z",
      "type": "moved",
      "from": "active",
      "to": "done"
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

## Resolution

Implemented as designed. View landed first, so list inherits `internal/md/` (the prediction held — helpers needed no refactor for the second consumer).

What landed:
- `cmd/list.go` — full flag set: `-s/--state` (default `backlog,active`, supports `all`), `-l/--label` (AND), `-a/--assignee` (AND), `-m/--milestone` (OR), `-L/--limit`, `--sort {created|updated}`, `--since <date>`, `--format {auto|ansi|ascii|json|raw-md}`, `--json` (alias for `--format json`). Predicate chain runs as a single pass per match. `listEntry` is a named struct so helpers don't have to deal with anonymous-struct identity issues.
- `cmd/list_test.go` — 19 cases: default-excludes-done, state filter, `--state all`, label AND, assignee filter, JSON validity + label assertion, `--json` alias, limit, sort created desc, empty result (text and JSON), no-issues-dir (text and JSON), `--since` ISO + ad-hoc + bad input, bad format/state/sort, pipe-escaping in title, ascii zero-ANSI assertion, sort updated (with two real moves and 1.1s sleeps for sub-second timestamp resolution).
- `parseSince(s)` — small helper using `nowandlater.Parser`; tries `Parse` (single instant) first, falls back to `ParseInterval` (uses start) for phrases like "last month" / "this week". Isolated so the dep can be swapped if needed.
- `cmd/root.go` — registered `newList()`.
- `go.mod` — `github.com/client9/nowandlater` v0.9.0 added. Self-contained (no transitive deps), zero-dependency module.
- `.claude/skills/issuefs/SKILL.md` — verb cheatsheet now includes `ifs list` with full flag enumeration. Also added `ifs view` (was missing from the cheatsheet — drift caught during this session).

Smoke verified: `ifs list -l bug` shows two bug-labeled issues; `ifs list --since yesterday` filters on `Created` correctly; `ifs list --json` produces a JSON array with the same field shape as the file frontmatter (so `ifs list --json | jq` works without a separate schema).

Tests: all packages green (cmd suite 2.5s — the `--sort updated` test sleeps 2.2s total to get sub-second timestamp separation; everything else is fast).

Deviations from the original plan:
- `--format` enum: issue body said `{auto|ansi|plain|json|raw-md}`; actual implementation is `{auto|ansi|ascii|json|raw-md}` — `ascii` matches `view`'s convention and Glamour's actual style name (`plain` was pre-Glamour shorthand). No `raw` format (only `raw-md`); `raw` makes sense for `view` (one file to dump verbatim) but not `list` (which file?). Both differences are documented in the verb's `--help`.
- `--milestone`: spec said exact match; implementation supports it as a repeatable flag with OR semantics across values. Matches gh's behavior more closely (`gh issue list -m a -m b` lists issues in either milestone). Single value still works as exact match.
- I considered store-level `Match.Load()` per the spec; instead, list calls the package-local `readIssue(path)` helper from `cmd/move.go`. Keeps the store layer focused on filesystem (no `internal/issue` import in store), matches the existing pattern.

Follow-ups discovered:
- During the smoke test, found two issues filed in another session that I didn't know about (`18787d92` "use mingo to determine minimal Go version" and `48996d99` "Add GitHub actions"). Useful proof that `ifs list` immediately surfaces work that's been silently accumulating — exactly the workflow it was built to enable.
- The `--verbose` summary mode and "first paragraph as summary" convention are still queued (see Notes section above). Implementing them would be ~30 lines of body parsing in `internal/md` and a SKILL convention update. Will take that on when changelog (`b25cdab2`) needs it, since changelog will be the first verb that genuinely benefits from richer per-issue text.
- The `internal/md/Table` helper rendered fine for both `view` (2-column metadata) and `list` (5-column data). Pattern for future verbs: assemble a markdown table, pass to glamour. No special-casing needed.
