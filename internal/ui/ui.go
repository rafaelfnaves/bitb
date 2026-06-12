package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	StyleTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	StyleID      = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	StyleGreen   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	StyleRed     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	StyleYellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	StyleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	StyleBold    = lipgloss.NewStyle().Bold(true)
	StyleCyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	StyleMagenta = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
)

// PRState returns a colored status string for a PR state.
func PRState(state string) string {
	switch strings.ToUpper(state) {
	case "OPEN":
		return StyleGreen.Render("● OPEN")
	case "MERGED":
		return StyleMagenta.Render("⎇ MERGED")
	case "DECLINED":
		return StyleRed.Render("✗ DECLINED")
	case "SUPERSEDED":
		return StyleDim.Render("~ SUPERSEDED")
	default:
		return state
	}
}

// PipelineState returns a colored status string for a pipeline state.
func PipelineState(state, result string) string {
	switch strings.ToUpper(result) {
	case "SUCCESSFUL":
		return StyleGreen.Render("✓ PASSED")
	case "FAILED":
		return StyleRed.Render("✗ FAILED")
	case "ERROR":
		return StyleRed.Render("✗ ERROR")
	case "STOPPED":
		return StyleDim.Render("■ STOPPED")
	}
	switch strings.ToUpper(state) {
	case "IN_PROGRESS":
		return StyleYellow.Render("● RUNNING")
	case "PENDING":
		return StyleCyan.Render("○ PENDING")
	case "HALTED":
		return StyleDim.Render("■ HALTED")
	}
	return StyleDim.Render(state)
}

// Table renders a simple text table.
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{headers: headers, widths: widths}
}

func (t *Table) AddRow(cells ...string) {
	row := make([]string, len(t.headers))
	for i := 0; i < len(t.headers) && i < len(cells); i++ {
		row[i] = cells[i]
		if visLen(cells[i]) > t.widths[i] {
			t.widths[i] = visLen(cells[i])
		}
	}
	t.rows = append(t.rows, row)
}

func (t *Table) Render() string {
	var sb strings.Builder

	// Header
	for i, h := range t.headers {
		sb.WriteString(StyleBold.Render(pad(h, t.widths[i])))
		if i < len(t.headers)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")

	// Separator
	for i, w := range t.widths {
		sb.WriteString(StyleDim.Render(strings.Repeat("─", w)))
		if i < len(t.widths)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range t.rows {
		for i, cell := range row {
			sb.WriteString(pad(cell, t.widths[i]))
			if i < len(row)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatDate converts an ISO8601 date string to a human-readable relative time.
func FormatDate(iso string) string {
	if iso == "" {
		return "—"
	}
	// Normalize Z suffix
	iso = strings.Replace(iso, "Z", "+00:00", 1)
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso[:10]
	}
	now := time.Now().UTC()
	delta := now.Sub(t.UTC())

	switch {
	case delta < time.Hour:
		mins := int(delta.Minutes())
		if mins < 1 {
			return "just now"
		}
		return fmt.Sprintf("%dm ago", mins)
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	case delta < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(delta.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// Truncate shortens a string to max visible characters, adding ellipsis.
func Truncate(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max-1]) + "…"
}

// MaskToken shows first 4 and last 4 characters of a token.
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// pad pads a string with spaces to width, accounting for lipgloss ANSI codes.
func pad(s string, width int) string {
	vl := visLen(s)
	if vl >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vl)
}

// visLen returns the visible length of a string (strips ANSI escape codes for width calculation).
func visLen(s string) int {
	// Simple approximation: count runes, subtract escape sequences
	inEsc := false
	count := 0
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		count++
	}
	return count
}
