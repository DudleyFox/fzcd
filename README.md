# fzcd — fuzzy cd

A fast, fzf-style interactive directory navigator written in Go.  
Browse subdirectories with fuzzy search, then `cd` straight into your target.

```
fzcd [path]
```

If `path` is omitted, starts in the current directory.

---

## Key bindings

| Key | Action |
|-----|--------|
| **Type anything** | Fuzzy-filter subdirectories |
| **↑ / ↓** | Move cursor up / down |
| **Tab** | Enter the highlighted directory |
| **Shift-Tab** | Go up to parent directory |
| **Backspace** | Delete last search character; if empty, go up |
| **Enter** | Select highlighted directory — prints its full path to stdout |
| **Ctrl-/** | Select the *current* directory (not the highlighted one) |
| **Esc / Ctrl-C** | Quit without selecting |

---

## UI layout

```
  /home/user/projects          ← current directory (header)
  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄
❯ fzcd                         ← highlighted match (fuzzy chars in orange)
  fzf
  my-app
  scripts
  ┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄┄
  4/12                          ← matched / total
  > fz_                         ← search prompt
  tab:enter  shift-tab:up  ...  ← help
```

---

## Installation

### Build from source

```bash
git clone https://github.com/yourname/fzcd
cd fzcd
go build -o fzcd .
sudo mv fzcd /usr/local/bin/
```

### Shell integration (required for `cd`)

Because a subprocess cannot change the parent shell's directory, you need a
small wrapper function. Add this to your `~/.bashrc` or `~/.zshrc`:

```bash
source /path/to/fzcd.sh
```

Or paste the wrapper directly:

```bash
fzcd() {
  local target
  target="$(command fzcd "$@")"
  if [ -n "$target" ]; then
    cd "$target" || return 1
  fi
}
```

For **Fish**, save to `~/.config/fish/functions/fzcd.fish`:

```fish
function fzcd
    set target (command fzcd $argv)
    if test -n "$target"
        cd $target
    end
end
```

---

## tmux-sessionizer integration

[tmux-sessionizer](https://github.com/ThePrimeagen/tmux-sessionizer) uses `fzf` to pick a project directory and open (or switch to) a tmux session for it. You can swap that `fzf` call for `fzcd` to get an interactive tree navigator instead of a flat fuzzy list.

The relevant line in tmux-sessionizer is:

```bash
selected=$(find_dirs | fzf)
```

Replace it with:

```bash
selected=$(fzcd ~)
```

`fzcd` prints the chosen path to stdout just like `fzf` does, so the rest of the script (`selected_name`, `tmux new-session`, `switch_to`, etc.) works unchanged.

### Full diff

```diff
-selected=$(find_dirs | fzf)
+selected=$(fzcd ~)
```

> **Note:** Because `fzcd` renders its TUI on stderr and emits the chosen path on stdout, it is safe to capture with `$()` — the same guarantee that makes the original `fzf` approach work.

---

## How it works

- Written in Go using [Bubbletea](https://github.com/charmbracelet/bubbletea) for the TUI
  and [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling.
- Fuzzy matching is a custom implementation inspired by fzf's scoring algorithm:
  consecutive character bonuses, word-boundary bonuses, prefix bonuses, and gap penalties.
- The selected path is printed to **stdout**; the TUI itself renders on **stderr**,
  so the output is safe to capture with `$()`.

---

## License

MIT
