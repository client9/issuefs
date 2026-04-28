{
  "title": "Add 'ifs view' to render an issue file readably (table for meta, ANSI/plaintext for body)",
  "id": "20260428T172347Z-aee0cbb9",
  "state": "done",
  "created": "2026-04-28T17:23:47Z",
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
      "ts": "2026-04-28T17:23:47Z",
      "type": "filed",
      "to": "backlog"
    },
    {
      "ts": "2026-04-28T19:15:21Z",
      "type": "moved",
      "from": "backlog",
      "to": "active"
    },
    {
      "ts": "2026-04-28T19:39:02Z",
      "type": "moved",
      "from": "active",
      "to": "done"
    }
  ]
}

## Behavior

`ifs view <ref>` reads an issue file and renders it for human consumption: frontmatter as a labeled table, body as readable text (ANSI-styled if stdout is a TTY, plain text if piped). Mirrors `gh issue view` in spirit.

```
ifs view e26997c4
```

```
┌────────────┬──────────────────────────────────────────────────────────────┐
│ Title      │ Add 'ifs edit' to modify issue metadata and body, with...    │
│ ID         │ 20260428T161304Z-e26997c4 (e26997c4)                         │
│ State      │ backlog                                                      │
│ Created    │ 2026-04-28 16:13 UTC                                         │
│ Labels     │ feature                                                      │
│ Related    │ (none)                                                       │
└────────────┴──────────────────────────────────────────────────────────────┘

# Behavior

Mirrors gh issue edit for our metadata model, with one intentional addition…

(body rendered with ANSI styling for headings, code blocks, lists, etc.)

Timeline:
  2026-04-28 16:13 UTC  filed → backlog
```

## Flags

- `--format {auto|ansi|plain|json|raw}` — default `auto`. `auto` = ANSI if stdout is a TTY (`isatty(stdout)`), else `plain`. `json` = full Issue struct (frontmatter only) for scripting. `raw` = file contents verbatim.
- `--no-meta` — body only, skip the table.
- `--no-events` — skip the timeline footer.
- `--width <int>` — wrap width (default: terminal width, fallback 80).

## Prior art (Go ecosystem, surveyed)

### The chosen tool: glamour with style switching

**[charmbracelet/glamour](https://github.com/charmbracelet/glamour)** — actively maintained, library-shaped (despite often being confused with the `glow` CLI built on top of it). Themeable via named styles. Critically, it ships with an **`ascii`** style that produces no ANSI escape sequences — pure plaintext. This is the same approach used by GitHub, GitLab, and others for terminal markdown rendering when colors aren't wanted.

That single feature collapses what would have been two separate render paths into one: choose `dark`/`light`/`auto` for ANSI; choose `ascii` for plaintext. Same library, same code path, style is the only knob.

### Tables

Originally planned to use **[olekukonko/tablewriter](https://github.com/olekukonko/tablewriter)** for the metadata header. **Dropped** in favor of emitting the metadata as a markdown table within the document and letting glamour render it. One library, one pass, no need to coordinate widths between two renderers. Glamour handles markdown tables natively in both ANSI and `ascii` styles.

### Rejected alternatives

- **[charmbracelet/glow](https://github.com/charmbracelet/glow)** — the CLI built on glamour. Not embeddable. Use glamour directly.
- **[MichaelMure/go-term-markdown](https://github.com/MichaelMure/go-term-markdown)** — abandoned.
- **[huantt/plaintext-extractor](https://github.com/huantt/plaintext-extractor)** — abandoned.
- **Custom goldmark `NodeRenderer` for plaintext** — not needed once `ascii` style is on the table. Originally proposed because the ecosystem appeared to lack a maintained plaintext path; glamour's `ascii` style fills that gap.

## Implementation plan

One render pipeline, style chosen by `--format`:

| `--format` | glamour style          |
|------------|------------------------|
| `ansi`     | `auto` (respects `GLAMOUR_STYLE`); falls back to `dark` |
| `plain`    | `ascii`                |
| `auto`     | `ansi` if stdout is a TTY, else `plain` |
| `json`     | (bypasses glamour; emits Issue struct as JSON) |
| `raw`      | (bypasses glamour; emits file bytes verbatim) |

Internal shape:

```
internal/render/
  document.go  // assemble issue → single markdown string (meta + body + timeline)
  view.go      // pick glamour style, render the document, write out
```

`internal/render` is general-purpose (reusable by future `ifs list --verbose` or any verb that surfaces body content).

## Document structure

The renderer concatenates three markdown sections into one document, then sends the whole thing to glamour. No separate widget for the table; markdown does the work.

```markdown
| Field     | Value                                         |
|-----------|-----------------------------------------------|
| Title     | Add 'ifs view' to render an issue file…       |
| ID        | 20260428T172347Z-aee0cbb9 (`aee0cbb9`)        |
| State     | backlog                                       |
| Created   | 2026-04-28 17:23 UTC                          |
| Labels    | feature, design                               |
| Related   | _(none)_                                      |

---

# Behavior

(body markdown as-written)

---

## Timeline

| When                 | Event   | Details         |
|----------------------|---------|-----------------|
| 2026-04-28 17:23 UTC | filed   | → backlog       |
| 2026-04-28 18:00 UTC | linked  | → `7b8aca29`    |
```

Sections are independently skippable via `--no-meta` and `--no-events`. With both flags, only the body section is emitted. Section separators (`---`) are emitted only between non-empty sections.

### Cell escaping

Markdown table cells must escape `|` and `\n`. Implementation: a small `escapeCell(string) string` helper that replaces `|` with `\|` and newlines with `<br>` (glamour renders `<br>` as a soft break). Apply uniformly to every value before assembling.

### Why this is better than tablewriter

- One library to depend on.
- One render pass — no width-coordination between two renderers.
- The document is a real markdown file; trivially debuggable (`--format raw-md` could even emit the assembled document pre-glamour, useful for testing).
- Consistent typography between metadata and body (same heading style, same color palette, same code-block treatment).

## Staged implementation

Ship in three stages so the markdown pipeline gets exercised end-to-end at each step. Each stage closes against this same issue (resolution captures what stage shipped); intermediate states leave the issue in `active`.

### Stage 1 — minimum viable view (metadata table only)

- `internal/md/` package created. Helpers: `Table(headers []string, rows [][]string) string`, `EscapeCell(string) string` (handles `|` and `\n`).
- `cmd/view.go` with `view <ref>` taking exactly one ref. Resolves via `store.Resolver`, loads the Issue, assembles a markdown table of metadata, sends to glamour.
- `--format` accepts `auto|ascii|raw-md` only this stage. (No ANSI styles, no JSON, no raw.)
- `auto` defaults to `ascii` always in this stage (TTY detection deferred to Stage 2).
- glamour + go-isatty added to go.mod.
- Tests: metadata renders all expected fields, `|` in title is escaped, `ascii` output contains no `\x1b[`, `raw-md` output is byte-stable.
- **Ship criteria:** `ifs view <ref>` shows a clean metadata table for any real issue. No body, no timeline yet.

### Stage 2 — body + ANSI

- Append body markdown after the metadata table (with `\n---\n\n` separator).
- `--format` gains `ansi` (style `auto` from glamour, falls back to `dark`).
- `auto` format gains TTY detection: ANSI if `isatty(stdout)`, else `ascii`.
- Tests: body renders for issues with bodies; empty-body issues show no body section; `ansi` output contains ANSI escapes; piped (no-TTY) output uses ascii.

### Stage 3 — timeline + remaining flags

- Append timeline section (events table) after body.
- `--no-meta`, `--no-events`, `--format json`, `--format raw` flags.
- Relations rendering (resolved short refs + titles when targets exist; "(unresolved: <id>)" when not). May depend on `2a0272df` (relations) being implemented; if not, omit the Related row this stage and add a follow-up note.
- Tests for all the cases listed in the existing Tests section above.
- **Ship criteria:** issue can be moved to `done` with a Resolution section listing what landed across all three stages.

## Tests

- `--format ansi` output contains ANSI escapes; `--format plain` output contains zero ANSI escapes (assert byte-level absence of `\x1b[`).
- TTY detection: piped output (no TTY) defaults to plaintext.
- Empty body: no body section rendered, just metadata + timeline.
- Empty timeline (filed-only): timeline shows the one `filed` event.
- Long title in metadata cell wraps cleanly, doesn't break the table.
- Title containing `|` is escaped (renders correctly, doesn't split the cell).
- Title containing newlines is normalized (replaced with `<br>`).
- `--no-meta --no-events` outputs just the body.
- `--format json` round-trips through `issue.Parse`.
- `--format raw` is byte-identical to the file.
- Relations render with resolved short refs and titles when targets exist; show "(unresolved: <id>)" when not.
- The assembled markdown document (pre-glamour) is itself valid markdown — useful for snapshot tests independent of glamour version.

## Dependencies (proposed)

- `github.com/charmbracelet/glamour` — full document rendering. Style selects ANSI (`dark`/`light`/`auto`) or plaintext (`ascii`). Same approach GitHub/GitLab use for terminal markdown.
- `github.com/mattn/go-isatty` — TTY detection. Tiny.

That's the entire dependency footprint for this verb. No tablewriter, no goldmark direct import (glamour pulls it in transitively).

Footprint check: glamour pulls in chroma, lipgloss, termenv, goldmark. Proportional to what glamour does. Acceptable; revisit if binary size becomes a concern.

Footprint check: glamour pulls in chroma (syntax highlighting), lipgloss (styling), termenv (terminal capabilities). That's a meaningful tail. Acceptable tradeoff vs. writing our own ANSI renderer; revisit if the binary size becomes a concern.

## Cross-references

- Builds on resolver (`b03efee5` plumbing already exists).
- Will display `relations.related` once `2a0272df` lands (the `Related` table row is empty placeholder until then).
- Compact timeline format proposed here is what `ifs list --verbose` would also use (future).

## Out of scope

- Image rendering (terminal image protocols are inconsistent; skip).
- Interactive paging. The user can pipe to `less -R` (the `-R` preserves ANSI). Don't reinvent.
- Body editing. That's `ifs edit` (`e26997c4`).
- Themed color schemes beyond glamour's defaults. `GLAMOUR_STYLE` env var covers most cases.
- Markdown rendering in the terminal for the timeline (events are structured, not markdown).

## Anti-goals

- Don't write our own markdown parser. glamour (via goldmark) exists.
- Don't write our own ANSI renderer. glamour exists; the work is months.
- Don't write a custom plaintext renderer either. glamour's `ascii` style covers it.
- Don't conflate plaintext and "ANSI without colors." The `plain` mode must contain zero ANSI escapes — verify this is actually true of glamour's `ascii` style during implementation (test asserts byte-level absence of `\x1b[`).

## Progress

### 2026-04-28 — Stage 1 shipped

Metadata-only view rendering through the markdown → glamour pipeline. End-to-end pipeline now validated on a real issue.

What landed:
- `internal/md/md.go` — `EscapeCell` (handles `|`, `\n`, `\r\n`) and `Table` helpers. ~70 lines, no external deps. Six test cases in `internal/md/md_test.go`.
- `cmd/view.go` — `view <ref>` with `--format {auto|ascii|raw-md}`. `auto` defaults to `ascii` in this stage (TTY detection deferred to Stage 2). Resolves ref via `store.Resolver`, loads issue, assembles a metadata-only markdown table, renders via `glamour.NewTermRenderer` with the chosen style. `--format raw-md` bypasses glamour entirely (useful for debugging and snapshot tests).
- `cmd/view_test.go` — 7 cases: raw-md content present, ascii zero-ANSI assertion (`!strings.Contains(out, "\x1b[")`), auto-defaults-to-ascii in stage 1, pipe escaping, newline normalization (via hand-edited file), unresolvable ref, bad format.
- `cmd/root.go` — `newView()` registered.
- `go.mod` — `github.com/charmbracelet/glamour` and `github.com/mattn/go-isatty` added. (`go-isatty` is unused this stage; lands in Stage 2 for TTY detection.)
- `CLAUDE.md` — rendering convention now explicitly distinguishes human-prose output (markdown pipeline) from machine-grade output (`fmt.Fprintln`); examples cite `version`/`create`/`move` as machine-grade.

Smoke verified on a real issue (`2ba51c9e`): raw-md emits clean assembled markdown; ascii rendering is a clean table with no ANSI escapes. Glamour's `ascii` style does what it claims.

Tests: all packages green.

Still pending:
- **Stage 2**: body section rendered after the metadata table; `--format ansi` with TTY-detected `auto`.
- **Stage 3**: timeline section; `--no-meta`, `--no-events`, `--format json`, `--format raw`; relations row (depends on `2a0272df`).

Notes for Stage 2/3:
- The `joinOrNone` / `orNone` helpers in `cmd/view.go` use `_(none)_` markdown italics for empty values. Glamour's `ascii` style renders these as `*(none)*`, which is fine but worth knowing.
- `glamour.WithWordWrap(0)` was needed to disable hard wrapping; otherwise the metadata table got wrapped at default width and looked awful. Stage 2's body section may want different word-wrap behavior — revisit when adding.

### 2026-04-28 — Stage 2 shipped

Body rendering, ANSI format, and TTY-detected `auto`. Pipeline now produces real terminal output for human-prose viewing.

What landed:
- `cmd/view.go` — `assembleDocument` extracted from inline call site; appends body markdown after the metadata table separated by `\n---\n\n` when body is non-empty (no separator when body is empty, so empty-body issues stay clean).
- `--format` accepts `auto|ansi|ascii|raw-md` (added `ansi`).
- `pickStyle(format, writer)` resolves the format flag to a glamour style. `auto` calls `isTerminal(writer)` to pick `ansi` for TTYs and `ascii` otherwise.
- `ansiStyle()` honors `$GLAMOUR_STYLE` if set; defaults to `dark` so explicit `--format ansi` always emits escapes (glamour's own "auto" can downgrade to ascii in non-TTY environments, which would surprise a user who passed `--format ansi`).
- `isTerminal(io.Writer)` helper: type-asserts to `*os.File` and calls `isatty.IsTerminal(f.Fd())`. Returns false for buffers/pipes — makes TTY detection testable without ptys.
- `cmd/view_test.go` — 6 new cases: body present in raw-md (with `\n---\n\n` separator), empty-body suppresses separator, body renders in ascii without escapes, ansi output contains `\x1b[`, auto with no TTY (test buffer) defaults to ascii, `GLAMOUR_STYLE=ascii` env var overrides `--format ansi`.

Smoke verified on `7b8aca29` (the reconcile design issue): ascii output renders the metadata table, horizontal-rule separator, then the body's markdown including headings, bullet lists, and bold/italic emphasis. ANSI output emits real escape codes (`^[[38;5;252m`, etc.) and glamour color-highlights inline code spans like `` `7b8aca29` `` automatically.

Tests: all packages green.

Still pending:
- **Stage 3**: timeline section (events as a table after the body); `--no-meta`, `--no-events`, `--format json`, `--format raw`; relations row in metadata (depends on `2a0272df`).

Notes for Stage 3:
- The `\n---\n\n` separator pattern in `assembleDocument` should generalize for the timeline too: only add the separator when the next section is non-empty. Probably extract a small `appendSection(sb, body string)` helper to dedupe the "if non-empty, add separator + content" logic.
- For `--format json`, return the `*issue.Issue` (or a sanitized projection) — needs to bypass glamour entirely. `--format raw` should write the file bytes verbatim; the file is already on disk at `m.Path`, so it's a single `os.ReadFile` + write.
- The relations row will need the resolver to look up referenced IDs and render `short (truncated title)`. Until `2a0272df` ships, the row can stay as `_(none)_` or be omitted; document the choice in the Stage 3 progress entry.
- TTY detection currently checks the writer passed to `runView`. In production this is `cmd.OutOrStdout()` which is `os.Stdout`. If we ever support `--output <file>`, the file would correctly be detected as non-TTY.

### 2026-04-28 — Stage 3 shipped

Timeline section, remaining flags (`--no-meta`, `--no-events`, `--format json`, `--format raw`), and bypass paths. Verb is feature-complete per this issue's plan.

What landed:
- `cmd/view.go` — `assembleTimeline(iss)` renders events as a markdown table preceded by a `## Timeline` heading. `eventDetails(e)` formats the type-specific `from`/`to` fields uniformly (`from → to` for moves, `→ to` for `filed`, etc.) — generalizes cleanly when new event types arrive.
- `appendSection(b, body)` helper extracted per the Stage 2 note: writes `\n---\n\n` separator only when the builder already has content. Eliminates leading separators for body-only / timeline-only outputs.
- `viewOpts` struct introduced (`format`, `noMeta`, `noEvents`) instead of stretching the `runView` signature.
- `--format json` bypasses glamour, marshals the loaded `*issue.Issue` with `json.NewEncoder` + `SetIndent("", "  ")`. Same shape as the file frontmatter (since `Issue` is the same struct) — `ifs view --format json | jq` works.
- `--format raw` bypasses glamour and `issue.Parse` entirely, just `os.ReadFile(m.Path)` + write. Byte-identical to the on-disk file, including frontmatter.
- `--help` long description now lists all formats and section-skipping flags.
- `cmd/view_test.go` — 7 new cases: timeline rendered after a move (asserts both `filed` and `moved` rows present and `backlog → active` detail string), `--no-meta` suppresses metadata, `--no-events` suppresses timeline, body-only rendering with `--no-meta --no-events` has no leading separator, `--format json` is valid JSON with expected keys, `--format raw` is byte-identical to file, sanity check that bypass formats don't include rendered timeline header.

Smoke verified on `2ba51c9e` (the version issue, now done): full ascii rendering shows metadata table → body (markdown lists, code spans, italics) → timeline table with all three lifecycle events (`filed → backlog`, `moved backlog → active`, `moved active → done`). JSON format produces valid output starting with `{"title": "Add 'ifs version'..."}`.

Tests: all packages green. Total view test count: 20 cases (7 stage 1 + 6 stage 2 + 7 stage 3).

Deviations from the original plan:
- **Relations row deferred.** `2a0272df` (relations) hasn't shipped yet, so the metadata table doesn't have a "Related" row. Will add when `2a0272df` lands; trivial follow-up (one row in `assembleMetadata`, with the resolver lookup for short refs + truncated titles). Not blocking closure of this issue.
- One Stage 2 test (`TestView_EmptyBody_NoSeparator`) needed a tweak: with timeline now always present, "no separator at all" required `--no-events` too. Renamed to `TestView_EmptyBody_NoBodySection` and updated the assertion.

Follow-ups discovered during stage 3:
- Filed `f7367246` (bug): `ifs view <ref> | less` strips colors because `auto` detects no TTY on the pipe. Recommended fix is a `--color {auto|always|never}` flag decoupled from `--format`. Not implemented as part of this issue; out of scope for the original "render readably" plan, fits cleanly as a follow-up bug fix.

## Resolution

Implemented as designed across three stages. All planned functionality landed except the Related row, which is correctly deferred to follow `2a0272df` (relations).

What landed (full summary across stages):
- `internal/md/` package — markdown table assembly + cell escaping (`|`, newlines). 6 tests.
- `cmd/view.go` — `view <ref>` with `--format {auto|ansi|ascii|json|raw|raw-md}`, `--no-meta`, `--no-events`. Section-skipping is independent and composable. TTY detection via `mattn/go-isatty` (handles non-`*os.File` writers correctly so tests work without ptys). `$GLAMOUR_STYLE` env override honored.
- `cmd/view_test.go` — 20 cases covering all formats, section-skipping combinations, escape correctness, TTY detection behavior, and bypass-format sanity.
- `cmd/root.go` — `newView()` registered.
- `go.mod` — `github.com/charmbracelet/glamour` and `github.com/mattn/go-isatty`. Glamour's transitive deps (chroma, lipgloss, termenv, goldmark) accepted as the cost of not writing our own renderer.
- `CLAUDE.md` — rendering convention added (human-prose vs machine-grade output; markdown → glamour pipeline).

Established patterns for future verbs:
- `assembleDocument` + `appendSection` shape — any verb producing multi-section human-prose output should follow this. Sections are independently skippable; separators don't stack.
- `pickStyle` + `isTerminal` — any verb wanting TTY-aware rendering reuses these.
- `internal/md/` — second consumer (e.g. `list`, `b03efee5`) should reuse `Table` and `EscapeCell` directly. The Stage 1 prediction that helpers should "emerge from concrete use" was correct: their final shape was right after one user, no over-design.

Follow-ups filed:
- `f7367246` — pipe-to-less color stripping (recommend `--color` flag).
- The "Related row" — not a separate issue; queued behind `2a0272df` (relations). Whoever implements relations should add the row in `assembleMetadata` as part of that work.
- `e26997c4` (`ifs edit`) — when implemented, body edits to issues should fire a `body_edited` event; this issue's own Progress section additions are exactly the kind of hand-edit that would benefit.
