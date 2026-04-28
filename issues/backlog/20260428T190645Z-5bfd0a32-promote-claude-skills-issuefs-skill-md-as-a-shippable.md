{
  "title": "Promote .claude/skills/issuefs/SKILL.md as a shippable artifact (install via 'ifs init')",
  "id": "20260428T190645Z-5bfd0a32",
  "state": "backlog",
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

## Proposed delivery mechanism

Two layers, smallest first:

### 1. Ship the skill in the repo at a stable path

Move (or symlink) `.claude/skills/issuefs/SKILL.md` to `share/skill/SKILL.md` (or similar). The current location can keep a copy/symlink so this repo's own Claude sessions still load it.

Stable path lets users do this without `ifs init`:

```bash
# global install
cp $(go env GOPATH)/pkg/mod/github.com/.../share/skill/SKILL.md ~/.claude/skills/issuefs/

# or, if cloned:
cp share/skill/SKILL.md ~/.claude/skills/issuefs/
```

### 2. `ifs init --install-skill {global|project|both}`

Extends the future `ifs init` verb (filed as `2a280436`) with a flag that installs the bundled skill into the chosen location. Defaults to no skill install (don't be opinionated unless asked).

Implementation: embed `share/skill/SKILL.md` into the binary via Go 1.16+ `embed`. Write to the chosen target path. Refuse to overwrite without `--force`; print a diff hint if a different version is already there.

```bash
ifs init                          # scaffolds issues/{backlog,active,done}/
ifs init --install-skill project  # also writes .claude/skills/issuefs/SKILL.md
ifs init --install-skill global   # writes ~/.claude/skills/issuefs/SKILL.md
ifs init --install-skill both
```

## Versioning

Bundle skill version with the binary version (`ifs version` already exists, `2ba51c9e` is done). On install, write a `# version: <semver>` comment in the skill's frontmatter or as the first body line so `ifs init --install-skill --force` can detect mismatches and prompt sensibly.

If we want to be fancy: `ifs skill check` compares installed copies against the bundled version and reports drift. Probably overkill for v1; mention as a possible follow-up.

## Discoverability

Add to README:
- "Optional: install the Claude Code skill so AI agents working in your repo follow ifs conventions automatically."
- Link to a doc page describing what the skill does and the trigger criteria.

## Out of scope

- Auto-update the installed skill on `ifs` upgrade. Cute, but invasive — users may have customized their copies. Detect drift, don't auto-overwrite.
- Distribution as a separate Claude Code plugin / marketplace entry. The skill ships with the binary that uses it; that's the simplest packaging.
- Translating the skill into other agent frameworks (Cursor rules, Copilot instructions). The shape of the content would survive a port, but each platform's discovery mechanism differs. File separately if anyone asks.

## Dependencies

- Builds on `ifs init` from `2a280436` (the empty-dirs bug). Best implemented as a follow-up to that issue rather than blocked on it — `ifs init --install-skill` could exist as a standalone verb if `init` doesn't land first.

## Resolution

(filled in when closed)
