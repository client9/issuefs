{
  "title": "Initialize all three state subdirectories on first run (with .gitkeep)",
  "id": "20260428T184618Z-2a280436",
  "state": "backlog",
  "created": "2026-04-28T18:46:18Z",
  "labels": [
    "bug"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T18:46:18Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Symptom

After cloning a repo with `issuefs` set up, `issues/active/` and `issues/done/` may not exist — they get lazy-created the first time `ifs move` puts a file there. Worse, git doesn't track empty directories, so even if the original author created them, they don't survive `git clone` unless something is in them.

Surfaced during the dogfooding workflow that closed `2ba51c9e`: after `ifs move 2ba51c9e done` succeeded locally, `git status` showed the moved file but git silently didn't add the new `issues/done/` directory. Required a manual `git add issues/done/` to recover.

## Reproduction

```bash
mkdir new-repo && cd new-repo && git init
ifs create -t "first issue"        # creates issues/backlog/ — fine
ifs move <ref> done                 # creates issues/done/ on the fly — but git sees only the moved file, not the new dir
git add .
git status                          # the rename works, but the dir is implicit
# Now clone the repo elsewhere:
git clone <this> /tmp/clone && ls /tmp/clone/issues/   # only `backlog`, no `active` or `done`
# Next user runs `ifs list --state done` — works (no entries) — but any tool relying on the dirs existing breaks.
```

## Suspected cause

`store.EnsureSubdir` is called lazily on demand by `move`. There's no init step that scaffolds all three states up-front. And we have no mechanism to make empty directories git-trackable.

## Fix proposal

Two options, not mutually exclusive:

### A. Eager scaffold on first `create`

When `ifs create` discovers the `issues/` root doesn't exist yet (or exists but lacks a state subdir), create all three (`backlog`, `active`, `done`) at once. Cost: one-time `MkdirAll` ×3. Benefit: subsequent moves don't surprise git.

Place a `.gitkeep` (empty file) in any newly-created subdir that has no `.md` files yet, so git tracks the directory. Remove `.gitkeep` from a subdir the moment a real `.md` file lands there (or just leave it — `.gitkeep` files are harmless and conventional).

### B. Explicit `ifs init` verb

`ifs init` scaffolds `issues/{backlog,active,done}/.gitkeep` in one shot. Useful for users who want explicit setup before filing the first issue, or for CI/template repos that bootstrap the structure.

Both A and B can coexist: `ifs init` is the explicit form; `ifs create` does the same scaffold lazily on first run if `init` wasn't called.

## Implementation notes

- `internal/store/store.go` — extend `Resolve` (or add a `Scaffold(root string) error`) that ensures all three subdirs exist with `.gitkeep`.
- Call from `cmd/create.go` after the existing `Resolve` call, before the per-state `EnsureSubdir`.
- New verb `cmd/init.go` if we go with option B.
- `EnsureSubdir` continues to work as-is (idempotent on existing dirs).

## Tests

- Fresh tempdir, `ifs create -t x` → all three subdirs exist with `.gitkeep`.
- Existing partial setup (only `backlog/` exists) → `create` adds the missing two.
- Existing complete setup → `create` is a no-op for scaffold (no spurious `.gitkeep` writes if real files are present).
- After moving a file into a `.gitkeep`-marked subdir, `.gitkeep` may stay (harmless) or be removed (cleaner) — pick one and document.

## Out of scope

- Migrating existing repos that already have the structure but no `.gitkeep` files. They'd benefit from `ifs init` being run once, but no automation needed.
- Renaming `.gitkeep` to anything fancier (`.placeholder`, `README.md`). `.gitkeep` is the universal convention; don't reinvent.

## Resolution

(filled in when closed)
