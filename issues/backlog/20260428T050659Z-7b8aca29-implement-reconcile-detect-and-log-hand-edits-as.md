{
  "title": "Implement reconcile: detect and log hand-edits as synthesized events",
  "id": "20260428T050659Z-7b8aca29",
  "state": "backlog",
  "created": "2026-04-28T05:06:59Z",
  "labels": [
    "feature",
    "design"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T05:06:59Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Concept

Add `ifs reconcile` to detect drift between the events log and the current frontmatter/body, then synthesize events to bring the log up to date. This is the snapshot reconciliation pattern from event sourcing: replay events to compute a projected state, compare to actual state, append one event per delta. CLI-correct usage produces no drift, so reconcile is a no-op — that's the dedup mechanism.

Motivation: not everyone uses the CLI. People hand-edit titles, swap labels, mv files between directories. Without reconcile, the events log silently rots into fiction.

## Hard parts (in order of nastiness)

### Timestamps for synthesized events

You don't know when a hand-edit happened. Strategies:
- **Git archaeology**: walk `git log -p` for each drifted field, attribute to the commit's date. Accurate but git-only and slow.
- **Now + marker**: synthesized events get `ts = now` plus an explicit `"synthesized": true` field so future readers know the timestamp is approximate. Always works; honest; sort order will lie for old drift.
- **mtime**: don't bother — unreliable across clones.

Combine: git when available, "now + marker" otherwise. The marker is non-negotiable.

### Vocabulary expansion

State drift is the only thing detectable today. To reconcile other fields, add event types:
- `titled {to}`
- `labeled {added, removed}`
- `assigned {added, removed}`
- `body_edited {hash}` — store SHA-256 of body, not content. Compare against the last `body_edited`/`filed` hash to detect drift. Bounded growth (~64 chars per edit), no prior content stored.
- `milestoned {to}`, `projected {added, removed}`, `templated {to}`

Body via hash is what makes body reconcile work without git or unbounded storage. Requires adding `bodyHash` to the `filed` event so the first projection has something to diff against.

### Directory ↔ frontmatter conflict

If file is in `done/` but frontmatter says `state: backlog`, frontmatter wins. Reconcile fixes the directory location and synthesizes a `moved` event. Matches the implicit assumption already in `verify` (which checks `state` against events, not against parent dir).

### Two modes

- `ifs reconcile` (default): write synthesized events, fix directory if needed, print what was added.
- `ifs reconcile --check`: exit non-zero on drift, change nothing. Pre-commit-hook / CI mode.

`gofmt -l` (check) vs `gofmt -w` (fix) is the precedent. Pre-commit hooks that modify files are controversial; offer both.

## Concerns

- **Convention erosion**: reconcile shifts the events log from "authoritative" to "best-effort." Document the contract change honestly.
- **Synthesized events are second-class**: the `"synthesized": true` marker matters — some consumers (AI summarizers, audit scripts) need to know which events were inferred vs observed.
- **Order ambiguity**: if 5 fields drift at once, pick a stable synthesis order (alphabetical by event type) so timelines are deterministic.
- **Body hash false positives**: whitespace-only changes (trailing newline, line endings) trigger spurious `body_edited` events. Normalize body before hashing (trim trailing whitespace, normalize line endings).
- **No undo**: events are append-only. If reconcile mis-labels an edit (records `titled` when it was a typo fix), the misleading entry stays. Less of a problem when synthesized events are clearly marked.

## Recommended phasing

1. **State-only reconcile first**. Uses existing `moved` event vocabulary, no schema additions. Validates the workflow end-to-end.
2. **Add `body_edited` with hash**. Forces `bodyHash` into `filed` event and exercises the synthesis machinery on a non-trivial type.
3. **Add `titled`/`labeled`/etc one at a time** as the absence is felt. Don't speculate.
4. **Skip git archaeology in v1**. "Now + synthesized marker" is honest enough. Add git mode later if accurate backfilled timestamps matter.
5. **Always emit `--check` alongside the writing mode** so pre-commit hooking works from day one.

## Anti-goal

Don't expand the event vocabulary speculatively. Each event type costs reconcile logic + projection logic + verify rules. Add only when there's a verb that emits the event OR a concrete drift case to demonstrate.
