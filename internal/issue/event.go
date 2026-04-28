package issue

import "time"

// Event types. Keep this vocabulary small; add a constant only when there's
// a verb that emits it.
const (
	EventFiled = "filed"
	EventMoved = "moved"
)

// Event is a single entry in an issue's history. ts and type are always
// present; type-specific fields use omitempty because events form a
// discriminated union — different rule than the flat top-level frontmatter.
type Event struct {
	Timestamp time.Time `json:"ts"`
	Type      string    `json:"type"`
	From      string    `json:"from,omitempty"`
	To        string    `json:"to,omitempty"`
}

// NewFiled returns the canonical "filed" event recorded at issue creation.
// to is the initial state.
func NewFiled(ts time.Time, to string) Event {
	return Event{Timestamp: ts.UTC().Truncate(time.Second), Type: EventFiled, To: to}
}

// NewMoved returns a "moved" event for a state transition.
func NewMoved(ts time.Time, from, to string) Event {
	return Event{Timestamp: ts.UTC().Truncate(time.Second), Type: EventMoved, From: from, To: to}
}
