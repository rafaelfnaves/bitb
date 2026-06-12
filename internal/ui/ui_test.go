package ui

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDate_Empty(t *testing.T) {
	if got := FormatDate(""); got != "—" {
		t.Errorf("FormatDate(\"\") = %q, want \"—\"", got)
	}
}

func TestFormatDate_JustNow(t *testing.T) {
	iso := time.Now().UTC().Format(time.RFC3339)
	got := FormatDate(iso)
	if got != "just now" && !strings.HasSuffix(got, "m ago") {
		t.Errorf("FormatDate(now) = %q, want \"just now\" or \"Nm ago\"", got)
	}
}

func TestFormatDate_Minutes(t *testing.T) {
	iso := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	got := FormatDate(iso)
	if got != "5m ago" {
		t.Errorf("FormatDate(-5m) = %q, want \"5m ago\"", got)
	}
}

func TestFormatDate_Hours(t *testing.T) {
	iso := time.Now().UTC().Add(-3 * time.Hour).Format(time.RFC3339)
	got := FormatDate(iso)
	if got != "3h ago" {
		t.Errorf("FormatDate(-3h) = %q, want \"3h ago\"", got)
	}
}

func TestFormatDate_Days(t *testing.T) {
	iso := time.Now().UTC().Add(-15 * 24 * time.Hour).Format(time.RFC3339)
	got := FormatDate(iso)
	if got != "15d ago" {
		t.Errorf("FormatDate(-15d) = %q, want \"15d ago\"", got)
	}
}

func TestFormatDate_OldDate(t *testing.T) {
	got := FormatDate("2024-01-15T10:00:00+00:00")
	if got != "2024-01-15" {
		t.Errorf("FormatDate(old) = %q, want \"2024-01-15\"", got)
	}
}

func TestFormatDate_WithZ(t *testing.T) {
	got := FormatDate("2024-01-15T10:00:00Z")
	if got != "2024-01-15" {
		t.Errorf("FormatDate(Z suffix) = %q, want \"2024-01-15\"", got)
	}
}

func TestTruncate_Short(t *testing.T) {
	if got := Truncate("hello", 10); got != "hello" {
		t.Errorf("Truncate(short) = %q, want \"hello\"", got)
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	if got := Truncate("hello", 5); got != "hello" {
		t.Errorf("Truncate(exact) = %q, want \"hello\"", got)
	}
}

func TestTruncate_Long(t *testing.T) {
	got := Truncate("hello world", 6)
	if got != "hello…" {
		t.Errorf("Truncate(long) = %q, want \"hello…\"", got)
	}
}

func TestMaskToken_Short(t *testing.T) {
	if got := MaskToken("abc"); got != "****" {
		t.Errorf("MaskToken(short) = %q, want \"****\"", got)
	}
}

func TestMaskToken_ExactEight(t *testing.T) {
	if got := MaskToken("12345678"); got != "****" {
		t.Errorf("MaskToken(8) = %q, want \"****\"", got)
	}
}

func TestMaskToken_Normal(t *testing.T) {
	got := MaskToken("abcdefghijkl")
	if got != "abcd...ijkl" {
		t.Errorf("MaskToken(normal) = %q, want \"abcd...ijkl\"", got)
	}
}

func TestPRState(t *testing.T) {
	cases := []struct {
		input    string
		contains string
	}{
		{"OPEN", "OPEN"},
		{"MERGED", "MERGED"},
		{"DECLINED", "DECLINED"},
		{"SUPERSEDED", "SUPERSEDED"},
		{"unknown", "unknown"},
	}
	for _, tc := range cases {
		got := PRState(tc.input)
		if !strings.Contains(got, tc.contains) {
			t.Errorf("PRState(%q) = %q, want to contain %q", tc.input, got, tc.contains)
		}
	}
}

func TestPipelineState(t *testing.T) {
	cases := []struct {
		state, result, contains string
	}{
		{"COMPLETED", "SUCCESSFUL", "PASSED"},
		{"COMPLETED", "FAILED", "FAILED"},
		{"COMPLETED", "ERROR", "ERROR"},
		{"COMPLETED", "STOPPED", "STOPPED"},
		{"IN_PROGRESS", "", "RUNNING"},
		{"PENDING", "", "PENDING"},
		{"HALTED", "", "HALTED"},
	}
	for _, tc := range cases {
		got := PipelineState(tc.state, tc.result)
		if !strings.Contains(got, tc.contains) {
			t.Errorf("PipelineState(%q, %q) = %q, want to contain %q", tc.state, tc.result, got, tc.contains)
		}
	}
}

func TestVisLen_PlainText(t *testing.T) {
	if got := visLen("hello"); got != 5 {
		t.Errorf("visLen(plain) = %d, want 5", got)
	}
}

func TestVisLen_WithANSI(t *testing.T) {
	s := "\x1b[31mred\x1b[0m"
	if got := visLen(s); got != 3 {
		t.Errorf("visLen(ansi) = %d, want 3", got)
	}
}

func TestTable_Render(t *testing.T) {
	tbl := NewTable("A", "B")
	tbl.AddRow("foo", "bar")
	tbl.AddRow("longer", "x")
	out := tbl.Render()

	if !strings.Contains(out, "foo") || !strings.Contains(out, "longer") {
		t.Errorf("Table.Render() missing row content: %s", out)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 { // header + separator + 2 rows
		t.Errorf("Table.Render() got %d lines, want 4", len(lines))
	}
}
