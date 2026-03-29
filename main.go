package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const helpText = `fzcd — fuzzy interactive directory navigator

USAGE
  fzcd [options] [path]

  If path is omitted, starts in the current directory.
  Prints the selected directory to stdout; use the shell wrapper to cd into it.

OPTIONS
  --help          Show this help message and exit
  --tmux=SPEC     Open in a tmux popup (default when inside tmux).
                  SPEC format: [center|top|bottom|left|right][,WIDTH%][,HEIGHT%]
                  Examples:
                    --tmux=center,63%,63%   centered, 63 wide x 63 tall  (default)
                    --tmux=right,40%        right side, 40% wide
                    --tmux=bottom,80%,30%   bottom, 80% wide x 30% tall
  --no-tmux       Run inline even when inside a tmux session

KEY BINDINGS
  Type              Fuzzy-filter subdirectories
  Up / Down         Move cursor up / down
  Tab               Enter highlighted directory
  Shift-Tab         Go up to parent directory
  Backspace         Delete last search char; if empty, go up a directory
  Enter             Select highlighted directory -- prints full path to stdout
  Ctrl-_            Select the current directory itself (not the highlighted one)
  Esc / Ctrl-C      Quit without selecting

SHELL INTEGRATION
  Because a subprocess cannot change the parent shell's directory you need a
  small wrapper. Add this to ~/.bashrc or ~/.zshrc:

    fzcd() {
      local target
      target="$(command fzcd "$@")"
      [ -n "$target" ] && cd "$target"
    }

  For Fish, save to ~/.config/fish/functions/fzcd.fish:

    function fzcd
      set target (command fzcd $argv)
      test -n "$target" && cd $target
    end
`

// tmuxSpec holds the parsed --tmux popup configuration.
type tmuxSpec struct {
	position string // center | top | bottom | left | right
	width    string // e.g. "52%"
	height   string // e.g. "52%"
}

func (t tmuxSpec) popupArgs() []string {
	var x, y string
	switch t.position {
	case "top":
		x, y = "C", "0"
	case "bottom":
		x, y = "C", "100%"
	case "left":
		x, y = "0", "C"
	case "right":
		x, y = "100%", "C"
	default: // center
		x, y = "C", "C"
	}
	return []string{
		"display-popup",
		"-E",
		"-x", x,
		"-y", y,
		"-w", t.width,
		"-h", t.height,
	}
}

func defaultTmuxSpec() tmuxSpec {
	return tmuxSpec{position: "center", width: "63%", height: "63%"}
}

// parseTmuxSpec parses values like "center,52%,52%".
// Accepts 0-3 comma-separated parts: [position][,width][,height]
func parseTmuxSpec(val string) (tmuxSpec, error) {
	spec := defaultTmuxSpec()
	if val == "" || val == "true" {
		return spec, nil
	}

	parts := strings.Split(val, ",")
	validPositions := map[string]bool{
		"center": true, "top": true, "bottom": true, "left": true, "right": true,
	}

	idx := 0
	if idx < len(parts) {
		p := strings.TrimSpace(parts[idx])
		if validPositions[p] {
			spec.position = p
			idx++
		}
	}
	if idx < len(parts) {
		spec.width = strings.TrimSpace(parts[idx])
		idx++
	}
	if idx < len(parts) {
		spec.height = strings.TrimSpace(parts[idx])
		idx++
	}
	if idx < len(parts) {
		return spec, fmt.Errorf("unexpected extra values in --tmux spec: %s", val)
	}
	return spec, nil
}

// insideTmux returns true when running inside a tmux session.
func insideTmux() bool {
	return os.Getenv("TMUX") != ""
}

func main() {
	startPath := "."
	useTmux := false
	noTmux := false
	spec := defaultTmuxSpec()
	tmuxSpecSet := false

	// ── Parse arguments ──────────────────────────────────────────────────
	for _, arg := range os.Args[1:] {
		switch {
		case arg == "--help" || arg == "-h":
			fmt.Print(helpText)
			os.Exit(0)

		case arg == "--no-tmux":
			noTmux = true

		case arg == "--tmux":
			useTmux = true
			tmuxSpecSet = true

		case strings.HasPrefix(arg, "--tmux="):
			val := strings.TrimPrefix(arg, "--tmux=")
			parsed, err := parseTmuxSpec(val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "fzcd: %v\n", err)
				os.Exit(1)
			}
			spec = parsed
			useTmux = true
			tmuxSpecSet = true

		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(os.Stderr, "fzcd: unknown flag: %s\n", arg)
			fmt.Fprintf(os.Stderr, "Run 'fzcd --help' for usage.\n")
			os.Exit(1)

		default:
			startPath = arg
		}
	}

	// Auto-enable tmux popup when inside tmux (unless suppressed)
	if insideTmux() && !noTmux && !tmuxSpecSet {
		useTmux = true
	}

	// ── Expand and validate start path ───────────────────────────────────
	if len(startPath) > 0 && startPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			startPath = home + startPath[1:]
		}
	}

	absPath, err := resolveAbsPath(startPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fzcd: invalid path: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "fzcd: not a directory: %s\n", absPath)
		os.Exit(1)
	}

	// ── tmux popup re-exec ────────────────────────────────────────────────
	// Re-launch ourselves inside a tmux display-popup. A temp file is used
	// to pass the selected path back from the inner process to the outer one.
	if useTmux && os.Getenv("FZCD_INNER") == "" {
		tmpFile, err := os.CreateTemp("", "fzcd-result-*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "fzcd: could not create temp file: %v\n", err)
			os.Exit(1)
		}
		tmpFile.Close()
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		self, err := os.Executable()
		if err != nil {
			self = os.Args[0]
		}

		innerCmd := fmt.Sprintf("FZCD_INNER=1 FZCD_RESULT=%s %s --no-tmux %s",
			shellQuote(tmpPath), shellQuote(self), shellQuote(absPath))

		args := append(spec.popupArgs(), "sh", "-c", innerCmd)
		cmd := exec.Command("tmux", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		cmd.Run()

		result, err := os.ReadFile(tmpPath)
		if err == nil {
			out := strings.TrimSpace(string(result))
			if out != "" {
				fmt.Println(out)
			}
		}
		return
	}

	// ── Normal (non-popup) run ────────────────────────────────────────────
	m := newModel(absPath)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fzcd: %v\n", err)
		os.Exit(1)
	}

	if fm, ok := result.(model); ok && fm.selected != "" {
		if resultPath := os.Getenv("FZCD_RESULT"); resultPath != "" {
			os.WriteFile(resultPath, []byte(fm.selected+"\n"), 0600)
		} else {
			fmt.Println(fm.selected)
		}
	}
}

// shellQuote wraps a string in single quotes for safe shell interpolation.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
