# yore shell integration for bash.  Add to ~/.bashrc:   eval "$(yore init bash)"
# Captures each command via a DEBUG trap and records it (with exit status) from
# PROMPT_COMMAND. For the most robust capture, consider bash-preexec.sh.
export YORE_SESSION=1   # lets yore detect that recording is enabled in this shell

_yore_debug() {
  # Ignore completion and the prompt command itself.
  [[ -n "$COMP_LINE" ]] && return
  [[ "$BASH_COMMAND" == "$PROMPT_COMMAND" ]] && return
  _YORE_CMD="$BASH_COMMAND"
}
trap '_yore_debug' DEBUG

_yore_record() {
  local exit=$?
  [[ -z "$_YORE_CMD" ]] && return
  command yore record --shell bash --exit "$exit" --dir "$PWD" \
    --command "$_YORE_CMD" >/dev/null 2>&1 &
  _YORE_CMD=
}
PROMPT_COMMAND="_yore_record${PROMPT_COMMAND:+; $PROMPT_COMMAND}"

# Press Ctrl-G to fuzzy-pick a command or recipe and drop it on the prompt.
_yore_pick() {
  local selected
  selected=$(yore pick) || return
  READLINE_LINE="$selected"
  READLINE_POINT=${#READLINE_LINE}
}
bind -x '"\C-g": _yore_pick' 2>/dev/null
