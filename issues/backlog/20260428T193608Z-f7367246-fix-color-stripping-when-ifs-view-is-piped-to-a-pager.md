{
  "title": "Fix color stripping when 'ifs view' is piped to a pager",
  "id": "20260428T193608Z-f7367246",
  "state": "backlog",
  "created": "2026-04-28T19:36:08Z",
  "labels": [
    "bug"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-28T19:36:08Z",
      "type": "filed",
      "to": "backlog"
    }
  ]
}

## Symptom

`ifs view <ref> | less` shows no colors. The metadata table renders in plain ascii instead of the styled ANSI version a TTY would get. Same for any pipe (`| grep`, `| head`, etc.) and any pager (`less`, `more`, `most`).

## Cause

`view`'s `--format auto` detects the writer's TTY-ness via `isatty(os.Stdout.Fd())`. When piped, stdout is a pipe (not a TTY), so `auto` picks `ascii`. Working as designed for grep/awk pipelines (you don't want ANSI escapes leaking into machine consumers), but the pager case is the dominant interactive use and it's broken.

## Reproduction

```bash
ifs view 7b8aca29              # styled ANSI (correct, in a terminal)
ifs view 7b8aca29 | less       # ascii, no colors (wrong, user wants colors here)
ifs view 7b8aca29 | less -R    # styled ANSI again — works because user passed -R explicitly,
                               # and ifs happens to also have emitted ANSI because... wait, no,
                               # it didn't. It emitted ascii because the pipe isn't a TTY.
                               # `less -R` only matters if the source produces escapes.
                               # So this DOESN'T fix it without also forcing color upstream.
```

So `less -R` alone isn't a workaround for the user — they'd also need `ifs view --format ansi <ref> | less -R`. Two flags to remember every time.

## Design space

### A. Add `--color {auto|always|never}` decoupled from `--format`

Standard pattern (`ls --color`, `grep --color`, `git --color`). `auto` does TTY detection like today; `always` forces ANSI regardless; `never` forces ascii regardless.

```bash
ifs view 7b8aca29 --color always | less -R   # works, two flags
```

Or, if we keep `--format`, `--color` overrides the auto-detection on the auto path:
- `--format auto --color always` → ansi style even when piped.
- `--format auto --color never` → ascii style even on TTY.
- `--format ansi`/`--format ascii` ignore `--color` (explicit format wins).

Cheap, well-understood, doesn't lock us out of B.

### B. Built-in pager invocation (the gh/git approach)

Detect TTY → spawn `$PAGER` (default `less`) → render ANSI → write to pager's stdin. Pass `LESS=FRX` (default for `git`) so colors render (`R`), pager exits if content fits in one screen (`F`), and the screen isn't cleared on exit (`X`).

```bash
ifs view 7b8aca29              # auto-pages if output > terminal height; ANSI throughout
ifs view 7b8aca29 --no-pager   # writes directly to terminal, no pager spawned
PAGER=cat ifs view 7b8aca29    # bypasses pager via env
```

Pros: matches `gh` / `git` UX; user doesn't think about it. Cons: more code, edge cases (pager not in PATH, pager exits early because user pressed `q`, signal handling).

### C. Document `LESS=R` and move on

Add a one-liner to README / `--help`: "set `LESS=R` in your shell rc to preserve color when piping to less." Almost no work; brittle (users forget; CI environments don't have it; `more`/`most` users still broken).

## Recommendation

**Ship A immediately**, plan B as a follow-up.

`--color {auto|always|never}` is ~15 lines of code, no new dependencies, and unlocks the `ifs view --color always | less -R` workflow with documented behavior. It also doesn't preclude later doing B — the auto-pager would just respect `--color always` as "force ANSI into the pager pipe."

C is fine as a documentation addition alongside A but isn't a fix on its own.

## Implementation (option A)

- `cmd/view.go`: add `--color {auto|always|never}` flag with default `auto`. Wire into `pickStyle`:
  - `auto` (default): unchanged from today (ansi if isatty, ascii otherwise).
  - `always`: ansi style regardless of TTY.
  - `never`: ascii style regardless of TTY.
- Validate the flag against the enum; error on invalid values.
- Update `--help` text to mention the pipe-to-less use case explicitly: `ifs view <ref> --color always | less -R`.

## Tests

- `--color always` with no TTY (test buffer) produces ANSI escapes.
- `--color never` with `--format ansi` produces no escapes (or: `--color` doesn't apply when format is explicit; pick a behavior and test it).
- `--color auto` matches current behavior (ascii in tests since the buffer isn't a TTY).
- Invalid `--color` value errors.

## Out of scope (option B follow-up)

- Spawning `$PAGER` automatically — separate issue, more invasive change.
- Detecting "would have fit in one screen, skip pager" — that's `less -F`'s job, not ours.
- `LESS=FRX`-style env defaults — that's the user's shell rc.

## Resolution

(filled in when closed)
