# yore shell integration for zsh.  Add to ~/.zshrc:   eval "$(yore init zsh)"
zmodload zsh/datetime 2>/dev/null

typeset -g _YORE_CMD _YORE_START

_yore_preexec() {
  _YORE_CMD=$1
  _YORE_START=$EPOCHREALTIME
}

_yore_precmd() {
  local exit=$?
  [[ -z $_YORE_CMD ]] && return
  local ms=0
  [[ -n $_YORE_START ]] && ms=$(( (EPOCHREALTIME - _YORE_START) * 1000 ))
  command yore record --shell zsh --exit $exit --dir "$PWD" \
    --duration-ms ${ms%.*} --command "$_YORE_CMD" &>/dev/null &!
  _YORE_CMD=
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec _yore_preexec
add-zsh-hook precmd  _yore_precmd

# Press Ctrl-G to fuzzy-pick a command or recipe and drop it on the prompt.
yore-pick-widget() {
  local selected
  selected=$(yore pick) || { zle reset-prompt; return }
  [[ -n $selected ]] && LBUFFER=$selected
  zle reset-prompt
}
zle -N yore-pick-widget
bindkey '^G' yore-pick-widget
