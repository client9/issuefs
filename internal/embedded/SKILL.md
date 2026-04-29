---
name: issuefs
description: Use when working in the issuefs repo or any repo containing an issues/ directory with bare-JSON-frontmatter markdown files. Triggers on requests to file/create/log/save an issue, comment, or design idea; to read pending work or backlog; to move issues between backlog/active/done; to verify issue files; or when the user dictates a discussion they want "saved for later." Skip for unrelated repos with no issues/ directory.
---

# issuefs (`ifs`)

> This skill describes how to use the `ifs` CLI. It is intentionally project-agnostic. Project-specific conventions live in the project's `CLAUDE.md`.

`ifs` is a Go CLI that stores issues as one markdown file per issue under `issues/{backlog,active,done}/`. Files use bare-JSON frontmatter (Hugo-style) followed by a markdown body. The tool is designed for human + AI collaboration: filing, retrieving, and acting on issues should be cheap from either side.

## When to use the tool vs. read files directly

- **Reading**: `cat`/Read the file directly. Bare-JSON frontmatter is parseable as-is; body is plain markdown. No tool needed.
- **Filing/moving/verifying**: always use `ifs`. Hand-writing a file is possible but skips event-log bookkeeping.

## Verb reference

```
ifs init                                     # scaffold issues/{backlog,active,done}/ with .gitkeep; idempotent
ifs create -t "<title>" [-b <body> | --body-file <path|->] [-l <label>...] [-a <assignee>...] [--state backlog|active|done] [-m <milestone>] [-p <project>...] [-T <template>]
ifs list [-s <state>...] [-l <label>...] [-a <assignee>...] [-m <milestone>...] [-L <limit>] [--sort created|updated] [--since <date>] [--format auto|ansi|ascii|json|raw-md] [--json]
ifs move <ref> <state>
ifs view <ref> [--format auto|ansi|ascii|json|raw|raw-md] [--no-meta] [--no-events]
ifs verify <file>...
ifs version [--short]
```

Run `ifs init` once after cloning a repo that uses `ifs` — it adds `.gitkeep` files so state directories survive in git even when they're empty. Safe to run on any repo (idempotent; reports "already initialized" if everything's set up).

`<ref>` accepts: a path, a full filename, the full ID, or any unique prefix of the 8-hex random suffix. So if `ifs create` reports id `20260428T050659Z-7b8aca29`, later commands can use `7b8a` (or `7b8aca29`).

`create` prints the created file's path on stdout. `move` prints the new path. `verify` is silent on success and prints `<path>: <error>` per failure (exits non-zero if any failed).

## File anatomy

```
{
  "title": "...",
  "id": "<ts>-<8hex>",
  "state": "backlog",
  "created": "<RFC3339>",
  "labels": [],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {"ts": "<RFC3339>", "type": "filed", "to": "backlog"}
  ]
}

Body markdown here.
```

The `events` array is an append-only log of metadata changes (modeled on GitHub's issue timeline). Current vocabulary: `filed`, `moved`. Body edits are not tracked as events; defer to git for body history.

`Verify` enforces a consistency contract: events non-empty, first is `filed` with `ts == created`, timestamp-monotonic, latest event with non-empty `to` equals current `state`.

## Filing conventions (important)

These conventions matter for downstream usability — both for humans scanning `ls issues/backlog/` and for future Claude sessions reading the backlog.

1. **Title is imperative action**, not a noun phrase. Good: `"Fix space rocket thrusters"`, `"Implement reconcile: detect and log hand-edits as synthesized events"`. Bad: `"Reconcile design"`, `"Rocket thrusters"`. When the title leads with an action verb, `ls` reads like a TODO list and triage is fast.

2. **Body is the distilled artifact, not the conversation log.** When filing an issue from a chat discussion, write the *conclusion* and *plan* — what someone needs to act on it later. Don't dump the back-and-forth that produced the conclusion. The transcript is in your conversation history; the issue body is what survives without you.

3. **Labels do real triage work.** Use them. Suggested initial set:
   - `feature` — new capability
   - `bug` — broken behavior
   - `design` — architectural / not-yet-decided
   - `refactor` — internal cleanup
   - Combine where useful: `feature`+`design` distinguishes "build this" from "still figuring out what to build."

4. **Default state is `backlog`** — explicit `--state active` only when the user is starting work right now.

5. **When closing an issue (moving to `done`), append a `## Resolution` section to the body.** This documents what was actually built — what landed, what deviated from the original plan, follow-ups discovered, decisions that were made differently than proposed. The original body is the *plan*; the Resolution section is the *outcome*. Future readers (and a future `ifs changelog`) need both.

   Minimal template:

   ```markdown
   ## Resolution

   Implemented as proposed, with these deviations:
   - <deviation> — <reason>

   What landed:
   - <file or code summary>
   - <test coverage>
   - <docs updates>

   Follow-ups: <none, or list of new issues filed>
   ```

   If implementation matched the plan exactly, a one-line resolution is fine: `Implemented as designed; <commit-or-summary>.` The point is to leave a trace, not to write a thesis.

6. **For multi-stage or multi-session work, append a `## Progress` section** as you go — don't wait until the issue closes. Each meaningful intermediate landing (a stage shipped, a checkpoint reached, a session ended with partial work) gets a dated subsection.

   This is the staged-work analog of the Resolution section: same purpose (record what actually happened), same audience (future readers including AI sessions that pick up the work), but for incremental landings instead of final closure. Without it, anyone resuming the issue has to reverse-engineer what's already done from code archaeology.

   Minimal template:

   ```markdown
   ## Progress

   ### 2026-04-28 — Stage 1 shipped

   What landed:
   - <files / changes>
   - <tests added>

   Still pending: <stages or items not yet done>
   ```

   For trivial updates, a one-line entry is fine: `- **2026-04-28 — Stage 1 shipped.** <summary>. Tests green.`

   When the issue eventually closes, the Resolution section either summarizes across all Progress entries or just references them. Don't delete Progress entries on closure — they're history.

   **Trigger rule for AI sessions**: any time you ship code or tests as part of an active issue and are about to stop (whether the issue is fully done or not), update the Progress section before stopping. If the issue is closing, that update can become the Resolution section directly.

## AI/agent workflow rules

- **`create` is a pure file operation. Never run `git add`/`commit`/`push` on its own.** The user may have unrelated changes in flight; auto-staging would silently mix them. If the user wants the new files staged or committed, do it as a separate explicit step *only when they ask*.
- **Batch creates, then surface them.** When asked to file multiple issues from a discussion, write them all, list the resulting short IDs, and let the user decide whether to commit (and as one commit or many).
- **Use `--body-file -` with a heredoc** for any body longer than a sentence or containing markdown structure. Inline `--body "..."` quoting gets ugly fast.
- **When picking up work**, read the file at the path `ifs create` reported, not just the title. The body has the plan; the events log has the history.

## Common pattern: parking a design discussion

User-and-Claude generate a wall-of-text design that isn't ready to implement. To save it:

```bash
ifs create -t "Implement <thing>: <one-line summary>" -l feature -l design --body-file - <<'EOF'
## Concept
<1-paragraph what and why>

## Hard parts
<bulleted list of non-obvious problems>

## Recommended phasing
<numbered list of incremental steps>

## Anti-goals
<what NOT to do, if relevant>
EOF
```

Future session: `ifs move <short-ref> active`, read the file, implement step 1.

## Common pattern: filing a bug from a chat report

```bash
ifs create -t "Fix <broken behavior>" -l bug --body-file - <<'EOF'
## Symptom
<what the user observed>

## Reproduction
<minimal steps>

## Suspected cause
<if known; omit if not>
EOF
```

## Things not to do

- Don't put the conversation transcript in the body. Write the distilled outcome.
- Don't pre-fill `--state active` for a backlog item — the user moves it when they start.
- Don't invent event types not in the current vocabulary (`filed`, `moved`). Adding a new type means updating `internal/issue/event.go` and the `Verify` consistency rules; it's a code change, not a file-content change.
- Don't hand-edit an issue file's `events` array. If you need to record a state change, run `ifs move` so the event is appended consistently.
- Don't run `ifs verify` and treat success as confirmation that the *content* is correct — `verify` only checks structural well-formedness, not that the title or body is what the user wanted.

## Reading the backlog

Quick survey: `ls issues/backlog/` shows filenames with timestamps + slugs. To read a specific one, open the file directly. To find by short ref:

```bash
ls issues/*/*<short>* 
```

For richer queries (by label, by date, by assignee), no built-in command exists yet — `grep -l` over the JSON frontmatter works as a stopgap (`grep -l '"labels".*"bug"' issues/*/*.md`).
