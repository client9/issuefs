package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/nickg/issuefs/internal/embedded"
	"github.com/nickg/issuefs/internal/store"
	"github.com/spf13/cobra"
)

type initOpts struct {
	installSkill []string
	force        bool
}

func newInit() *cobra.Command {
	o := &initOpts{}
	c := &cobra.Command{
		Use:   "init",
		Short: "Initialize the issues/ directory layout (and optionally install the skill)",
		Long: `Create issues/{backlog,active,done}/ with .gitkeep files in each.
Idempotent. Safe to run on a fresh repo, on an existing repo missing some
state directories, or on a fully-set-up repo (in which case it's a no-op).

With --install-skill, also write the bundled Claude Code skill to one or
both standard locations:

  project   <cwd>/.claude/skills/issuefs/SKILL.md
  global    ~/.claude/skills/issuefs/SKILL.md

The flag is repeatable, so '--install-skill project --install-skill global'
writes both. If the target file already exists and is byte-identical to the
bundled version, it's a silent no-op. If it differs, init refuses without
--force.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd.OutOrStdout(), cmd.ErrOrStderr(), o)
		},
	}
	c.Flags().StringSliceVar(&o.installSkill, "install-skill", nil, "also install the skill: project|global (repeatable)")
	c.Flags().BoolVar(&o.force, "force", false, "overwrite an existing skill file even if it differs from the bundled version")
	return c
}

func runInit(stdout, stderr io.Writer, o *initOpts) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Validate --install-skill targets up front.
	for _, t := range o.installSkill {
		if t != "project" && t != "global" {
			return fmt.Errorf("--install-skill: %q is not one of 'project' or 'global'", t)
		}
	}

	// Scaffold issues/ tree.
	root := filepath.Join(cwd, store.DirName)
	created, err := store.Scaffold(root)
	if err != nil {
		return err
	}
	if len(created) == 0 {
		fmt.Fprintln(stdout, "already initialized")
	} else {
		for _, p := range created {
			rel, relErr := filepath.Rel(cwd, p)
			if relErr != nil {
				rel = p
			}
			fmt.Fprintf(stdout, "created %s\n", rel)
		}
	}

	// Skill installation, if requested.
	if len(o.installSkill) == 0 {
		return nil
	}
	// Deduplicate target list (e.g. --install-skill project --install-skill project).
	targets := make([]string, 0, 2)
	for _, t := range o.installSkill {
		if !slices.Contains(targets, t) {
			targets = append(targets, t)
		}
	}
	for _, t := range targets {
		path, err := skillTargetPath(t, cwd)
		if err != nil {
			return err
		}
		if err := installSkill(stdout, stderr, cwd, path, o.force); err != nil {
			return err
		}
	}
	return nil
}

// skillTargetPath returns the absolute path the skill should be installed to
// for the given target name.
func skillTargetPath(target, cwd string) (string, error) {
	switch target {
	case "project":
		return filepath.Join(cwd, ".claude", embedded.SkillRelDir, embedded.SkillFilename), nil
	case "global":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("--install-skill global: cannot resolve home directory: %w", err)
		}
		return filepath.Join(home, ".claude", embedded.SkillRelDir, embedded.SkillFilename), nil
	default:
		return "", fmt.Errorf("internal: unknown target %q", target)
	}
}

// installSkill writes the bundled skill content to path. If a file already
// exists and is byte-identical, it's a silent no-op (with a status line).
// If it differs, refuses unless force is true.
func installSkill(stdout, stderr io.Writer, cwd, path string, force bool) error {
	rel, relErr := filepath.Rel(cwd, path)
	if relErr != nil {
		rel = path
	}

	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if bytes.Equal(existing, embedded.Skill) {
			fmt.Fprintf(stdout, "skill already installed at %s (matches bundled version)\n", rel)
			return nil
		}
		if !force {
			return fmt.Errorf("skill at %s differs from bundled version; pass --force to overwrite", rel)
		}
		// fall through to write
	case os.IsNotExist(err):
		// new install; fall through
	default:
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, embedded.Skill, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "created %s\n", rel)
	return nil
}
