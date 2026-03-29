package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Colours ───────────────────────────────────────────────────────────────────

const (
	colHeader       = "#7DCFFF"
	colHeaderBg     = "#1a1b26"
	colNormal       = "#565f89"
	colSelected     = "#ffffff"
	colMatchNorm    = "#ff9e64"
	colMatchSel     = "#ffb86c"
	colPrompt       = "#9ece6a"
	colInput        = "#c0caf5"
	colStatus       = "#565f89"
	colError        = "#f7768e"
	colPaneBorder   = "#3b4261"
	colPaneTitle    = "#7DCFFF"
	colKeyLabel     = "#9ece6a"
	colKeyDesc      = "#c0caf5"
	colKeySep       = "#565f89"
	colSectionHead  = "#bb9af7"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colHeader)).
			Background(lipgloss.Color(colHeaderBg)).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colNormal))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colSelected)).
			Bold(true)

	matchCharStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colMatchNorm)).
			Bold(true)

	matchCharSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colMatchSel)).
				Bold(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colPrompt)).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colInput))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colStatus)).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colError)).
			Bold(true)

	borderStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(colPaneBorder))
	paneTitleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(colPaneTitle)).Bold(true)
	keyLabelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(colKeyLabel)).Bold(true)
	keyDescStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(colKeyDesc))
	keySepStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(colKeySep))
	sectionHeadStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colSectionHead)).Bold(true)
)

// ── Help pane content ─────────────────────────────────────────────────────────

type helpEntry struct {
	key  string
	desc string
}

type helpSection struct {
	title   string
	entries []helpEntry
}

var helpSections = []helpSection{
	{
		title: "Navigation",
		entries: []helpEntry{
			{"↑ / ↓",     "move cursor"},
			{"Tab",       "enter directory"},
			{"Shift-Tab", "go up"},
			{"Backspace", "delete / go up"},
		},
	},
	{
		title: "Selection",
		entries: []helpEntry{
			{"Enter",  "select highlighted dir"},
			{"Ctrl-_", "select current dir"},
			{"Esc",    "quit"},
		},
	},
	{
		title: "Search",
		entries: []helpEntry{
			{"Type", "fuzzy filter"},
		},
	},
}

// visibleLen returns the printable character count of s, stripping ANSI escapes.
func visibleLen(s string) int {
	inEsc := false
	count := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if inEsc {
			if b == 'm' {
				inEsc = false
			}
			continue
		}
		if b == '\x1b' {
			inEsc = true
			continue
		}
		// Count leading byte of UTF-8 sequences only
		if b < 0x80 || b >= 0xC0 {
			count++
		}
	}
	return count
}

// padToWidth pads (or truncates) a rendered string to exactly w visible chars.
func padToWidth(s string, w int) string {
	vis := visibleLen(s)
	if vis < w {
		return s + strings.Repeat(" ", w-vis)
	}
	return s
}

// renderHelpPane returns the complete box as a slice of pre-rendered rows.
// totalW is the TOTAL width of the box including its border characters.
// totalH is the TOTAL height including the top and bottom border lines.
func renderHelpPane(totalW, totalH int) []string {
	if totalW < 8 || totalH < 4 {
		return nil
	}

	// Interior: subtract 2 for left/right border chars, 2 for inner padding spaces
	innerW := totalW - 4
	// Content rows: subtract 2 for top and bottom border lines
	contentH := totalH - 2

	// ── Build content lines ───────────────────────────────────────────────
	var content []string
	for _, sec := range helpSections {
		if len(content) > 0 {
			content = append(content, "")
		}
		content = append(content, sectionHeadStyle.Render(sec.title))
		for _, e := range sec.entries {
			line := keyLabelStyle.Render(e.key) +
				keySepStyle.Render("  ") +
				keyDescStyle.Render(e.desc)
			content = append(content, line)
		}
	}

	// Pad content to fill contentH
	for len(content) < contentH {
		content = append(content, "")
	}
	content = content[:contentH]

	// ── Assemble rows ─────────────────────────────────────────────────────
	title    := " keys "
	titleVis := len(title) // plain rune count (no ANSI in the raw string)
	// dashes = totalW - 2 (corners) - titleVis
	dashCount := totalW - 2 - titleVis
	if dashCount < 0 {
		dashCount = 0
	}

	topLine := borderStyle.Render("┌") +
		paneTitleStyle.Render(title) +
		borderStyle.Render(strings.Repeat("─", dashCount)+"┐")

	bottomLine := borderStyle.Render("└" + strings.Repeat("─", totalW-2) + "┘")

	rows := make([]string, 0, totalH)
	rows = append(rows, topLine)

	for _, c := range content {
		// Each content row: │ + space + content padded to innerW + space + │
		padded := padToWidth(c, innerW)
		rows = append(rows, borderStyle.Render("│")+" "+padded+" "+borderStyle.Render("│"))
	}

	rows = append(rows, bottomLine)
	return rows
}

// ── Layout constants ──────────────────────────────────────────────────────────

// helpPaneTotal is the total width of the right pane including its borders.
const helpPaneTotal = 36

// View implements tea.Model.
func (m model) View() string {
	// Fixed rows: 1 header + 1 status + 1 prompt
	fixedLines := 3
	if m.err != nil {
		fixedLines++
	}

	listHeight := m.height - fixedLines
	if listHeight < 1 {
		listHeight = 1
	}

	showHelp := m.width >= helpPaneTotal+24
	leftW := m.width
	if showHelp {
		leftW = m.width - helpPaneTotal - 1 // 1-char gutter
	}

	// ── Header (full width) ───────────────────────────────────────────────
	header := headerStyle.Width(m.width).Render("  " + m.currentDir)

	// ── Error line ────────────────────────────────────────────────────────
	errLine := ""
	if m.err != nil {
		errLine = errorStyle.Render(fmt.Sprintf("  error: %v", m.err))
	}

	// ── Left pane: directory list ─────────────────────────────────────────
	windowStart := 0
	if m.cursor >= listHeight {
		windowStart = m.cursor - listHeight + 1
	}
	windowEnd := windowStart + listHeight
	if windowEnd > len(m.filtered) {
		windowEnd = len(m.filtered)
	}

	leftLines := make([]string, 0, listHeight)
	if len(m.filtered) == 0 {
		if m.query == "" {
			leftLines = append(leftLines, normalStyle.Render("  (empty directory)"))
		} else {
			leftLines = append(leftLines, normalStyle.Render("  (no matches)"))
		}
	} else {
		for i := windowStart; i < windowEnd; i++ {
			leftLines = append(leftLines, renderDirEntry(m.filtered[i], i == m.cursor))
		}
	}
	for len(leftLines) < listHeight {
		leftLines = append(leftLines, "")
	}

	// ── Right pane: help box ──────────────────────────────────────────────
	// totalH = listHeight: the box spans the full list area (top + content + bottom)
	var helpLines []string
	if showHelp {
		helpLines = renderHelpPane(helpPaneTotal, listHeight)
		// Pad to listHeight in case renderHelpPane returned fewer lines
		for len(helpLines) < listHeight {
			helpLines = append(helpLines, "")
		}
	}

	// ── Compose output ────────────────────────────────────────────────────
	var sb strings.Builder

	sb.WriteString(header)
	sb.WriteRune('\n')

	if errLine != "" {
		sb.WriteString(errLine)
		sb.WriteRune('\n')
	}

	for i := 0; i < listHeight; i++ {
		// Left side padded to leftW
		left := padToWidth(leftLines[i], leftW)

		if showHelp {
			right := ""
			if i < len(helpLines) {
				right = helpLines[i]
			}
			sb.WriteString(left + " " + right)
		} else {
			sb.WriteString(left)
		}
		sb.WriteRune('\n')
	}

	// Status and prompt
	sb.WriteString(statusStyle.Render(fmt.Sprintf("  %d/%d", len(m.filtered), len(m.allDirs))))
	sb.WriteRune('\n')
	sb.WriteString(promptStyle.Render("  > ") + inputStyle.Render(m.query))

	return sb.String()
}

// renderDirEntry formats a single directory entry row with fuzzy highlights.
func renderDirEntry(entry filteredDir, isSelected bool) string {
	name := entry.name
	positions := entry.result.positions

	posSet := make(map[int]bool, len(positions))
	for _, p := range positions {
		posSet[p] = true
	}

	var rendered strings.Builder
	for i, ch := range name {
		s := string(ch)
		switch {
		case posSet[i] && isSelected:
			rendered.WriteString(matchCharSelectedStyle.Render(s))
		case posSet[i]:
			rendered.WriteString(matchCharStyle.Render(s))
		case isSelected:
			rendered.WriteString(selectedStyle.Render(s))
		default:
			rendered.WriteString(normalStyle.Render(s))
		}
	}

	if isSelected {
		return selectedStyle.Render(" ❯ ") + rendered.String()
	}
	return "   " + rendered.String()
}
