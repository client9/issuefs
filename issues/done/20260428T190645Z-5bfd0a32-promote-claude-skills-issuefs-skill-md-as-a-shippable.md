{
  "title": "Promote .claude/skills/issuefs/SKILL.md as a shippable artifact (install via 'ifs init')",
  "id": "20260428T190645Z-5bfd0a32",
  "state": "done",
  "created": "2026-04-28T19:06:45Z",
  "labels": [
    "feature"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T19:06:45Z",
      "type": "filed",
      "to": "backlog"
    },
    {
      "ts": "2026-04-29T02:09:51Z",
      "type": "moved",
      "from": "backlog",
      "to": "active"
    },
    {
      "ts": "2026-04-29T02:15:48Z",
      "type": "moved",
      "from": "active",
      "to": "done"
    }
  ]
}

## Motivation

The skill at `.claude/skills/issuefs/SKILL.md` is genuinely portable — it describes how to use `ifs`, not how to build the issuefs codebase. Already in use across multiple projects via cut-and-paste. That's a smell: shipping the skill as a real artifact means consumers get updates automatically rather than drifting per-copy.

## What "promotion" means here

The skill becomes a **distributable resource** of the issuefs project, with installation paths that mirror how Claude Code already discovers skills:

- **Global install**: copy to `~/.claude/skills/issuefs/SKILL.md`. Available in every repo the user opens.
- **Project install**: copy to `<repo>/.claude/skills/issuefs/SKILL.md`. Travels with the repo, available to anyone (including teammates and CI sessions) who clones it.

The skill's trigger description already gates it correctly (only fires in repos with `issues/` directories using `ifs`'s file shape), so global install doesn't pollute unrelated work.

## Delivery mechanism (updated: `ifs init` now exists)

`2a280436` shipped, so `ifs init` is in the binary. This issue adds **`--install-skill {global|project|both}` as a flag on the existing init verb** rather than introducing a new sibling verb.

### Embed the skill in the binary

Move `.claude/skills/issuefs/SKILL.md` to a stable path that's both:
- Loadable for *this repo's* own Claude sessions (so editing the skill in source still affects local sessions).
- `//go:embed`-able for the binary distribution.

Two-location options:
- **A.** Keep `.claude/skills/issuefs/SKILL.md` as the canonical source. Add a build-time copy or symlink from `internal/embedded/skill/SKILL.md` (or wherever the embed lives). One source of truth; build-time replication.
- **B.** Move the canonical source to `share/skill/SKILL.md` (or `assets/skill/SKILL.md`). Symlink `.claude/skills/issuefs/SKILL.md` to it for this repo's sessions. Embed reads `share/skill/SKILL.md` directly. One source of truth; no build-time copy.

Recommendation: **B**. Simpler, no build-step coordination, the embed package directly references the canonical file. Symlink is one-time setup.

### `ifs init --install-skill <target>`

Extends the existing `init` verb. Defaults to no install (`init` alone scaffolds dirs only, as it does today).

```bash
ifs init                              # current behavior — scaffold dirs only
ifs init --install-skill project      # ALSO write <repo>/.claude/skills/issuefs/SKILL.md
ifs init --install-skill global       # ALSO write ~/.claude/skills/issuefs/SKILL.md
ifs init --install-skill both         # ALSO write both
```

Implementation:
- `internal/embedded/skill.go` — `//go:embed share/skill/SKILL.md`, exposed as `var SkillContent []byte` and `var SkillVersion string` (computed from `ifs version`).
- `cmd/init.go` — new `--install-skill` flag (`stringSliceVar` accepting `global`, `project`, or both via repeat). Validates values; resolves target path (`~/.claude/skills/issuefs/SKILL.md` or `<cwd>/.claude/skills/issuefs/SKILL.md`); calls `os.MkdirAll` on the parent; writes content via `WriteNew`-style refuse-to-overwrite; reports per-target output.
- Refuse to overwrite an existing skill file without `--force`. If the existing file is byte-identical to the bundled version, skip silently (idempotent). If it differs, print a hint about `--force` and the path so the user can diff manually.

### Output shape

Stays consistent with `init`'s existing per-line `created <path>` style:

```bash
$ ifs init --install-skill project
created issues/backlog/.gitkeep
created issues/active/.gitkeep
created issues/done/.gitkeep
created .claude/skills/issuefs/SKILL.md
```

If everything already exists:

```bash
$ ifs init --install-skill project
already initialized
skill already installed at .claude/skills/issuefs/SKILL.md (matches bundled version)
```

## Versioning

Skill version tracks the binary version (`ifs version` is done, `2ba51c9e`). The simplest scheme: byte-identical comparison between bundled and installed content.

- If installed file is byte-identical to bundled → "matches bundled version", silent no-op (or quiet status line).
- If different → refuse without `--force`; suggest the user diff the two paths.
- No embedded version comment needed initially — content equality is sufficient. Add a version comment later only if drift detection grows in scope (e.g. "what version was installed last?").

`ifs skill check` is overkill for v1; `init --install-skill` already reports drift on each invocation.

## Discoverability

Add to README:
- "Optional: install the Claude Code skill so AI agents working in your repo follow ifs conventions automatically."
- Link to a doc page describing what the skill does and the trigger criteria.

## Out of scope

- Auto-update the installed skill on `ifs` upgrade. Cute, but invasive — users may have customized their copies. Detect drift, don't auto-overwrite.
- Distribution as a separate Claude Code plugin / marketplace entry. The skill ships with the binary that uses it; that's the simplest packaging.
- Translating the skill into other agent frameworks (Cursor rules, Copilot instructions). The shape of the content would survive a port, but each platform's discovery mechanism differs. File separately if anyone asks.

## Dependencies

- ~~Builds on `ifs init` from `2a280436` (the empty-dirs bug).~~ **Resolved**: `2a280436` is closed, `ifs init` exists. `--install-skill` slots in as a flag on the existing verb.

## Readiness note

issuefs isn't 1.0 — features like `edit`, `reconcile`, and relations are still backlogged. But the core (create / list / view / move / verify / version / init) is functional and the skill itself is stable enough to be useful across other repos (already in use via cut-and-paste). Shipping it as an installable artifact is justified now even though the binary itself will keep evolving; the skill describes a small enough surface that it can be versioned independently of the broader feature set.

## Resolution

Implemented. SKILL.md is now a real artifact installable into any project (or globally) via `ifs init --install-skill {project|global}`. Idempotent, drift-aware, force-overridable.

What landed:
- `internal/embedded/embed.go` — new package. Exposes `Skill []byte` (`//go:embed SKILL.md`), `SkillFilename = "SKILL.md"`, `SkillRelDir = "skills/issuefs"`. The package directory IS the embed source — flat layout.
- `internal/embedded/SKILL.md` — canonical skill content (moved from `.claude/skills/issuefs/`).
- `.claude/skills/issuefs/SKILL.md` — **symlink** → `../../../internal/embedded/SKILL.md`. Single source of truth; edits affect both the repo's local Claude sessions AND the bytes installed by the new flag. Verified: `cat` follows the symlink, `//go:embed` reads the actual file.
- `cmd/init.go` — extended with `initOpts` (`installSkill []string`, `force bool`). New helpers: `skillTargetPath(target, cwd)` resolves "project"/"global" to absolute paths, `installSkill(...)` does byte-equality check + idempotent / drift-detect / force-override logic. Targets deduped so `--install-skill project --install-skill project` is harmless. Flag is repeatable for both targets in one invocation.
- `cmd/init_test.go` — 7 new cases: project install creates file, identical re-install is silent no-op, drift refuses without `--force` and preserves user edits, `--force` overwrites and restores bundled content, global install via `t.Setenv("HOME", ...)` for isolation, both targets in one invocation, bad target errors. Existing tests unchanged.
- `README.md` — rewritten from a one-liner to a proper landing page covering install, quick-start verbs, and the `--install-skill` workflow.
- `CLAUDE.md` — architecture section documents `internal/embedded/` and the symlink, including a recovery command if the symlink ever breaks.

Smoke verified end-to-end on a clean `/tmp/skill-smoke` repo: project install writes `.claude/skills/issuefs/SKILL.md`, content matches expected frontmatter, idempotent re-run reports "matches bundled version", manual edit + re-install errors with `--force` hint, `--force` restores bundled content.

Tests: all packages green.

Deviations from the original plan:
- **Embed location**: issue proposed `share/skill/SKILL.md`. Implementation went with `internal/embedded/SKILL.md` because `//go:embed` does not allow `..` in patterns — the file must live at-or-below the embed.go's directory. Putting both in `internal/embedded/` keeps the package self-contained without a multi-level path. The principle (one canonical file, embed reads it directly) holds.
- **Versioning**: stuck with byte-identical comparison. No embedded version comment.
- **`--force` requires `--install-skill`**: not enforced. Lone `--force` is a no-op rather than an error. Felt overkill.

Follow-ups discovered:
- The symlink is Unix-only in practice. Windows users cloning with `core.symlinks=false` may end up with a text-file-content "symlink". CLAUDE.md documents the recovery command. A future cross-platform fix could be a build-time copy step.
- A future `ifs skill check` (audit installed copies vs bundled) is probably unnecessary — `init --install-skill` already reports drift on each invocation.
- `internal/embedded/` is a natural home for any other build-time-bundled assets in the future (issue templates, reference docs, etc.).
