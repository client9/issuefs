package cmd

import (
	"strings"
	"testing"
)

// withLdflags sets the package-level vars for the duration of the test.
func withLdflags(t *testing.T, v, c, d string) {
	t.Helper()
	oldV, oldC, oldD := version, commit, date
	version, commit, date = v, c, d
	t.Cleanup(func() {
		version, commit, date = oldV, oldC, oldD
	})
}

func runV(t *testing.T, args ...string) string {
	t.Helper()
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

func TestVersion_LdflagsAllSet(t *testing.T) {
	withLdflags(t, "0.2.0", "abc1234", "2026-05-01")
	got := strings.TrimSpace(runV(t, "version"))
	want := "ifs 0.2.0 (abc1234, 2026-05-01)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVersion_LdflagsOnlyVersion(t *testing.T) {
	withLdflags(t, "0.2.0", "", "")
	got := strings.TrimSpace(runV(t, "version"))
	want := "ifs 0.2.0"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVersion_Short(t *testing.T) {
	withLdflags(t, "0.2.0", "abc1234", "2026-05-01")
	got := strings.TrimSpace(runV(t, "version", "--short"))
	if got != "0.2.0" {
		t.Errorf("got %q, want %q", got, "0.2.0")
	}
}

func TestVersion_Fallback(t *testing.T) {
	// All ldflag vars empty: must fall back to debug.ReadBuildInfo or (devel).
	// In a test binary built with `go test`, ReadBuildInfo returns info but
	// Main.Version is "(devel)" and vcs.* settings are typically absent. We
	// only assert that output is non-empty, starts with "ifs ", and exits 0.
	withLdflags(t, "", "", "")
	got := strings.TrimSpace(runV(t, "version"))
	if !strings.HasPrefix(got, "ifs ") {
		t.Errorf("expected output to start with %q, got %q", "ifs ", got)
	}
	if got == "ifs " {
		t.Errorf("expected a version token after %q, got %q", "ifs ", got)
	}
}

func TestVersion_FallbackShort(t *testing.T) {
	withLdflags(t, "", "", "")
	got := strings.TrimSpace(runV(t, "version", "--short"))
	if got == "" {
		t.Errorf("--short with no ldflags should still produce something, got empty")
	}
}

func TestVersion_NoArgs(t *testing.T) {
	withLdflags(t, "0.2.0", "abc1234", "2026-05-01")
	root := newRoot()
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"version", "extra"})
	if err := root.Execute(); err == nil {
		t.Errorf("expected error for unexpected positional arg")
	}
}
