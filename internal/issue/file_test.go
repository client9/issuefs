package issue

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMarshalAlwaysEmitsFullFieldSet(t *testing.T) {
	iss := New()
	iss.Title = "hi"
	iss.ID = "20260427T143022Z-9f2a4b7c"
	iss.State = StateBacklog
	iss.Created = time.Date(2026, 4, 27, 14, 30, 22, 0, time.UTC)

	out, err := Marshal(iss)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]any
	dec := json.NewDecoder(bytes.NewReader(out))
	if err := dec.Decode(&raw); err != nil {
		t.Fatalf("decode: %v\nfile:\n%s", err, out)
	}
	for _, k := range []string{"title", "id", "state", "created", "labels", "assignees", "milestone", "projects", "template"} {
		if _, ok := raw[k]; !ok {
			t.Errorf("missing field %q in: %s", k, out)
		}
	}
	// Empty slices, not null.
	if !strings.Contains(string(out), `"labels": []`) {
		t.Errorf("labels should be [], got: %s", out)
	}
}

func TestRoundTripWithBody(t *testing.T) {
	iss := New()
	iss.Title = "Fix space rocket thrusters"
	iss.ID = "20260427T143022Z-9f2a4b7c"
	iss.State = StateBacklog
	iss.Created = time.Date(2026, 4, 27, 14, 30, 22, 0, time.UTC)
	iss.Labels = []string{"bug", "urgent"}
	iss.Body = "This is the body.\n\nWith multiple paragraphs."

	out, err := Marshal(iss)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Parse(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("parse: %v\nfile:\n%s", err, out)
	}
	if got.Title != iss.Title {
		t.Errorf("title: got %q want %q", got.Title, iss.Title)
	}
	if got.Body != iss.Body {
		t.Errorf("body: got %q want %q", got.Body, iss.Body)
	}
	if !got.Created.Equal(iss.Created) {
		t.Errorf("created: got %v want %v", got.Created, iss.Created)
	}
	if len(got.Labels) != 2 || got.Labels[0] != "bug" {
		t.Errorf("labels: %v", got.Labels)
	}
}

func TestRoundTripEmptyBody(t *testing.T) {
	iss := New()
	iss.Title = "no body"
	iss.ID = "20260427T143022Z-0001"
	iss.State = StateBacklog
	iss.Created = time.Date(2026, 4, 27, 14, 30, 22, 0, time.UTC)

	out, err := Marshal(iss)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(out, []byte("\n\n")) != 0 {
		t.Errorf("empty-body file should not contain a blank line, got:\n%s", out)
	}
	got, err := Parse(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != "" {
		t.Errorf("body: got %q want empty", got.Body)
	}
}

func TestFilename(t *testing.T) {
	if got := Filename("20260427T143022Z-9f2a4b7c", "fix-space-rocket"); got != "20260427T143022Z-9f2a4b7c-fix-space-rocket.md" {
		t.Errorf("got %q", got)
	}
	if got := Filename("20260427T143022Z-9f2a4b7c", ""); got != "20260427T143022Z-9f2a4b7c.md" {
		t.Errorf("got %q", got)
	}
}
