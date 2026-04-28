{
  "title": "Add 'relations' frontmatter field with link/unlink verbs (v1: symmetric 'related')",
  "id": "20260428T165215Z-2a0272df",
  "state": "backlog",
  "created": "2026-04-28T16:52:15Z",
  "labels": [
    "feature"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T16:52:15Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Behavior

Issues gain a `relations` field in the frontmatter that holds typed cross-references to other issues. v1 ships with a single relationship type, `related` (symmetric). Two new verbs:

- `ifs link <ref-a> <ref-b>` — create a `related` link between two issues. Bidirectional: writes both files, appends a `linked` event to both event logs.
- `ifs unlink <ref-a> <ref-b>` — symmetric removal. Both files, both event logs.

Body parsing of `#<short>` mentions is **out of scope** (would require markdown-AST traversal to skip code blocks/quotes; not worth the parser surface for v1).

## Frontmatter shape (typed-ready from day one)

```json
{
  ...,
  "relations": {
    "related": ["20260428T161304Z-e26997c4", "20260428T155641Z-b03efee5"]
  }
}
```

Nested map shape (`relations: {<type>: [<full-id>...]}`) rather than flat (`related: [...]`). Costs one extra level of nesting now; eliminates a frontmatter migration when typed relations (`blocks`, `parent`, etc.) ship later. v1 only emits the `related` key. Empty state: `"relations": {"related": []}` (always-present full field set per existing convention).

## Inverse-type registry

`internal/issue/relations.go` defines:

```go
var inverseType = map[string]string{
    "related": "related", // symmetric (self-inverse)
    // future: "blocks": "blocked-by", "blocked-by": "blocks", etc.
}
```

`link`/`unlink` consume this map without per-type branching. Adding a new typed relation later is one map entry plus a `--type` flag on the verbs (out of scope for v1, but the plumbing supports it).

## Storage: full IDs, not short refs

`relations.related` stores full IDs (`20260428T161304Z-e26997c4`), not short refs (`e26997c4`). Reasoning:
- Storage should be unambiguous; presentation can use shorts via the resolver.
- Robust against the (theoretical) day when 8-hex prefixes start colliding.
- Renders fine in `ifs show` because the resolver can map ID → short for display.

Verbs accept any ref form the resolver accepts (path, full filename, full ID, short prefix); the link verb resolves to the full ID before storing.

## Event vocabulary (additions)

| Event type  | Payload                | Emitted by         |
|-------------|------------------------|--------------------|
| `linked`    | `{to: "<full-id>"}`    | `ifs link` (both sides) |
| `unlinked`  | `{to: "<full-id>"}`    | `ifs unlink` (both sides) |

The receiving issue's event reads "this issue was linked to X" — the `to` field always names the *other* issue. Symmetric type means both sides record the same event type with their counterpart's ID in `to`.

`Verify`'s `validateEvents` projection rules don't need updating for `linked`/`unlinked` (these don't affect any frontmatter scalar); but `Verify` does gain the relations consistency check below.

## `Verify` additions

- Every full ID in `relations.<type>` must point to an existing issue file under `issues/{backlog,active,done}/`. Unresolvable references error.
- For each entry in `A.relations.<type>`, the file at the referenced ID must contain `A`'s full ID in its `relations.<inverse-of-type>` array. Asymmetric relations error.
- Self-link (`A.relations.related` contains A's own ID) errors.

Implementation: after parsing each file, build a global index of `id → relations`, then walk and check both directions.

## `link` algorithm

1. Resolve `ref-a` and `ref-b` via `store.Resolver` (each call may error if unresolvable or ambiguous).
2. If `a.id == b.id`: error ("cannot link an issue to itself").
3. Read both files via `issue.Parse`.
4. If `b.id` already in `a.relations.related` AND `a.id` already in `b.relations.related`: no-op (idempotent), exit 0 silently.
5. Mutate: add `b.id` to `a.relations.related`; add `a.id` to `b.relations.related` (deduplicate; sort alphabetically for stable on-disk order).
6. Append event to `a`: `linked {to: b.id, ts: now}`.
7. Append event to `b`: `linked {to: a.id, ts: now}`.
8. Marshal both. Write `a` first, then `b`.
9. If write of `b` fails after `a` succeeds: print a clear error including both paths and the partial state. Do not attempt rollback. (Acceptable v1 cost; reconcile will eventually be able to repair.)

`unlink` is the same shape with removal and `unlinked` events. If neither side currently has the link: no-op.

## Failure modes (decided)

- **Unresolvable target ref**: error before any write.
- **Ambiguous target ref**: error before any write (resolver already does this).
- **Self-link**: error.
- **Already linked** (both sides): silent no-op, exit 0.
- **Half-linked state on disk** (one side has it, other doesn't): treated as "already linked" if both writes would converge to the same state; otherwise treated as needs-completion (write the missing side, append the missing event). Equivalent to a partial-link recovery.
- **Source-write succeeds, target-write fails**: error with both paths printed; no rollback. Document the limitation in `--help`.

## Concurrent safety

Read → mutate → write. Two concurrent `ifs link` invocations on overlapping issues can interleave (last-write-wins per file). Acceptable for v1; document. A future file-locking pass could harden.

## CLI shape

```
ifs link <ref-a> <ref-b>
ifs unlink <ref-a> <ref-b>
```

No flags in v1. The verbs are positional-only. `--type` would be a future addition when non-`related` relationship types ship; until then it would be confusing dead UI.

## Implementation files

- `internal/issue/issue.go` — add `Relations Relations` field to Issue (always-emitted; `New()` initializes `Relations.Related = []string{}`).
- `internal/issue/relations.go` — `Relations` struct (currently just `Related []string` mapped to JSON `"related"`; nested under `relations`); inverse-type registry; helpers (`HasLink`, `AddLink`, `RemoveLink`).
- `internal/issue/event.go` — `EventLinked = "linked"`, `EventUnlinked = "unlinked"`, constructors `NewLinked(ts, toID)` and `NewUnlinked(ts, toID)`.
- `internal/issue/file.go` — extend `validate` with relations-consistency rules (resolves IDs, checks symmetry).
- `cmd/link.go` — `newLink()` and `newUnlink()` wired into `cmd/root.go`.

## Tests

- Link two issues → both files have each other in `relations.related`; both have `linked` events with correct `to`; both verify clean.
- Unlink → both files have it removed; both have `unlinked` events; both verify clean.
- Idempotent link (link twice) → second is silent no-op, no extra event.
- Idempotent unlink (unlink absent link) → silent no-op, no event.
- Self-link errors before any write.
- Unresolvable target errors before any write.
- Ambiguous target errors before any write.
- `Verify` catches: asymmetric relation, self-link in stored data, dangling reference.
- Half-linked state recovery: pre-create a half-linked pair manually, run `ifs link`, both sides converge with a single new event on the missing side.

## Cross-references

- Builds on the resolver added with `b03efee5` plumbing (full prefix lookup is needed both for the verb arg parsing and for `Verify`'s relations-existence check). No blocker — resolver already exists.
- Shares the inverse-type registry concept with future typed-relation work (sub-issues, blocks/blocked-by). v1 populates only `related <-> related`.
- Will benefit from `ifs reconcile` (`7b8aca29`) once that lands: half-linked drift becomes a reconcile-fixable case rather than a manual-repair case.
- `ifs show` (not yet filed) will render `relations.related` as a "See also" section when displayed.

## Out of scope (deferred)

- Body parsing for `#<short>` mentions. Would require markdown-AST traversal to skip code blocks, inline code, blockquotes, etc. — not worth the parser surface for v1. Revisit as opt-in `ifs reconcile --extract-mentions`.
- `ifs verify --check-mentions`. Same reasoning.
- Typed relations beyond `related`: `blocks`/`blocked-by`, `parent`/`child`, `duplicates`/`duplicate-of`, `supersedes`/`superseded-by`. The schema and registry are designed to absorb these; verbs will gain a `--type` flag when they ship.
- Operational semantics that consume relations (e.g., `ifs done` warning if the issue has open `blocks` references). Belongs to whichever typed-relation issue introduces `blocks`.
- `--link <ref>` flag on `create`/`edit` for one-shot link establishment at issue creation time. Logical follow-up; small change once the verbs exist.
- File-locking for concurrent safety.

## Anti-goals

- No body parsing in v1, regardless of how convenient it would be.
- No silent rollback on partial-write failure (error, surface the paths, let user fix).
- No per-type verbs (`ifs block`, `ifs subissue`) until at least two typed relations exist and the right factoring is clearer.
