{
  "title": "Implement template-driven list output with doublebrace helpers",
  "id": "20260430T041524Z-95aaca75",
  "state": "done",
  "created": "2026-04-30T04:15:24Z",
  "labels": [
    "feature",
    "design"
  ],
  "assignees": [],
  "milestone": "",
  "projects": [],
  "template": "",
  "events": [
    {
      "ts": "2026-04-30T04:15:24Z",
      "type": "filed",
      "to": "backlog"
    },
    {
      "ts": "2026-04-30T04:15:35Z",
      "type": "moved",
      "from": "backlog",
      "to": "active"
    },
    {
      "ts": "2026-04-30T04:19:58Z",
      "type": "moved",
      "from": "active",
      "to": "done"
    }
  ]
}

## Concept
Replace the hardwired `ifs list` output path with a `--template <file>` flag. With no flag, keep the current list output by rendering it through a builtin template. When the template source file ends in `.md` or another markdown extension, pass the rendered text through the existing markdown rendering pipeline; otherwise emit plain text directly.

Use `github.com/client9/doublebrace` for reusable template helper functions. Add local helpers only if doublebrace does not cover a needed case cleanly.

## Plan
- Add a template field to list options and remove the current output-mode hardwiring.
- Introduce a template input struct that carries the filtered/sorted entries plus any metadata needed by templates.
- Load templates from either an embedded builtin template or a user-supplied file.
- Classify templates by file extension to decide markdown rendering vs plain-text output.
- Keep the builtin template stable enough to reproduce the current list table.
- Reuse the existing markdown renderer for markdown templates; do not add a second rendering path.

## Template helpers
- Use doublebrace for general template helpers such as `join`, `joinOrNone`, `orNone`, `formatDate`, `dict`, and any comparable convenience helpers it already provides.
- Add local helpers only for repo-specific formatting that doublebrace does not cover well, such as markdown cell escaping or list-specific row assembly.

## Tests
- Default output matches the current list shape when no template flag is supplied.
- Markdown template files render through the existing markdown renderer.
- Non-markdown template files emit plain text only.
- Template parse and execution errors surface cleanly.
- Empty results still produce empty output, not a blank rendered page.
- Existing list filters, sorting, and limits still work.
- Titles containing `|` still survive the builtin template path correctly.

## Files likely involved
- `cmd/list.go`
- `cmd/list_test.go`
- embedded builtin template resources
- a small template-render helper in `cmd/`

## Resolution

Implemented as designed, with `--template` replacing the old hardwired output path.

What landed:
- `cmd/list.go` now filters/sorts as before, then renders through a template-aware output path.
- `cmd/list_template.go` loads builtin or file-based templates, uses `doublebrace` helpers, and routes markdown templates through the existing renderer.
- `internal/embedded/templates/list.md` holds the builtin list template.
- `cmd/list_test.go` covers default output, markdown vs plain templates, helper escaping, extension detection, filter/sort/limit behavior, and template errors.

Validation:
- `go test ./...`
