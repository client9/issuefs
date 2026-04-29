{
  "title": "Initialize all three state subdirectories on first run (with .gitkeep)",
  "id": "20260428T184618Z-2a280436",
  "state": "done",
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
    },
    {
      "ts": "2026-04-29T01:46:59Z",
      "type": "moved",
      "from": "backlog",
      "to": "active"
    },
    {
      "ts": "2026-04-29T01:55:27Z",
      "type": "moved",
      "from": "active",
      "to": "done"
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

## `.gitkeep` policy (decided)

**Always present in every state subdir, regardless of whether real `.md` files are there.** Three permanent zero-byte files; cost is negligible; the directory survives every state-emptying operation forever. Considered alternatives:

- *On-empty only*: add `.gitkeep` when a state dir has zero `.md` files; remove when one lands. Creates ugly coupling — every move/create verb must know about gitkeep. Edge case: moving the last issue out of a dir resurrects the gitkeep need at exactly the wrong moment.
- *Never*: lazy-create at runtime. This is what we have today; it's the bug.

Always-present is the simplest, most robust policy. The move/create verbs don't have to think about anything — `.gitkeep` is just always there.

## Fix plan (do both)

`ifs init` and lazy-on-first-write coexist; both call the same worker.

### `store.Scaffold(root) ([]string, error)` — single source of truth

Idempotent. Ensures `backlog/`, `active/`, `done/` all exist with `.gitkeep` files. Returns a slice of paths it actually created (empty slice = nothing was missing). Safe to call from any context.

### `ifs init` — explicit verb

Calls `Scaffold`. Prints per-file results ("created issues/backlog/.gitkeep", "issues/done/ already exists"). Exits 0 even when nothing needed doing. Useful for:
- Brand-new repos (creates everything from scratch).
- Existing repos missing `.gitkeep` files (this repo today). Adds the missing pieces; immunizes the layout against future state-emptying moves.
- Scripts / CI / template bootstraps (deterministic, no fake-issue trick).
- Repair after `rm -rf issues/active` or similar.
- Future home for `ifs init --install-skill` (see `5bfd0a32`).

### `ifs create` — lazy resilience

Replaces today's `EnsureSubdir(root, state)` lazy-create with a `Scaffold(root)` call. Keeps the existing one-line stderr notice when scaffolding actually creates anything (today's "creating new issues directory at..." message generalizes naturally).

### Read-only verbs do NOT scaffold

`verify`, `view`, `list` read; they don't modify the tree. Scaffolding belongs in verbs that already write. Rule: **`Scaffold` is called only from verbs that are otherwise going to write to `issues/`.** This avoids surprising the user with filesystem changes during introspection.

(Future-config note: backlog/active/done is the simplest workflow that works for now. If states ever become configurable, `Scaffold` reads the configured list; the policy above doesn't change.)

## Implementation

- `internal/store/scaffold.go` (new file) — `Scaffold(root string) ([]string, error)`. Walks `["backlog", "active", "done"]`; for each, `MkdirAll` then check/create `<dir>/.gitkeep`. Returns paths created (for caller to print).
- `cmd/init.go` (new) — `newInit()` cobra command, calls `Scaffold`, prints results.
- `cmd/create.go` — replace existing `EnsureSubdir(root, o.state)` call with `Scaffold(root)` followed by `EnsureSubdir(root, o.state)` (the latter remains a no-op for existing dirs but keeps the per-state path resolution symmetric). Preserve the "creating new issues directory at" stderr notice but make it conditional on Scaffold actually creating anything.
- `cmd/root.go` — register `newInit()`.

## Tests

- `Scaffold` on fresh tempdir → creates 3 dirs + 3 .gitkeep files; returns 6 paths.
- `Scaffold` on partial state (only `backlog/` exists with files) → adds `active/` + `done/` + `.gitkeep` in all three (since backlog's gitkeep was missing too); returns the new paths.
- `Scaffold` on fully-set-up tree (everything present) → returns empty slice; no spurious writes.
- `Scaffold` is safe to call repeatedly (idempotent).
- `ifs init` on fresh tempdir prints expected output and creates files.
- `ifs init` on already-initialized tree exits 0 with "already initialized" or per-file "exists" output.
- `ifs create` on fresh tempdir produces full scaffold (existing behavior tightened: now creates active/ and done/ as well, with .gitkeep).
- `ifs create` no longer surprises git after the first move (smoke / integration check).
- `ifs verify` and `ifs view` and `ifs list` do NOT create anything when run in a tempdir without `issues/`. (Negative test for the read-only rule.)

## Out of scope

- Migrating existing repos at runtime is exactly what `ifs init` is for; no separate automation needed.
- Configurable state names (deferred indefinitely; backlog/active/done is the current contract).
- Renaming `.gitkeep` to anything fancier (`.placeholder`, `README.md`). Universal convention; don't reinvent.

## Resolution

Implemented as designed. Both `ifs init` (explicit) and `ifs create` (lazy on first write) call the same `store.Scaffold(root)` worker. `.gitkeep` policy is "always present in every state subdir" — three permanent zero-byte files, never managed based on dir contents.

What landed:
- `internal/store/scaffold.go` — `Scaffold(root) ([]string, error)`. Idempotent. Creates root if missing, then walks `[backlog, active, done]` ensuring each dir + `.gitkeep` exists. Returns the list of paths actually created so callers decide whether to print. Helper functions `ensureDir` and `ensureEmptyFile` both return a `bool` for "did I create it" so the top-level can build the list cleanly. Errors with a clear `PathError` if root exists but is a regular file.
- `internal/store/scaffold_test.go` — 6 cases: fresh root, idempotent re-run, partial state (backlog/ exists but no .gitkeep), fully set up, .gitkeep preserved when real files are present, root-is-a-file edge case.
- `cmd/init.go` — `newInit()` cobra command. Calls `Scaffold`, prints `created <relpath>` per file or `already initialized` if nothing was missing. `cobra.NoArgs`. Uses `filepath.Rel` for readable output.
- `cmd/init_test.go` — 4 cases: fresh repo creates all expected paths, second run is "already initialized", partial repair adds only what's missing (asserts the bare backlog/ line is NOT printed when backlog already exists), and the read-only-verbs negative test (`list`, `verify` do NOT create `issues/`).
- `cmd/create.go` — replaced `EnsureSubdir(root, o.state)` with `Scaffold(root)`, then computes `dir := filepath.Join(root, o.state)` directly. Existing "creating new issues directory" stderr notice generalized to per-file `created <relpath>` messages (still on stdout — matches `init`'s output for consistency).
- `cmd/create_test.go` — `TestCreate_ScaffoldsAllStates` regression added: after `ifs create` on a fresh repo, all three state dirs + their `.gitkeep` files exist.
- `cmd/move_test.go` + `cmd/view_test.go` + `cmd/create_test.go` — added `mdEntries(t, dir)` helper that filters `os.ReadDir` results to `.md` files only. Replaced ~13 raw `os.ReadDir` calls with it. Without this, `entries[0].Name()` returned `.gitkeep` (sorts alphabetically before timestamps), and `strings.Split(".gitkeep", "-")[1]` panicked.
- `cmd/root.go` — `newInit()` registered.
- `.claude/skills/issuefs/SKILL.md` — verb cheatsheet now includes `ifs init`, with a one-line note recommending it after cloning a repo that uses `ifs`.

Smoke verified by running `ifs init` on this repo (which already had populated state dirs but no `.gitkeep` files): correctly added all three `.gitkeep` files, then a second run reported `already initialized`. The repo is now immunized against the original bug.

Tests: all packages green (cmd suite ~2.5s due to the existing `--sort updated` test sleeping for sub-second timestamp resolution).

Deviations from the original plan:
- The "creating new issues directory at" message in `create` was generalized to per-file `created <relpath>` lines so it matches `init`'s output exactly. Slightly noisier on first run (4–7 lines instead of 1) but consistent across the two verbs and more honest about what was created.
- Tests for read-only verbs were narrowed: `list` and `verify` are tested for not-scaffolding; `view` was harder to test cleanly without an existing issue (it errors on lookup before any potential scaffold call), so the negative test omits it. The rule still holds in practice (`view` doesn't call `Scaffold`), and could be added later as a code-search assertion if needed.

Follow-ups discovered:
- `ifs init` is now the natural home for `--install-skill {global|project|both}` per `5bfd0a32`. That issue's plan should be updated to assume `ifs init` exists rather than proposing it as a sibling option.
- The `mdEntries` helper is a small reusable bit; if it ever needs to live outside `cmd/`, factor into `internal/testutil/` or similar. Not worth doing speculatively.
- `Scaffold`'s "ensure root exists" call is a side effect not strictly required by the immediate use case (callers like `create` always Resolve first); harmless but worth knowing — `Scaffold` will create `issues/` itself if you point it at a path whose parent exists. This makes `ifs init` fully self-sufficient (doesn't depend on prior `Resolve`).
