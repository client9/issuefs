# Why IssueFS

## What is IssueFS?

IssueFS (`ifs`) stores issues as markdown files alongside your code. Each issue is a single file under `issues/{backlog,active,done}/` — plain text, readable in any editor, renderable on GitHub, and trackable in git like any other source file.

The format is designed for both humans and AI agents: a structured JSON frontmatter block (machine-readable metadata: title, state, labels, assignee, timestamps, an append-only event log) followed by a freeform markdown body (human-readable narrative: plan, decisions, status, context).

## What problem does it solve?

Most issue trackers cover the bookends of the development lifecycle well: **filing a report** and **marking it closed**. The middle is largely invisible.

Consider a typical feature or bug fix. It passes through roughly:

1. Discussion and triage
2. Approval to work on
3. Implementation plan
4. The implementation itself
5. Current status and blockers
6. Completion

GitHub Issues handles 1 and 6. Steps 2–5 live on a developer's machine — in a local notes file, a chat window, or nowhere at all. The final pull request gestures at the plan but is really a diff, not a narrative. The *why* behind the code — the constraints that shaped it, the alternatives that were rejected, the tradeoffs that were consciously made — is typically lost.

IssueFS fills that gap. It's the **work record**: the plan, the decisions, the current status. It doesn't replace your issue tracker; it captures what your issue tracker doesn't.

## What problem does it not solve?

IssueFS is not a discussion system.

GitHub Issues, Linear, Jira — these are built for conversation: threaded comments, @mentions, reactions, notifications, community participation. That's the right tool for external bug reports, feature debates, and anything that benefits from multiple stakeholders weighing in.

IssueFS has no comment model by design. Discussion belongs in your existing tracker. When a GitHub Issue gets triaged and a decision is made to act on it, that decision — the scope, the approach, the plan — moves into an IssueFS file. The IssueFS body can reference the upstream issue number for traceability (`see: github#1234`). The two systems complement each other; they don't compete.

## Why now?

Human-paced development is slow, but slowness creates records. Standups, PR reviews, design docs, code review comments — these are friction, but friction leaves traces. When something breaks months later, there is usually someone who remembers, a thread to search, a document to find. Institutional knowledge accumulates through the overhead people complain about.

At AI speed, that friction is gone. A feature can go from idea to committed code in an hour. An AI agent has no memory between sessions — the chat is compacted, the session closes, and the context that would have accumulated across those human checkpoints simply does not exist. The code tells you *what* was built. It does not tell you *why*, what was tried and rejected, or what constraint made the obvious fix wrong.

This creates a failure mode that is new and underappreciated: **an AI session introduces a bug**. You open a new session to debug it. The new session can see the code. It cannot see the intent. Without the original reasoning, the debugger is working from incomplete information — and may reproduce the same error with more confidence. Worse: the fix may be technically correct but violate the original design intent, closing one bug while opening another.

IssueFS is **intent preservation**. Writing down the plan, the constraints, and the decisions is low cost during development and high value during debugging. Pointing an AI at `issues/done/20260427T143022Z-9f2a4b7c-fix-space-rocket-thrusters.md` and asking "given this bug, where did the implementation diverge from the plan?" is only possible if the plan was written down somewhere durable.

## Architecture: local and distributed vs. centralized and remote

IssueFS is explicitly local-first. Issue files live in the repository, travel with every clone, and are readable without credentials, network access, or API calls.

This is a deliberate tradeoff, not an oversight.

**What local-first gives you:**
- Works offline and in air-gapped environments
- No authentication, no rate limits, no API breakage
- AI agents and scripts can read issues directly without integration work
- Issues are versioned alongside the code that implements them — `git log` covers both
- Forking the repo forks the issue history too; no orphaned references

**What local-first does not give you:**
- A web UI accessible to people without a clone
- Notifications and subscriptions
- Cross-repository issue references
- Community participation from external contributors

The Go language project's issue tracker is a useful illustration of the limits of centralized trackers at scale: a feature request accumulates thousands of comments, spawns an external proposal document, links to a separate implementation discussion, and eventually closes with a note pointing to a commit. The connective tissue is scattered and partially lost. IssueFS does not try to solve that problem — it is not designed for community-scale coordination. It is designed for the work that happens *after* a decision is made, in a repo, by the people doing the work.

## Summary

| | GitHub Issues | IssueFS |
|---|---|---|
| External bug reports | ✓ | — |
| Community discussion | ✓ | — |
| Implementation plan | — | ✓ |
| Current status | — | ✓ |
| Design decisions | — | ✓ |
| AI-readable without API | — | ✓ |
| Travels with the repo | — | ✓ |
| Notifications / subscriptions | ✓ | — |
| Web UI | ✓ | — |

Use GitHub Issues (or equivalent) for the conversation. Use IssueFS for the work record. They are different things.
