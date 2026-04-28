package issue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Filename returns "<id>-<slug>.md".
func Filename(id, slug string) string {
	if slug == "" {
		return id + ".md"
	}
	return id + "-" + slug + ".md"
}

// Marshal renders an Issue as bare-JSON frontmatter followed by the body.
// Layout: pretty-printed JSON object, then (if body is non-empty) a blank
// line and the body. The file always ends in a single newline.
func Marshal(iss *Issue) ([]byte, error) {
	js, err := json.MarshalIndent(iss, "", "  ")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Write(js)
	buf.WriteByte('\n')
	if iss.Body != "" {
		buf.WriteByte('\n')
		buf.WriteString(strings.TrimRight(iss.Body, "\n"))
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// Parse reads a single bare-JSON frontmatter object followed by an optional
// markdown body. Body leading newlines are trimmed.
func Parse(r io.Reader) (*Issue, error) {
	iss, _, err := parse(r, false)
	return iss, err
}

// Verify is like Parse but stricter: unknown fields are rejected, required
// fields must be present and valid, and the separator between frontmatter
// and body must be exactly one blank line (or EOF for an empty body).
func Verify(r io.Reader) (*Issue, error) {
	iss, sep, err := parse(r, true)
	if err != nil {
		return nil, err
	}
	if err := validateSeparator(sep, iss.Body != ""); err != nil {
		return nil, err
	}
	if err := iss.validate(); err != nil {
		return nil, err
	}
	return iss, nil
}

// parse decodes the frontmatter and returns the issue plus the raw bytes
// between the closing "}" and the first body character (or EOF).
func parse(r io.Reader, strict bool) (*Issue, string, error) {
	dec := json.NewDecoder(r)
	if strict {
		dec.DisallowUnknownFields()
	}
	iss := New()
	if err := dec.Decode(iss); err != nil {
		return nil, "", fmt.Errorf("decode frontmatter: %w", err)
	}
	rest, err := io.ReadAll(io.MultiReader(dec.Buffered(), r))
	if err != nil {
		return nil, "", err
	}
	s := string(rest)
	bodyStart := 0
	for bodyStart < len(s) && s[bodyStart] == '\n' {
		bodyStart++
	}
	sep := s[:bodyStart]
	body := strings.TrimRight(s[bodyStart:], "\n")
	iss.Body = body
	return iss, sep, nil
}

func validateSeparator(sep string, hasBody bool) error {
	if hasBody {
		if sep == "\n\n" {
			return nil
		}
		return fmt.Errorf("frontmatter and body must be separated by a single blank line, got %d newline(s)", len(sep))
	}
	if sep == "" || sep == "\n" {
		return nil
	}
	return fmt.Errorf("file with no body should end at the closing '}', got %d trailing newline(s)", len(sep))
}

func (i *Issue) validate() error {
	if strings.TrimSpace(i.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if i.ID == "" {
		return fmt.Errorf("id is required")
	}
	if i.State == "" {
		return fmt.Errorf("state is required")
	}
	if !IsValidState(i.State) {
		return fmt.Errorf("state %q is not one of %v", i.State, ValidStates())
	}
	if i.Created.IsZero() {
		return fmt.Errorf("created is required")
	}
	return i.validateEvents()
}

// validateEvents enforces the consistency contract between the events log
// and the frontmatter scalar fields:
//   - events is non-empty
//   - first event is "filed" with ts == created
//   - events are timestamp-monotonic (non-decreasing)
//   - the latest event with a non-empty "to" equals the current state
func (i *Issue) validateEvents() error {
	if len(i.Events) == 0 {
		return fmt.Errorf("events must be non-empty (at least the 'filed' event)")
	}
	first := i.Events[0]
	if first.Type != EventFiled {
		return fmt.Errorf("first event must be %q, got %q", EventFiled, first.Type)
	}
	if !first.Timestamp.Equal(i.Created) {
		return fmt.Errorf("first event ts (%s) must equal created (%s)", first.Timestamp.Format(time.RFC3339), i.Created.Format(time.RFC3339))
	}
	for k := 1; k < len(i.Events); k++ {
		if i.Events[k].Timestamp.Before(i.Events[k-1].Timestamp) {
			return fmt.Errorf("events must be timestamp-monotonic (event %d is before event %d)", k, k-1)
		}
	}
	latestTo := ""
	for _, e := range i.Events {
		if e.To != "" {
			latestTo = e.To
		}
	}
	if latestTo == "" {
		return fmt.Errorf("no event carries a 'to' field; cannot derive expected state")
	}
	if latestTo != i.State {
		return fmt.Errorf("state %q does not match latest event 'to' %q", i.State, latestTo)
	}
	return nil
}
