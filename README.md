# issuefs

Issue File System — treat issues as files, keep issues with code, for humans and AI.

`ifs` is a Go CLI that stores issues as one markdown file per issue under `issues/{backlog,active,done}/`. Files use bare-JSON frontmatter followed by a markdown body, so they're readable in any editor and renderable on GitHub. Each issue carries an append-only event log of metadata changes.

## Install

```bash
go install github.com/nickg/issuefs@latest
```

## Quick start

```bash
ifs init                            # scaffold issues/{backlog,active,done}/
ifs create -t "Fix space rocket thrusters" -l bug
ifs list
ifs view <short-ref>
ifs move <short-ref> active
ifs move <short-ref> done
```

## Use with Claude Code

`ifs` ships a Claude Code skill that teaches AI sessions how to file, browse, and update issues consistently. Install it once after cloning a repo (or globally for all repos):

```bash
ifs init --install-skill project    # writes <repo>/.claude/skills/issuefs/SKILL.md
ifs init --install-skill global     # writes ~/.claude/skills/issuefs/SKILL.md
ifs init --install-skill project --install-skill global
```

The skill is gated by a trigger description that only fires in repos with an `issues/` directory using `ifs`'s file shape, so global install doesn't pollute unrelated work. Re-running `init --install-skill` is idempotent: byte-identical files are silently skipped, and modifications you've made to the skill are preserved unless you pass `--force`.
