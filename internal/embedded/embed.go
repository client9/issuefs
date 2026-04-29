// Package embedded holds resources bundled into the ifs binary at build time.
//
// SKILL.md is the canonical issuefs skill (the user-facing portion of CLAUDE
// Code's skill system; describes how to use the ifs CLI). The repo's local
// .claude/skills/issuefs/SKILL.md is a symlink to this file so the canonical
// content lives in one place. `ifs init --install-skill` writes this content
// to a target location in another repo or the user's home dir.
package embedded

import _ "embed"

//go:embed SKILL.md
var Skill []byte

// SkillFilename is the conventional basename for an installed skill.
const SkillFilename = "SKILL.md"

// SkillRelDir is the conventional path under a project's or user's .claude/
// directory where the skill lives.
const SkillRelDir = "skills/issuefs"
