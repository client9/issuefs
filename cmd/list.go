package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/client9/nowandlater"
	"github.com/nickg/issuefs/internal/issue"
	"github.com/nickg/issuefs/internal/store"
	"github.com/spf13/cobra"
)

// listItem pairs a Match with its loaded Issue. Fields are exported so text
// templates can access them directly.
type listItem struct {
	Match store.Match
	Issue *issue.Issue
}

type listOpts struct {
	states     []string
	labels     []string
	assignees  []string
	milestones []string
	limit      int
	sort       string
	since      string
	template   string
}

func newList() *cobra.Command {
	o := &listOpts{}
	c := &cobra.Command{
		Use:   "list",
		Short: "List issues with filters",
		Long: `Enumerate issues across state directories, applying filters.

Default state filter is "backlog,active" (matches gh's "open" default).
Output is driven by a Go text/template. With no --template flag, list renders
its built-in template, which matches the current tabular output. Markdown
templates are rendered through the existing markdown pipeline; other templates
emit plain text.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.OutOrStdout(), o)
		},
	}
	f := c.Flags()
	f.StringSliceVarP(&o.states, "state", "s", []string{"backlog", "active"}, "filter by state (repeatable; backlog|active|done|all)")
	f.StringSliceVarP(&o.labels, "label", "l", nil, "filter by label (repeatable, AND semantics)")
	f.StringSliceVarP(&o.assignees, "assignee", "a", nil, "filter by assignee (repeatable, AND semantics)")
	f.StringSliceVarP(&o.milestones, "milestone", "m", nil, "filter by milestone (repeatable; OR semantics — matches if any value is the milestone)")
	f.IntVarP(&o.limit, "limit", "L", 30, "maximum issues to display")
	f.StringVar(&o.sort, "sort", "created", "sort order: created|updated (desc)")
	f.StringVar(&o.since, "since", "", "only include issues created/updated on or after this date (ISO or 'last month', '3 days ago', etc.)")
	f.StringVar(&o.template, "template", "", "template file used to render the list output")
	return c
}

func runList(stdout io.Writer, o *listOpts) error {
	switch o.sort {
	case "created", "updated":
		// ok
	default:
		return fmt.Errorf("--sort must be one of created, updated (got %q)", o.sort)
	}

	states, err := normalizeStates(o.states)
	if err != nil {
		return err
	}

	var sinceT time.Time
	if o.since != "" {
		sinceT, err = parseSince(o.since)
		if err != nil {
			return fmt.Errorf("--since: %w", err)
		}
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
		return renderList(stdout, listTemplateData{}, o.template)
	}

	r, err := store.NewResolver(root)
	if err != nil {
		return err
	}

	matches := r.All()
	entries := make([]listItem, 0, len(matches))
	for _, m := range matches {
		if !slices.Contains(states, m.State) {
			continue
		}
		iss, err := readIssue(m.AbsPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.AbsPath, err)
		}
		if !labelsMatch(iss.Labels, o.labels) {
			continue
		}
		if !assigneesMatch(iss.Assignees, o.assignees) {
			continue
		}
		if len(o.milestones) > 0 && !slices.Contains(o.milestones, iss.Milestone) {
			continue
		}
		if !sinceT.IsZero() {
			ts := issueSortTime(iss, o.sort)
			if ts.Before(sinceT) {
				continue
			}
		}
		entries = append(entries, listItem{Match: m, Issue: iss})
	}

	// Sort: descending by created or updated time.
	sort.Slice(entries, func(i, j int) bool {
		ti := issueSortTime(entries[i].Issue, o.sort)
		tj := issueSortTime(entries[j].Issue, o.sort)
		return ti.After(tj)
	})

	if o.limit > 0 && len(entries) > o.limit {
		entries = entries[:o.limit]
	}

	return renderList(stdout, listTemplateData{Entries: entries, Count: len(entries), Now: time.Now().UTC()}, o.template)
}

func normalizeStates(states []string) ([]string, error) {
	if len(states) == 0 {
		return []string{"backlog", "active"}, nil
	}
	out := make([]string, 0, len(states))
	for _, s := range states {
		if s == "all" {
			return issue.ValidStates(), nil
		}
		if !issue.IsValidState(s) {
			return nil, fmt.Errorf("--state value %q is not one of %v (or 'all')", s, issue.ValidStates())
		}
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out, nil
}

// labelsMatch returns true if every required label is present on the issue
// (AND semantics, matching gh).
func labelsMatch(have, want []string) bool {
	for _, w := range want {
		if !slices.Contains(have, w) {
			return false
		}
	}
	return true
}

// assigneesMatch returns true if every required assignee is on the issue.
func assigneesMatch(have, want []string) bool {
	for _, w := range want {
		if !slices.Contains(have, w) {
			return false
		}
	}
	return true
}

// issueSortTime returns the timestamp used for sorting and `--since` filtering.
// "created" uses iss.Created; "updated" uses the latest event timestamp.
func issueSortTime(iss *issue.Issue, key string) time.Time {
	if key == "updated" && len(iss.Events) > 0 {
		return iss.Events[len(iss.Events)-1].Timestamp
	}
	return iss.Created
}

// parseSince accepts ISO 8601 dates and ad-hoc human formats via nowandlater.
// Tries the single-instant Parse first, falls back to ParseInterval (using the
// start of the interval) for phrases like "last month".
func parseSince(s string) (time.Time, error) {
	p := nowandlater.Parser{}
	if t, err := p.Parse(s); err == nil {
		return t, nil
	}
	start, _, err := p.ParseInterval(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse %q as a date or interval: %w", s, err)
	}
	return start, nil
}
