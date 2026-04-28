package issue

import (
	"strings"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 4, 27, 14, 30, 22, 0, time.UTC)

func goodIssue() *Issue {
	iss := New()
	iss.Title = "ok"
	iss.ID = "20260427T143022Z-9f2a4b7c"
	iss.State = StateBacklog
	iss.Created = fixedTime
	iss.Events = []Event{NewFiled(fixedTime, StateBacklog)}
	return iss
}

func goodFile() string {
	iss := goodIssue()
	iss.Body = "hello"
	out, _ := Marshal(iss)
	return string(out)
}

func TestVerify_Good(t *testing.T) {
	if _, err := Verify(strings.NewReader(goodFile())); err != nil {
		t.Errorf("expected ok, got %v", err)
	}
}

func TestVerify_GoodEmptyBody(t *testing.T) {
	out, _ := Marshal(goodIssue())
	if _, err := Verify(strings.NewReader(string(out))); err != nil {
		t.Errorf("expected ok, got %v", err)
	}
}

func TestVerify_GoodWithMoves(t *testing.T) {
	iss := goodIssue()
	iss.State = StateDone
	iss.Events = append(iss.Events,
		NewMoved(fixedTime.Add(time.Hour), StateBacklog, StateActive),
		NewMoved(fixedTime.Add(2*time.Hour), StateActive, StateDone),
	)
	out, _ := Marshal(iss)
	if _, err := Verify(strings.NewReader(string(out))); err != nil {
		t.Errorf("expected ok, got %v", err)
	}
}

func TestVerify_StateMismatch(t *testing.T) {
	iss := goodIssue()
	iss.State = StateActive // events say backlog
	out, _ := Marshal(iss)
	if _, err := Verify(strings.NewReader(string(out))); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Errorf("expected state-mismatch error, got %v", err)
	}
}

func TestVerify_NoEvents(t *testing.T) {
	iss := goodIssue()
	iss.Events = []Event{}
	out, _ := Marshal(iss)
	if _, err := Verify(strings.NewReader(string(out))); err == nil || !strings.Contains(err.Error(), "non-empty") {
		t.Errorf("expected non-empty error, got %v", err)
	}
}

func TestVerify_FirstEventNotFiled(t *testing.T) {
	iss := goodIssue()
	iss.Events = []Event{NewMoved(fixedTime, StateBacklog, StateActive)}
	iss.State = StateActive
	out, _ := Marshal(iss)
	if _, err := Verify(strings.NewReader(string(out))); err == nil || !strings.Contains(err.Error(), "filed") {
		t.Errorf("expected filed-first error, got %v", err)
	}
}

func TestVerify_FiledTsMismatch(t *testing.T) {
	iss := goodIssue()
	iss.Events = []Event{NewFiled(fixedTime.Add(time.Minute), StateBacklog)}
	out, _ := Marshal(iss)
	if _, err := Verify(strings.NewReader(string(out))); err == nil || !strings.Contains(err.Error(), "must equal created") {
		t.Errorf("expected ts-mismatch error, got %v", err)
	}
}

func TestVerify_NonMonotonic(t *testing.T) {
	iss := goodIssue()
	iss.State = StateActive
	iss.Events = []Event{
		NewFiled(fixedTime, StateBacklog),
		NewMoved(fixedTime.Add(2*time.Hour), StateBacklog, StateActive),
		NewMoved(fixedTime.Add(time.Hour), StateActive, StateBacklog),
	}
	out, _ := Marshal(iss)
	if _, err := Verify(strings.NewReader(string(out))); err == nil || !strings.Contains(err.Error(), "monotonic") {
		t.Errorf("expected monotonic error, got %v", err)
	}
}

func TestVerify_UnknownField(t *testing.T) {
	in := strings.Replace(goodFile(), `"events"`, `"lables":[],"events"`, 1)
	if _, err := Verify(strings.NewReader(in)); err == nil {
		t.Errorf("expected unknown-field error")
	}
}

func TestVerify_MissingTitle(t *testing.T) {
	iss := goodIssue()
	iss.Title = ""
	out, _ := Marshal(iss)
	if _, err := Verify(strings.NewReader(string(out))); err == nil {
		t.Errorf("expected missing-title error")
	}
}

func TestVerify_BadState(t *testing.T) {
	iss := goodIssue()
	iss.State = "wip"
	iss.Events = []Event{NewFiled(fixedTime, "wip")}
	out, _ := Marshal(iss)
	if _, err := Verify(strings.NewReader(string(out))); err == nil || !strings.Contains(err.Error(), "state") {
		t.Errorf("expected bad-state error, got %v", err)
	}
}

func TestVerify_BadSeparator(t *testing.T) {
	out, _ := Marshal(goodIssue())
	in := strings.TrimRight(string(out), "\n") + "body"
	if _, err := Verify(strings.NewReader(in)); err == nil {
		t.Errorf("expected separator error")
	}
}

func TestVerify_TooManyBlankLines(t *testing.T) {
	out, _ := Marshal(goodIssue())
	in := strings.TrimRight(string(out), "\n") + "\n\n\nbody\n"
	if _, err := Verify(strings.NewReader(in)); err == nil {
		t.Errorf("expected separator error")
	}
}

func TestVerify_NotJSON(t *testing.T) {
	if _, err := Verify(strings.NewReader("not json\n")); err == nil {
		t.Errorf("expected decode error")
	}
}
