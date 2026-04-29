{
  "title": "Extend 'ifs init --install-skill' to support Codex, Gemini, and AGENTS.md conventions",
  "id": "20260429T022453Z-72d6e56c",
  "state": "backlog",
  "created": "2026-04-29T02:24:53Z",
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
      "ts": "2026-04-29T02:24:53Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Motivation

`--install-skill` ships SKILL.md to Claude Code's discovery paths (`.claude/skills/issuefs/SKILL.md`, project or global). The agent ecosystem is broader than Claude Code, and the skill content itself is mostly framework-neutral — it describes how to use `ifs`, not how Claude works. Other agent tools could benefit from the same guidance if shipped to the right path with the right format.

Concretely, three target conventions worth supporting:

1. **Codex** (OpenAI Codex CLI / similar) — convention TBD; recent tooling appears to use `AGENTS.md` at repo root or `.codex/` subdirectories. Needs research before implementation.
2. **Gemini** (Google Gemini Code Assist / Gemini CLI) — convention TBD; commonly `.gemini/` or `GEMINI.md`. Needs research.
3. **AGENTS.md (cross-tool convention)** — `AGENTS.md` at repo root, intended as a unified instruction file readable by any agent (started as a community convention to avoid per-tool sprawl). Several tools now read it as a fallback when their native config is absent.

## Proposed flag design

Add `--type` (or `--agent`, `--target-format` — bikeshed at implementation time) alongside the existing `--install-skill` location flag:

```bash
ifs init --install-skill project --type claude     # current behavior; default
ifs init --install-skill project --type codex
ifs init --install-skill project --type gemini
ifs init --install-skill project --type agents     # writes AGENTS.md at repo root
ifs init --install-skill project --type all        # writes every known format
```

`--type` repeats like `--install-skill` does, so combinations work:

```bash
ifs init --install-skill project --type claude --type agents
```

Default `--type` stays `claude` for backward compatibility. Adding `--install-skill` without `--type` continues to do exactly what it does today.

## Content adaptation

The current SKILL.md has Claude-Code-specific bits:
- YAML-style frontmatter (`name:`, `description:`) that drives Claude Code's description-based skill triggering.
- A leading sentence: "This skill describes how to use the `ifs` CLI."
- Workflow rules that reference Claude Code's prompt syntax (e.g. "When the user types `/<skill-name>`...").

Two strategies for handling this across formats:

**A. Single source, light per-target transformation.** The canonical file (`internal/embedded/SKILL.md`) stays Claude-shaped. Per-target writers do small transformations:
- `claude` → write as-is.
- `agents` / `codex` / `gemini` → strip the YAML frontmatter (which is Claude-Code-specific trigger metadata, meaningless to other tools); leave the body intact. May need to rewrite "This skill" → "This document" or similar.

Pros: one source of truth; transformations are mechanical; new target formats are just new transformations.
Cons: transformations need test coverage; framing tweaks can drift from the source over time.

**B. Per-target source files.** Maintain separate `internal/embedded/SKILL.claude.md`, `SKILL.agents.md`, etc.

Pros: full control per target; no transformation logic.
Cons: N files to keep in sync; the bulk content is identical across all of them; updates require touching every file.

Recommendation: **A**. Start with one transformation (strip Claude frontmatter for non-claude targets) and grow only if real divergence emerges.

## Path resolution per type

Each `--type` × `--install-skill` combo resolves to a target path:

| --type   | project location                          | global location                              |
|----------|-------------------------------------------|----------------------------------------------|
| claude   | `<repo>/.claude/skills/issuefs/SKILL.md`  | `~/.claude/skills/issuefs/SKILL.md`          |
| codex    | TBD (research)                            | TBD (research)                               |
| gemini   | TBD (research)                            | TBD (research)                               |
| agents   | `<repo>/AGENTS.md`                        | TBD — `~/AGENTS.md` makes no sense; probably project-only |

Note `agents` may not have a meaningful `global` install (a single repo-root `AGENTS.md` doesn't generalize to the home directory). Probably error if `--type agents --install-skill global` is requested, with a hint to use `--install-skill project`.

## Implementation skeleton

- `internal/embedded/` gains a small per-type table mapping `(type, location) → relative path`. Plus a `Transform(type, content) []byte` function for content adaptation per strategy A.
- `cmd/init.go`'s `installSkill` becomes target-type-aware: looks up the path via the table, applies the transform, then runs the existing byte-equality / drift / force logic.
- Validation: invalid `--type` errors before any write; `--type agents --install-skill global` errors with a clear message.

## Tests

Per type:
- Project install creates the expected file at the expected path with the expected (transformed) content.
- Identical re-install is a silent no-op.
- Drift refuses without `--force`; `--force` overwrites.

Plus combination tests:
- `--type claude --type agents` writes both with their respective transformations.
- `--type all` writes every known type.
- Bad `--type` errors.
- `--type agents --install-skill global` errors with a useful hint.

## Out of scope

- Translating the *content* substantively (e.g. rewriting workflow rules for tool-specific UX). The skill's value is in the conventions and verbs it describes; those are tool-neutral. Light framing transformations are fine; rewriting the body is too much divergence to maintain.
- Importing skill content from other tools (going the other way). Not the use case here.
- A `--list-types` flag enumerating supported targets. `--help` covers it.
- Auto-detecting which agent the user has installed. Too magic; explicit `--type` is clearer.

## Dependencies

- Research: actual conventions for Codex, Gemini, and current state of `AGENTS.md` adoption. Without this, the path resolution table can't be filled in. **This is the main blocker** — implementation is straightforward once the targets are known.

## Resolution

(filled in when closed)
