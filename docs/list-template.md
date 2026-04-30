# `ifs list --template` Data Model

`ifs list --template <file>` renders a Go `text/template` against a single input object. The object contains the filtered, sorted, and limited issues that `ifs list` would otherwise display.

## Top-Level Object

The template receives a value with these fields:

```go
type listTemplateData struct {
    Entries []listItem
    Count   int
    Now     time.Time
}
```

- `Entries` - the list rows, already filtered, sorted, and truncated by `--limit`.
- `Count` - number of entries in `Entries`.
- `Now` - current UTC time when rendering started.

## Entry Object

Each item in `Entries` has this shape:

```go
type listItem struct {
    Match store.Match
    Issue *issue.Issue
}
```

### `Match`

`Match` comes from the store resolver and identifies the file in two ways:
`Path` is the repo-relative URL-style path for templates, while `AbsPath` is the
filesystem path used by the CLI.

```go
type Match struct {
    Path    string
    AbsPath string
    State string
    Name  string
    ID    string
    Short string
}
```

- `Path` - repo-relative, URL-friendly path to the issue file, for example
  `issues/done/20260428T004315Z-9f2a4b7c-fix-rockets.md`.
- `AbsPath` - absolute filesystem path to the issue file, used internally by
  `ifs` when reading or moving files.
- `State` - issue state directory: `backlog`, `active`, or `done`.
- `Name` - filename, for example `20260428T004315Z-9f2a4b7c-fix-rockets.md`.
- `ID` - the full issue ID, for example `20260428T004315Z-9f2a4b7c`.
- `Short` - the 8-character random suffix.

### `Issue`

`Issue` is the parsed frontmatter + body from [internal/issue/issue.go](../internal/issue/issue.go).

```go
type Issue struct {
    Title     string
    ID        string
    State     string
    Created   time.Time
    Labels    []string
    Assignees []string
    Milestone string
    Projects  []string
    Template  string
    Events    []Event
    Body      string
}
```

- `Title` - issue title.
- `ID` - full issue ID.
- `State` - current state stored in frontmatter.
- `Created` - creation timestamp.
- `Labels` - labels as a string slice.
- `Assignees` - assignees as a string slice.
- `Milestone` - milestone name, or `""` if unset.
- `Projects` - project names as a string slice.
- `Template` - issue template name, if any.
- `Events` - append-only event log.
- `Body` - markdown body text.

## Available Template Functions

The template uses `github.com/client9/doublebrace` plus a small local helper set.

### From `doublebrace`

Commonly useful functions include:

- `default`
- `cond`
- `dict`
- `list`
- `join`
- `split`
- `lenRunes`
- `truncate`
- `replace`
- `lower`
- `upper`
- `contains`
- `hasPrefix`
- `hasSuffix`
- `pathBase`
- `pathDir`
- `pathExt`
- `pathJoin`
- `now`
- `parseTime`

See the `doublebrace` package docs for the full set.

### Local Helpers

- `mdCell` - escape a string for use inside a markdown table cell.
- `escapeCell` - same behavior as `mdCell`.

## Example

```gotemplate
{{- if .Entries -}}
| ID | State | Title | Created |
| --- | --- | --- | --- |
{{- range .Entries }}
| {{ mdCell .Match.Short }} | {{ mdCell .Match.State }} | {{ mdCell .Issue.Title }} | {{ mdCell (.Issue.Created.Format "2006-01-02") }} |
{{- end }}
{{- end -}}
```

## Notes

- `ifs list` already applies state, label, assignee, milestone, `--since`, sort, and `--limit` before the template runs.
- Markdown templates are rendered through the existing markdown pipeline.
- Non-markdown templates emit plain text directly.
