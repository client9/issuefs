package issue

import (
	"slices"
	"time"
)

// Valid issue states. Each is also the directory name under issues/.
const (
	StateBacklog = "backlog"
	StateActive  = "active"
	StateDone    = "done"
)

func ValidStates() []string { return []string{StateBacklog, StateActive, StateDone} }

func IsValidState(s string) bool {
	return slices.Contains(ValidStates(), s)
}

// Issue is the on-disk representation of a single issue. The full field set
// is always emitted (no omitempty) so each file is self-describing for AI
// and automation consumers.
type Issue struct {
	Title     string    `json:"title"`
	ID        string    `json:"id"`
	State     string    `json:"state"`
	Created   time.Time `json:"created"`
	Labels    []string  `json:"labels"`
	Assignees []string  `json:"assignees"`
	Milestone string    `json:"milestone"`
	Projects  []string  `json:"projects"`
	Template  string    `json:"template"`
	Events    []Event   `json:"events"`
	Body      string    `json:"-"`
}

// New returns an Issue with empty (non-nil) slices so it round-trips to
// JSON arrays rather than nulls.
func New() *Issue {
	return &Issue{
		Labels:    []string{},
		Assignees: []string{},
		Projects:  []string{},
		Events:    []Event{},
	}
}
