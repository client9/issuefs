package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/nickg/issuefs/internal/issue"
	"github.com/nickg/issuefs/internal/store"
	"github.com/spf13/cobra"
)

type createOpts struct {
	title     string
	body      string
	bodyFile  string
	assignees []string
	labels    []string
	milestone string
	projects  []string
	template  string
	state     string
}

func newCreate() *cobra.Command {
	o := &createOpts{}
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Long: `Create a new issue file under issues/<state>/.

Modeled on "gh issue create"; interactive flags (--editor, --web) are
intentionally omitted.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCreate(cmd.OutOrStdout(), cmd.InOrStdin(), o)
		},
	}
	f := c.Flags()
	f.StringVarP(&o.title, "title", "t", "", "issue title (required)")
	f.StringVarP(&o.body, "body", "b", "", "issue body")
	f.StringVarP(&o.bodyFile, "body-file", "F", "", "read body from file ('-' for stdin)")
	f.StringSliceVarP(&o.assignees, "assignee", "a", nil, "assign a person (repeatable)")
	f.StringSliceVarP(&o.labels, "label", "l", nil, "add a label (repeatable)")
	f.StringVarP(&o.milestone, "milestone", "m", "", "milestone")
	f.StringSliceVarP(&o.projects, "project", "p", nil, "add to a project (repeatable)")
	f.StringVarP(&o.template, "template", "T", "", "template name")
	f.StringVar(&o.state, "state", issue.StateBacklog, "initial state: backlog|active|done")
	_ = c.MarkFlagRequired("title")
	c.MarkFlagsMutuallyExclusive("body", "body-file")
	return c
}

func runCreate(stdout io.Writer, stdin io.Reader, o *createOpts) error {
	title := strings.TrimSpace(o.title)
	if title == "" {
		return fmt.Errorf("--title must not be empty")
	}
	if !issue.IsValidState(o.state) {
		return fmt.Errorf("--state must be one of %v", issue.ValidStates())
	}

	body := o.body
	if o.bodyFile != "" {
		b, err := readBody(stdin, o.bodyFile)
		if err != nil {
			return err
		}
		body = b
	}

	now := time.Now()
	iss := issue.New()
	iss.Title = title
	iss.ID = issue.NewID(now)
	iss.State = o.state
	iss.Created = now.UTC().Truncate(time.Second)
	if o.labels != nil {
		iss.Labels = o.labels
	}
	if o.assignees != nil {
		iss.Assignees = o.assignees
	}
	if o.projects != nil {
		iss.Projects = o.projects
	}
	iss.Milestone = o.milestone
	iss.Template = o.template
	iss.Body = body
	iss.Events = []issue.Event{issue.NewFiled(iss.Created, iss.State)}

	data, err := issue.Marshal(iss)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, found, err := store.Resolve(cwd)
	if err != nil {
		return err
	}
	if !found {
		fmt.Fprintf(stdout, "creating new issues directory at %s\n", root)
	}
	dir, err := store.EnsureSubdir(root, o.state)
	if err != nil {
		return err
	}
	name := issue.Filename(iss.ID, issue.Slug(iss.Title))
	path, err := store.WriteNew(dir, name, data)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, path)
	return nil
}

func readBody(stdin io.Reader, path string) (string, error) {
	if path == "-" {
		b, err := io.ReadAll(stdin)
		return string(b), err
	}
	b, err := os.ReadFile(path)
	return string(b), err
}
