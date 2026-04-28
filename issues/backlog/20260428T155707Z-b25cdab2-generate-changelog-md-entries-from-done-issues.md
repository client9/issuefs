{
  "title": "Generate CHANGELOG.md entries from done issues (keepachangelog format)",
  "id": "20260428T155707Z-b25cdab2",
  "state": "backlog",
  "created": "2026-04-28T15:57:07Z",
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
      "ts": "2026-04-28T15:57:07Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Concept

`ifs changelog` reads done issues and proposes entries for `CHANGELOG.md` in [keepachangelog 1.1.0](https://keepachangelog.com/en/1.1.0/) format. The completion timestamp is the latest `moved → done` event (handles re-opens correctly). Builds on `ifs list`'s filter machinery.

## Modes

- `ifs changelog` — print proposed `## [Unreleased]` additions to stdout. Read-only, default.
- `ifs changelog --update` — write to `CHANGELOG.md`, creating it if absent, inserting under `## [Unreleased]`.
- `ifs changelog --check` — exit non-zero if there are unrecorded done items. CI-friendly: catches "you closed an issue but didn't run --update."
- `ifs changelog --since <date>` — explicit cutoff override.

## Done vs released (decided)

These are distinct events.

- `ifs changelog` proposes additions to `## [Unreleased]`.
- A future `ifs release <version>` (separate issue) renames `## [Unreleased]` to `## [<version>] - <today>` and inserts a fresh empty `## [Unreleased]` above.

Matches keepachangelog's prescribed workflow.

## Idempotency (decided)

Tag generated entries with the short ID as a trailing token:

```
- Add `--editor` flag to `ifs create` (#3953e7f3)
```

On regeneration, skip any issue whose `(#<id>)` already appears in `CHANGELOG.md`. Self-contained — the changelog file is its own state.

Hand-edited entries without the `(#<id>)` tag could duplicate. Acceptable cost; a future `verify`-style pass could warn.

## Open policy questions (need decisions before implementation)

### 1. Inference vs explicit category

Keepachangelog has six categories: Added, Changed, Deprecated, Removed, Fixed, Security. Each done issue maps to exactly one. Three options:

- **Infer from labels** (cheap, requires consistent labeling): `feature` → Added, `bug` → Fixed, `enhancement` → Changed, `breaking` → Removed, `security` → Security, `deprecation` → Deprecated.
- **Explicit field**: new frontmatter key `changelog: added|fixed|...`. Unambiguous, more friction per issue.
- **Hybrid**: infer from labels by default; honor explicit `changelog:` as override.

Recommendation: hybrid. Most issues "just work"; edge cases get explicit control.

### 2. Opt-in vs opt-out for changelog inclusion

Not every done issue belongs in a user-facing changelog (internal refactors, doc fixes, meta-issues).

- **Opt-out**: include all done issues; skip those labeled `internal`/`chore`. Easy to forget the label.
- **Opt-in**: only include issues whose labels (or explicit field) map to a changelog category. Forces intentional decisions; avoids changelog noise.

Recommendation: opt-in (the second). Combines naturally with question 1's hybrid: if an issue has no changelog-relevant label and no explicit field, it's silently skipped.

### 3. Standardized label vocabulary

If we infer from labels, the vocabulary needs to be documented and stable. Proposed set:

- `feature` → Added
- `bug` → Fixed
- `enhancement` → Changed
- `breaking` → Removed (or Changed; needs a decision)
- `security` → Security
- `deprecation` → Deprecated
- `internal` / `chore` / `refactor` / `docs` → not in changelog

Belongs in `CLAUDE.md` and `.claude/skills/issuefs/SKILL.md` once decided so future filings stay consistent.

## Why issue-driven beats commit-driven

Worth recording the rationale since this is a real fork in the road:

- Issue bodies are curated; commit messages often aren't.
- One issue, one entry: a feature spread across 12 commits is one bullet, not 12.
- Labels do categorization for free; conventional-commits requires its own adoption cost.
- Doesn't depend on commit discipline (squash-merge, rebase-heavy workflows still work).

Cost: requires *issue* discipline. Every shippable change needs a done issue. Not a small ask.

## Dependencies

Requires `ifs list` to be implemented first. Reuses its filter predicate chain and `Match.Load()`.

## Anti-goals

- Don't infer from git tags by default. Separate concept; `--since-tag <ref>` could be added later if asked.
- Don't generate from commit messages. The whole point is issue-driven.
- Don't auto-update on every `done` move. Explicit `ifs changelog --update` (or a pre-commit hook the user opts into) keeps the user in control.
