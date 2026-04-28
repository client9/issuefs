{
  "title": "Add 'ifs view' to render an issue file readably (table for meta, ANSI/plaintext for body)",
  "id": "20260428T172347Z-aee0cbb9",
  "state": "backlog",
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
