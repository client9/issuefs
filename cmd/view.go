package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/mattn/go-isatty"
	"github.com/nickg/issuefs/internal/issue"
	"github.com/nickg/issuefs/internal/md"
	"github.com/nickg/issuefs/internal/store"
	"github.com/spf13/cobra"
)

type viewOpts struct {
	format   string
	noMeta   bool
	noEvents bool
}

func newView() *cobra.Command {
	o := &viewOpts{}
	c := &cobra.Command{
		Use:   "view <ref>",
		Short: "Render an issue for human reading",
		Long: `Render an issue file as a human-readable document.

Output is markdown assembled in-memory and rendered via charmbracelet/glamour.

Sections (each independently skippable):
  metadata table → body → timeline.

Formats:
  auto    ANSI on a TTY, ascii otherwise
  ansi    ANSI escapes (style from $GLAMOUR_STYLE, default "dark")
  ascii   plaintext (no escapes)
  json    full Issue struct (frontmatter only)
  raw     file contents verbatim
  raw-md  assembled markdown pre-glamour (debugging)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runView(cmd.OutOrStdout(), args[0], o)
		},
	}
	c.Flags().StringVar(&o.format, "format", "auto", "output format: auto|ansi|ascii|json|raw|raw-md")
	c.Flags().BoolVar(&o.noMeta, "no-meta", false, "omit the metadata table")
	c.Flags().BoolVar(&o.noEvents, "no-events", false, "omit the timeline section")
	return c
}

func runView(stdout io.Writer, ref string, o *viewOpts) error {
	switch o.format {
	case "auto", "ansi", "ascii", "json", "raw", "raw-md":
		// ok
	default:
		return fmt.Errorf("--format must be one of auto, ansi, ascii, json, raw, raw-md (got %q)", o.format)
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
		return fmt.Errorf("no issues directory found from %s", cwd)
	}

	r, err := store.NewResolver(root)
	if err != nil {
		return err
	}
	m, err := r.Lookup(ref)
	if err != nil {
		return err
	}

	// Formats that bypass glamour entirely.
	switch o.format {
	case "raw":
		data, err := os.ReadFile(m.AbsPath)
		if err != nil {
			return err
		}
		_, err = stdout.Write(data)
		return err
	case "json":
		iss, err := readIssue(m.AbsPath)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(iss)
	}

	iss, err := readIssue(m.AbsPath)
	if err != nil {
		return err
	}
	doc := assembleDocument(iss, m.Short, o)

	if o.format == "raw-md" {
		_, err := io.WriteString(stdout, doc)
		return err
	}

	style := pickStyle(o.format, stdout)
	rendered, err := renderMarkdown(doc, style)
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdout, rendered)
	return err
}

// pickStyle resolves --format to the glamour style name. "auto" looks at the
// writer to decide ansi vs ascii; explicit "ansi"/"ascii" honor the user's
// choice regardless of TTY.
func pickStyle(format string, w io.Writer) string {
	switch format {
	case "ascii":
		return "ascii"
	case "ansi":
		return ansiStyle()
	case "auto":
		if isTerminal(w) {
			return ansiStyle()
		}
		return "ascii"
	}
	return "ascii"
}

// ansiStyle returns the style to use for ANSI output. Honors GLAMOUR_STYLE if
// set; otherwise defaults to "dark" so we always emit ANSI escapes when the
// user explicitly requested ansi (glamour's "auto" can downgrade to ascii in
// non-TTY environments, which would surprise a user who passed --format ansi).
func ansiStyle() string {
	if s := os.Getenv("GLAMOUR_STYLE"); s != "" {
		return s
	}
	return "dark"
}

// isTerminal reports whether w is a terminal. Buffers and pipes are not.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

// assembleDocument builds the full markdown document for an issue. Sections
// are joined by horizontal rules; empty sections (or sections suppressed by
// flags) are omitted along with their separator.
func assembleDocument(iss *issue.Issue, short string, o *viewOpts) string {
	var b strings.Builder
	if !o.noMeta {
		appendSection(&b, assembleMetadata(iss, short))
	}
	if iss.Body != "" {
		appendSection(&b, strings.TrimRight(iss.Body, "\n")+"\n")
	}
	if !o.noEvents {
		appendSection(&b, assembleTimeline(iss))
	}
	return b.String()
}

// appendSection writes `body` to b. If b already has content, prepends a
// horizontal-rule separator. No-op for empty bodies — keeps separators from
// stacking around omitted sections.
func appendSection(b *strings.Builder, body string) {
	if body == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("\n---\n\n")
	}
	b.WriteString(body)
}

func assembleMetadata(iss *issue.Issue, short string) string {
	rows := [][]string{
		{"Title", md.EscapeCell(iss.Title)},
		{"ID", fmt.Sprintf("%s (`%s`)", md.EscapeCell(iss.ID), md.EscapeCell(short))},
		{"State", md.EscapeCell(iss.State)},
		{"Created", md.EscapeCell(iss.Created.Format("2006-01-02 15:04 MST"))},
		{"Labels", md.EscapeCell(joinOrNone(iss.Labels))},
		{"Assignees", md.EscapeCell(joinOrNone(iss.Assignees))},
		{"Milestone", md.EscapeCell(orNone(iss.Milestone))},
		{"Projects", md.EscapeCell(joinOrNone(iss.Projects))},
	}
	return md.Table([]string{"Field", "Value"}, rows)
}

// assembleTimeline renders the events log as a markdown table preceded by a
// "## Timeline" heading. Returns "" if there are no events (shouldn't happen
// in a verify-clean issue, but be defensive).
func assembleTimeline(iss *issue.Issue) string {
	if len(iss.Events) == 0 {
		return ""
	}
	rows := make([][]string, 0, len(iss.Events))
	for _, e := range iss.Events {
		rows = append(rows, []string{
			md.EscapeCell(e.Timestamp.Format("2006-01-02 15:04 MST")),
			md.EscapeCell(e.Type),
			md.EscapeCell(eventDetails(e)),
		})
	}
	var b strings.Builder
	b.WriteString("## Timeline\n\n")
	b.WriteString(md.Table([]string{"When", "Event", "Details"}, rows))
	return b.String()
}

// eventDetails renders the type-specific fields of an event into a single
// human-readable string. Add cases as new event types are introduced.
func eventDetails(e issue.Event) string {
	switch {
	case e.From != "" && e.To != "":
		return fmt.Sprintf("%s → %s", e.From, e.To)
	case e.To != "":
		return "→ " + e.To
	case e.From != "":
		return e.From + " →"
	default:
		return ""
	}
}

func joinOrNone(xs []string) string {
	if len(xs) == 0 {
		return "_(none)_"
	}
	return strings.Join(xs, ", ")
}

func orNone(s string) string {
	if s == "" {
		return "_(none)_"
	}
	return s
}

func renderMarkdown(doc, style string) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(0), // disable hard wrapping; let the terminal/pager handle it
	)
	if err != nil {
		return "", err
	}
	defer r.Close()
	return r.Render(doc)
}
