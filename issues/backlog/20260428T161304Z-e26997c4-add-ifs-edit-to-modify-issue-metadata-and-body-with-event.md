{
  "title": "Add 'ifs edit' to modify issue metadata and body, with event-log entries",
  "id": "20260428T161304Z-e26997c4",
  "state": "backlog",
  "created": "2026-04-28T16:13:04Z",
  "labels": [
    "feature"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T16:13:04Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Behavior

Mirrors [`gh issue edit`](https://cli.github.com/manual/gh_issue_edit) for our metadata model, with one intentional addition: an `--editor`/`-e` flag that opens `$EDITOR` on the body. (gh's edit verb omits this flag — opens an interactive editor only on create. Looks like an oversight on their part. We include it because hand-editing a JSON-frontmatter file just to tweak the body is exactly the friction `--editor` is for. See the dogfooding observation in the conversation around issue `3953e7f3`.)

Every successful edit appends one or more events to the frontmatter `events` array — one per field that actually changed. No-op edits (passing the current value) do not append events.

## Flags (mirror gh, plus `--editor`)

- `-t, --title <string>` — replace title.
- `-b, --body <string>` — replace body.
- `-F, --body-file <path|->` — replace body from file or stdin.
- `-e, --editor` — open `$EDITOR` (fallback `$VISUAL`, fallback `vi`) on a temp file pre-loaded with the current body. Body after editor exits is the new body. Mutually exclusive with `--body`/`--body-file`.
- `--add-assignee <name>` — repeatable.
- `--remove-assignee <name>` — repeatable.
- `--add-label <name>` — repeatable.
- `--remove-label <name>` — repeatable.
- `--add-project <name>` — repeatable.
- `--remove-project <name>` — repeatable.
- `-m, --milestone <name>` — set milestone.
- `--remove-milestone` — clear milestone.

`ifs edit <ref>` with no other flags errors out (matches gh). Don't auto-open the editor; require explicit `-e`.

## Event vocabulary required

This verb forces expansion of the event log. Each event type below must be added to `internal/issue/event.go`, with corresponding `Verify` consistency rules and reconcile projections (see issue `7b8aca29`):

| Field changed       | Event type   | Payload                      |
|---------------------|--------------|------------------------------|
| title               | `titled`     | `{to}`                       |
| body                | `body_edited`| `{hash}` (SHA-256 of body)   |
| labels              | `labeled`    | `{added: [], removed: []}`   |
| assignees           | `assigned`   | `{added: [], removed: []}`   |
| projects            | `projected`  | `{added: [], removed: []}`   |
| milestone (set)     | `milestoned` | `{to}`                       |
| milestone (cleared) | `milestoned` | `{to: ""}`                   |

Body event uses a hash, not content (per the reconcile design rationale: bounded growth, no prior-content storage). Adding `body_edited` requires also adding `bodyHash` to the `filed` event so projection has an initial value to diff against.

## Filename stability (decided)

Changing the title does **not** rename the file. The slug is set at create time and is stable thereafter. Reasons:
- Filenames are referenced by paths in PR descriptions, commit messages, links — renaming silently breaks those.
- Git rename detection works best with stable basenames.
- The `id` (timestamp + 8-hex) is the canonical reference; the slug is browseability only.

A future `ifs rename <ref>` could explicitly opt into a slug update if the divergence becomes painful. Out of scope here.

## Idempotency

Each field gets one event per actual change. If `ifs edit 7b8a --add-label bug` is run twice, the second invocation is a no-op (label already present, no event appended). Same rule for `--remove-label` against an absent label.

## Concurrent edit safety

Read → mutate → write. If two `ifs edit` invocations race, the loser silently overwrites the winner's changes (last-write-wins). Acceptable for a CLI; document in `--help`. A future "lock file or mtime check" could harden this.

## Implementation

- `cmd/edit.go`: flag parsing, no-flag-error, `--editor`/`--body`/`--body-file` mutual exclusion.
- Resolver lookup → `Match.Load()` → mutate Issue struct → diff old vs new → emit events for actual changes → write back via `issue.Marshal` → preserve filename and directory (no move).
- `internal/issue/event.go`: add the event types above plus their constructors.
- `internal/issue/file.go`: extend `validateEvents` projection to cover the new event types so `verify` keeps working.
- Hash helper: `internal/issue/body.go` with `BodyHash(s string) string` returning SHA-256 hex of normalized body (trim trailing whitespace, normalize line endings — same normalization the reconcile design proposes).

## Tests

- Title change appends one `titled` event with new value.
- Body change appends one `body_edited` event with correct hash; no event when unchanged.
- `--add-label` for new label appends `labeled {added: [x]}`; for existing label is a no-op.
- `--add-label x --remove-label y` in one command appends a single combined `labeled` event with both lists populated.
- `--remove-milestone` appends `milestoned {to: ""}`.
- `-e` round-trip: pre-populated temp file content matches current body; modified content becomes new body.
- No-flag invocation exits non-zero with usage hint.
- Filename unchanged after title edit; file remains in same state directory.

## Dependencies

- Editor logic shares plumbing with issue `3953e7f3` (`--editor` on `create`). Implement either first, then reuse the helper for the other.
- Event vocabulary expansion overlaps with issue `7b8aca29` (reconcile). Order doesn't matter; both consume the same `internal/issue/event.go` additions. Whichever lands second updates `Verify`'s projection rules to cover the new events.

## Out of scope

- `--add-project-card`, `--remove-project-card` (gh-specific GitHub Projects v2 wiring).
- Editing the `events` array directly. Events are append-only by contract; mutating history is what reconcile is for.
- `ifs rename` (slug update on title change). Separate issue if/when needed.
