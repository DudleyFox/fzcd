# ── fzcd shell integration ────────────────────────────────────────────────────
#
# fzcd prints the selected path to stdout. Since a child process cannot change
# the parent shell's working directory, you need a small shell wrapper function
# that calls fzcd and then cd's into whatever it returns.
#
# Add the relevant snippet to your shell's rc file.
# ─────────────────────────────────────────────────────────────────────────────

# ── Bash / Zsh ────────────────────────────────────────────────────────────────
#
# Add to ~/.bashrc or ~/.zshrc:
#
#   source /path/to/fzcd.sh
#
# Then invoke with:   fzcd          (starts in current directory)
#                     fzcd ~/code   (starts in ~/code)
#
fzcd() {
  local target
  target="$(command fzcd "$@")"
  if [ -n "$target" ]; then
    cd "$target" || return 1
  fi
}


# ── Fish ──────────────────────────────────────────────────────────────────────
#
# Save the following as ~/.config/fish/functions/fzcd.fish:
#
# function fzcd
#     set target (command fzcd $argv)
#     if test -n "$target"
#         cd $target
#     end
# end
