{
  "title": "Make 'ifs move' synchronize directory and frontmatter (idempotent, self-healing)",
  "id": "20260428T195733Z-0f9a1b5f",
  "state": "done",
  "created": "2026-04-28T19:57:33Z",
  "labels": [
    "bug"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T19:57:33Z",
      "type": "filed",
      "to": "backlog"
    },
    {
      "ts": "2026-04-28T21:12:32Z",
      "type": "moved",
      "from": "backlog",
      "to": "active"
    },
    {
      "ts": "2026-04-28T21:14:40Z",
      "type": "moved",
      "from": "active",
      "to": "done"
    }
  ]
}

## Contract

After `ifs move <ref> <state>` succeeds, the file's location AND its frontmatter `state` field both equal `<state>`. Always. Regardless of where they were before.

This makes the verb idempotent and self-healing, and supports the workflow of `git mv` first (to preserve git rename detection) followed by `ifs move` to sync metadata.

## Current behavior is wrong

The same-state check uses the file's physical location, not the frontmatter:

```go
// cmd/move.go
if m.State == target {  // m.State comes from the directory
    return nil          // no-op — frontmatter NOT updated
}
```

This produces silent inconsistency in two scenarios:

### Scenario 1: `git mv` leaves frontmatter stale

```bash
git mv issues/backlog/foo.md issues/active/foo.md
ifs move foo active
# → silent no-op
# → frontmatter still says "state": "backlog"
# → no `moved` event appended
# → directory and frontmatter disagree, verify still passes
```

### Scenario 2: Wrong-target after partial `git mv`

```bash
git mv issues/backlog/foo.md issues/done/foo.md
ifs move foo active
# → m.State = "done" (directory), iss.State = "backlog" (frontmatter)
# → not same-state, proceeds
# → appends event: `moved from "done" to "active"`
# → but no `done` transition ever happened — the event lies
```

`ifs verify` doesn't catch either case because verify only checks frontmatter ↔ events consistency, not directory ↔ frontmatter.

## Proposed semantics

Truth table for `ifs move <ref> <state>`:

| Frontmatter `state` | Directory | Target | Action |
|---|---|---|---|
| backlog | backlog | active | rename `backlog/`→`active/`; set state=active; append `moved backlog→active` |
| backlog | active   | active | (no rename needed); set state=active; append `moved backlog→active` |
| backlog | done     | active | rename `done/`→`active/`; set state=active; append `moved backlog→active` |
| active  | active   | active | true no-op (already in sync); no event |
| active  | backlog  | active | rename `backlog/`→`active/`; (no fm change); append `moved active→active`? — see below |

The last row is the awkward edge case: directory is wrong but frontmatter is already correct. Two options:
- **A.** Move file, no event (it was always *supposed* to be in active per frontmatter; the directory was the bug, no transition happened).
- **B.** Move file, append a `moved active→active` event documenting the repair.

(A) is cleaner — events should record state *transitions*, and there was none. The repair is invisible at the event level. Recommend (A).

## Key changes

1. **Same-state check uses `iss.State` (frontmatter), not `m.State` (directory).**
2. **Event `from` field is `iss.State` (frontmatter)**, not `m.State`. Records what the issue *thought* its state was, not what filesystem said.
3. **Rename is conditional**: only `os.Rename` when `m.State != target`. If the file is already in the target dir, skip the rename.
4. **Frontmatter update is conditional**: only mutate `iss.State` and append `moved` event when `iss.State != target`. The "directory wrong, frontmatter right" repair case (last row of table) updates only the file location.
5. The "true no-op" stderr message ("X is already in Y") fires only when both directory AND frontmatter already equal target.

## Implementation

`cmd/move.go`:

```go
iss, err := readIssue(m.Path)
if err != nil { return err }

needFmUpdate := iss.State != target
needRename := m.State != target

if !needFmUpdate && !needRename {
    fmt.Fprintf(stderr, "%s is already in %s\n", m.Short, target)
    fmt.Fprintln(stdout, m.Path)
    return nil
}

if needFmUpdate {
    iss.Events = append(iss.Events, issue.NewMoved(now, iss.State, target))
    iss.State = target
    data, err := issue.Marshal(iss)
    if err != nil { return err }
    if err := os.WriteFile(m.Path, data, 0o644); err != nil { return err }
}

if needRename {
    targetDir, err := store.EnsureSubdir(root, target)
    if err != nil { return err }
    newPath := filepath.Join(targetDir, m.Name)
    if _, err := os.Stat(newPath); err == nil {
        return fmt.Errorf("destination already exists: %s", newPath)
    }
    if err := os.Rename(m.Path, newPath); err != nil { return err }
    fmt.Fprintln(stdout, newPath)
} else {
    fmt.Fprintln(stdout, m.Path)
}
```

Order matters: write-then-rename (existing convention) preserved when both happen. The conditional makes each independent.

## Tests

Each row of the truth table:

- **Standard move** (fm and dir both match source state, target differs): file renamed, fm updated, event appended.
- **`git mv`-first repair** (fm stale, dir matches target): file stays put, fm updated, event appended.
- **Wrong-target after partial `git mv`** (fm and dir disagree, target is third state): file moves to target, fm updates from `iss.State` to target, event records `iss.State→target` (not `dir→target`).
- **True no-op** (fm and dir both equal target): no rename, no event, stderr notice.
- **Directory-only repair** (fm matches target, dir wrong): file moves to match fm; no event (no state transition happened); no stderr notice (this is silent fixing).

Plus existing tests should still pass; the new logic is a superset.

## Cross-references

- This bug overlaps with `7b8aca29` (reconcile). Reconcile will eventually handle directory ↔ frontmatter drift across all issues; `ifs move` should still do the right thing for the single issue it's invoked on.
- Once both ship, the workflow is:
  - `ifs reconcile`: scan everything, fix all drift.
  - `ifs move xxx STATE`: targeted, idempotent, self-healing for one issue.
- They cooperate cleanly and aren't redundant.

## Out of scope

- Reconciling other drift types (title, labels, body) — that's `7b8aca29`.
- Refusing to act on drift instead of healing it. Not a useful default; users would always pass `--fix` and we'd have implemented the wrong thing.

## Resolution

Implemented as designed. All five truth-table rows behave correctly; the `git mv` → `ifs move` workflow now syncs the frontmatter and appends an honest `moved` event.

What landed:
- `cmd/move.go` — `runMove` now computes `needFmUpdate := iss.State != target` and `needRename := m.State != target` independently. True no-op (both false) prints the "already in" stderr notice and exits. Frontmatter update appends `moved` event with `from: iss.State` (not `m.State`) — records the actual transition the issue thought it was making, not whatever the directory happened to be. Rename is skipped when the file is already in the target directory. The "directory wrong, frontmatter right" repair case (Row 5 of the table) does the rename and appends no event — a directory bug isn't a state transition. `--help` long description rewritten to state the new contract explicitly.
- `cmd/move_test.go` — 6 new cases covering each truth-table row plus a verify-after-sync sanity check. `gitMove` test helper simulates `git mv` (file moves, frontmatter doesn't); `parseIssueFile` and `countMovedEvents` helpers keep assertions tight.

Smoke verified end-to-end: `git mv backlog/X.md active/X.md && ifs move <short> active` correctly sets `state: active` in frontmatter and appends `moved backlog→active` (correct `from`).

Tests: all packages green.

Deviations from the original plan:
- None. The implementation followed the proposed code sketch nearly verbatim. The one cosmetic change: tracked the final output path in a `finalPath` variable so the success-line print works for both the "renamed" and "fm-update-only" paths without duplicating the `Fprintln`.

Follow-ups discovered:
- This bug is now closed, but a related class remains: **`ifs verify` doesn't catch directory ↔ frontmatter mismatches.** A file in `active/` with `state: backlog` in its frontmatter passes verify today (verify only checks frontmatter ↔ events). Worth a separate small bug to add a "directory matches frontmatter state" check to `Verify`. Not filed yet — file if/when someone hits it again.
- `ifs reconcile` (`7b8aca29`), when implemented, will also handle this drift (across all issues at once). The verify check above is the cheaper, in-the-moment version.
- The `Row 5` no-event-on-directory-repair semantics are a small but principled choice: events should record state *transitions*, not *repairs*. If we ever want to record repairs, an `event.Type == "repaired"` (with `field: "directory"`) would be the right shape, but adding that speculatively is YAGNI. Note here in case it ever comes up.
