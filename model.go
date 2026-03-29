package main

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// filteredDir is a subdirectory entry with its fuzzy match result.
type filteredDir struct {
	name   string
	result matchResult
}

// model is the bubbletea application state.
type model struct {
	// Navigation state
	currentDir string   // absolute path of the directory we're browsing
	allDirs    []string // all immediate subdirs in currentDir
	query      string   // current search string typed by user

	// Filtered & sorted results
	filtered []filteredDir

	// UI state
	cursor   int    // index into filtered list
	selected string // the path returned on Enter (empty = user quit)
	err      error  // last filesystem error (shown briefly)

	// Terminal dimensions
	width  int
	height int
}

func newModel(startDir string) model {
	m := model{
		currentDir: startDir,
		width:      80,
		height:     24,
	}
	m.loadDirs()
	return m
}

// loadDirs reads the current directory and rebuilds the filtered list.
func (m *model) loadDirs() {
	dirs, err := listDirs(m.currentDir)
	if err != nil {
		m.err = err
		m.allDirs = nil
	} else {
		m.err = nil
		m.allDirs = dirs
	}
	m.rebuildFilter()
}

// rebuildFilter applies the current query to allDirs and sorts results.
func (m *model) rebuildFilter() {
	m.filtered = make([]filteredDir, 0, len(m.allDirs))

	for _, name := range m.allDirs {
		result := fuzzyMatch(name, m.query)
		if result.matched {
			m.filtered = append(m.filtered, filteredDir{name: name, result: result})
		}
	}

	if m.query != "" {
		// Sort by score descending, then alphabetically for ties
		sort.SliceStable(m.filtered, func(i, j int) bool {
			if m.filtered[i].result.score != m.filtered[j].result.score {
				return m.filtered[i].result.score > m.filtered[j].result.score
			}
			return m.filtered[i].name < m.filtered[j].name
		})
	}

	// Clamp cursor
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {

		// Quit without selecting
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		// Move up in list
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}

		// Move down in list
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}

		// Tab: enter the highlighted directory (navigate into it)
		case tea.KeyTab:
			if len(m.filtered) > 0 {
				m.navigateInto(m.filtered[m.cursor].name)
			}

		// Shift-Tab: go up a directory
		case tea.KeyShiftTab:
			m.navigateUp()

		// Enter: select highlighted directory and return its full path
		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				m.selected = joinPath(m.currentDir, m.filtered[m.cursor].name)
			}
			return m, tea.Quit

		// Ctrl-/: select the current directory itself (not a subdir).
		// Terminals typically send Ctrl-/ as ctrl+underscore (0x1F).
		case tea.KeyCtrlUnderscore:
			m.selected = m.currentDir
			return m, tea.Quit

		// Backspace: delete last query char, or go up if query is empty
		case tea.KeyBackspace:
			if len(m.query) > 0 {
				// Remove last UTF-8 character from query
				runes := []rune(m.query)
				m.query = string(runes[:len(runes)-1])
				m.cursor = 0
				m.rebuildFilter()
			} else {
				m.navigateUp()
			}

		// Any printable character: append to query
		default:
			if msg.Type == tea.KeyRunes {
				m.query += string(msg.Runes)
				m.cursor = 0
				m.rebuildFilter()
			}
		}
	}

	return m, nil
}

// navigateInto descends into a named subdirectory and clears the query.
func (m *model) navigateInto(name string) {
	newPath := joinPath(m.currentDir, name)
	m.currentDir = newPath
	m.query = ""
	m.cursor = 0
	m.loadDirs()
}

// navigateUp goes to the parent directory and clears the query.
func (m *model) navigateUp() {
	parent := parentDir(m.currentDir)
	if parent == m.currentDir {
		return // already at root
	}
	m.currentDir = parent
	m.query = ""
	m.cursor = 0
	m.loadDirs()
}

// joinPath safely joins a base dir and a relative name.
func joinPath(base, name string) string {
	if strings.HasSuffix(base, "/") {
		return base + name
	}
	return base + "/" + name
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
