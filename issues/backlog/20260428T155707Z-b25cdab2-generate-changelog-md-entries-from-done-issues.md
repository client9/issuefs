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

## Concept (revised, much simpler)

Add `--template <path-or-@name>` to `ifs list`. Render the filtered+sorted issues through a Go [`text/template`](https://pkg.go.dev/text/template), letting the user (or a bundled template) decide the output format. Ship a `@changelog` bundled template that emits keepachangelog markdown.

The user's workflow becomes:

```bash
ifs list --since 2026-04-01 --state done --template @changelog
```

Cut-and-paste the output into `CHANGELOG.md` or pipe it into `gh release create --notes-file -`. That's the whole feature.

This replaces every previous version of this issue. The point isn't a clever changelog tool; it's that the generic `--template` flag on `list` already does the job once we ship one good template. **The template IS the design.**

## Why this is the right shape

- **Generic.** The same mechanism produces release notes, status reports, sprint summaries, weekly digests, anything. Changelog is the demo, not the only use.
- **No state, no idempotency, no hand-edit contract.** Output is text. User redirects, edits, pastes. Re-running produces a fresh draft from current state.
- **Policy lives in the template.** Decisions like "should `Removed` be its own section or just a `**Breaking:**` prefix on Changed" become template authoring choices, not Go code branches. Want a different changelog format? Write a different template.
- **Categorization is a template func, not a baked-in algorithm.** A `categoryOf` helper in the funcMap lets the template author decide what counts as Added vs Changed.
- **Future-proof.** Want git-tag-aware cutoffs ("everything since v1.0.0")? Add `--since-tag <ref>` later. Want to bundle more templates (`@release-notes`, `@status`)? Add them. The infrastructure exists; each addition is small.

## Flag

```
ifs list ... --template <path-or-@name>
```

- If the value starts with `@`, look up the bundled template by that name (e.g. `@changelog`).
- Otherwise, read the value as a path to a template file.
- Conflicts with other output flags (`--format`, `--json`) — `--template` overrides them. Mutually-exclusive validation in cobra.

## Data passed to the template

```go
type TemplateData struct {
    Issues  []*issue.Issue  // already filtered + sorted per other list flags
    Now     time.Time
    Since   time.Time       // zero if no --since
    State   []string        // the --state filter values
}
```

Issues are passed as already-loaded `*issue.Issue` pointers (full frontmatter accessible). The template can range over them, group, filter further, format anything from the struct.

## Funcmap

Stdlib `text/template` is anemic; ship a small custom funcMap rather than pulling in [Sprig](https://github.com/Masterminds/sprig) (~600KB dep). Add Sprig later only if users ask.

Initial funcMap:

| Name           | Purpose                                                                  |
|----------------|--------------------------------------------------------------------------|
| `categoryOf`   | `(*Issue) string` — returns "Added"/"Changed"/"Fixed"/"Deprecated"/"Removed"/"Security"/"" based on labels and explicit `changelog:` field. |
| `isBreaking`   | `(*Issue) bool` — true if `breaking` label is present.                  |
| `groupBy`      | `(string-key-func, []*Issue) map[string][]*Issue` — generic group helper. |
| `shortID`      | `(*Issue) string` — the 8-hex random suffix.                             |
| `issuePath`    | `(*Issue) string` — relative path to the issue file (for markdown links).|
| `formatDate`   | `(time.Time, layout string) string` — `time.Format` wrapper.             |
| `latestEvent`  | `(*Issue, type string) *Event` — most recent event of a given type (for "completed at" timestamps from `moved → done`). |
| `hasLabel`     | `(*Issue, string) bool` — membership check.                              |
| `dict`         | varargs key/value → map — common pattern for passing structured data into sub-templates. |

Keep this list minimal at first; add as templates need them.

## The bundled `@changelog` template

Ships in `internal/embedded/templates/changelog.md`. Renders keepachangelog 1.1.0 structure with all six standard subheadings (always present, even if empty), imperative-voice bullets, `**Breaking:**` prefix from labels, and a trailing HTML comment listing uncategorized issues. Approximate sketch:

```
{{- $issues := .Issues -}}
## [Unreleased] - {{ formatDate .Now "2006-01-02" }}
{{ range $cat := list "Added" "Changed" "Deprecated" "Removed" "Fixed" "Security" }}
### {{ $cat }}
{{ range $issues -}}
{{- if eq (categoryOf .) $cat -}}
- {{ if isBreaking . }}**Breaking:** {{ end }}{{ .Title }} ([#{{ shortID . }}]({{ issuePath . }}))
{{ end -}}
{{ end -}}
{{ end -}}

<!-- Uncategorized done issues:
{{ range $issues -}}
{{- if eq (categoryOf .) "" }}  - [#{{ shortID . }}]({{ issuePath . }}) {{ .Title }}
{{ end -}}
{{ end -}}
-->
```

(Real template will be polished; the shape matches the previous "draft" sample. Sorting happens in `ifs list` before the template runs — newest first, breaking sorted to top.)

## Label vocabulary (decided, drives `categoryOf`)

Same as the previous version of this issue:

| Label         | Category     |
|---------------|--------------|
| `feature`     | Added        |
| `enhancement` | Changed      |
| `bug`         | Fixed        |
| `security`    | Security     |
| `deprecation` | Deprecated   |
| `removal`     | Removed      |
| `breaking`    | (modifier; sorts first within category) |

Opt-out (return `""` from `categoryOf`): `internal`, `chore`, `refactor`, `docs`, `design`. Multiple-category-labels case: alphabetically-first wins (deterministic).

Optional explicit override: a `changelog: added|changed|fixed|deprecated|removed|security|skip` frontmatter field. When set, `categoryOf` returns that value (or `""` for `skip`) instead of inferring.

To be documented in `CLAUDE.md` and `.claude/skills/issuefs/SKILL.md` when this issue ships.

## Tests

- `--template @changelog` produces output containing all six standard subheadings.
- `feature`-labeled issue appears in Added section.
- `bug`-labeled issue appears in Fixed section.
- `breaking`+`enhancement` produces `**Breaking:**` prefix in Changed.
- Issue with no recognized label appears in trailing HTML comment.
- Explicit `changelog: skip` moves an issue to the trailing comment regardless of labels.
- `--template <path>` reads from a user-provided file.
- `--template @nonexistent` errors with a clear message and lists known bundled templates.
- `--template` and `--format` together error (mutually exclusive).
- Each funcMap function tested directly via small template fixtures.

## Implementation files

- `cmd/list.go` — add `--template` flag; new render path that bypasses Glamour/JSON when set.
- `internal/embedded/templates/` — new directory under the embed package. `//go:embed templates/*.md` exposes them as `fs.FS`.
- `internal/embedded/embed.go` — add `Templates fs.FS` and `LookupTemplate(name string) ([]byte, error)`.
- `internal/issue/category.go` — `Category(*Issue) string` and `IsBreaking(*Issue) bool` (the funcMap functions delegate here so the policy is shared with any future Go code that wants it).
- `cmd/template.go` — render helper. Builds funcMap, parses template, executes against `TemplateData`.

## Dependencies

`ifs list` (`b03efee5`) — done. Reuse its filter/sort machinery; this is just one more output mode.

## Anti-goals

- **No `ifs changelog` verb.** Not needed; the template approach makes a dedicated verb redundant. (If discoverability becomes a concern later, an `ifs changelog` verb that's literally `ifs list --state done --template @changelog` could be a one-line shortcut.)
- **No `--update` mode.** User redirects.
- **No `--check` mode.** No state to check.
- **No idempotency tracking.** Stateless generator.
- **No Sprig dep in v1.** Custom funcMap covers the bundled template's needs. Revisit if user templates grow ambitious.
- **No git-aware features in v1.** No `--since-tag`, no auto-detect last release. Simple text-based `--since <date>`.
- **No author attribution.**
- **No template auto-discovery from filesystem locations** (e.g. `~/.config/ifs/templates/`). Bundled `@<name>` and explicit paths only. Convention can grow if needed.

## Title

The issue title still says "keepachangelog format" but the implementation is now generic-templates-with-a-changelog-template. Title is acceptable for now — closing the issue keeps the changelog-driven story discoverable via `ifs list -l feature -L 1 --since <last release>` etc. Don't rename mid-flight.

## Why issue-driven beats commit-driven (preserved)

- Issue bodies are curated; commit messages often aren't.
- One issue, one entry: a feature spread across 12 commits is one bullet, not 12.
- Labels do categorization for free; conventional-commits requires its own adoption cost.
- Doesn't depend on commit discipline (squash-merge, rebase-heavy workflows still work).

Cost: requires *issue* discipline. Mitigated here by the trailing-comment listing of uncategorized issues — they surface to the editor rather than being silently dropped.
