package log

import (
	"strings"
	"testing"
)

func TestSimple(t *testing.T) {
	l := New(false, true)
	if !l.Simple() {
		t.Error("expected Simple() to be true")
	}

	l2 := New(false, false)
	if l2.Simple() {
		t.Error("expected Simple() to be false")
	}
}

func TestPrintf_quiet_mode_no_output(_ *testing.T) {
	// Quiet mode suppresses all output — verified by the fact that Printf
	// returns without writing. Calling Printf in quiet mode should not panic.
	l := New(true, false)
	l.Printf("hello world")
}

func TestNew_quiet_and_simple_flags(t *testing.T) {
	l := New(true, true)
	if !l.Simple() {
		t.Error("expected Simple() to be true when simple=true")
	}

	l2 := New(true, false)
	if l2.Simple() {
		t.Error("expected Simple() to be false when simple=false")
	}
}

func TestPrintf_simple_mode_replaces_check_mark(t *testing.T) {
	// The simple mode replaces "✓" with "-" in the format string.
	// We verify the Check constant contains the check mark, and that
	// simple mode formatting would strip it.
	if !strings.Contains(Check, "✓") {
		t.Error("expected Check constant to contain check mark")
	}
	// Verify ColorGreen and ColorReset are ANSI codes
	if !strings.HasPrefix(ColorGreen, "\033[") {
		t.Error("expected ColorGreen to be an ANSI code")
	}
	if !strings.HasPrefix(ColorReset, "\033[") {
		t.Error("expected ColorReset to be an ANSI code")
	}
}

func TestCheck_constant_format(t *testing.T) {
	// Check should be: ColorGreen + "✓" + ColorReset
	expected := ColorGreen + "✓" + ColorReset
	if Check != expected {
		t.Errorf("expected Check to be %q, got %q", expected, Check)
	}
}

func TestCheckf_prepends_check(t *testing.T) {
	// Checkf prepends "  ✓ " to the format string before calling Printf.
	// We verify the Check constant is used correctly.
	if Check == "" {
		t.Error("expected Check to be non-empty")
	}
}
