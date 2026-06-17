# yore shell integration for fish.  Add to ~/.config/fish/config.fish:
#   yore init fish | source
set -gx YORE_SESSION 1   # lets yore detect that recording is enabled in this shell

function _yore_record --on-event fish_postexec
    set -l code $status
    test -z "$argv[1]"; and return
    command yore record --shell fish --exit $code --dir "$PWD" \
        --duration-ms $CMD_DURATION --command "$argv[1]" >/dev/null 2>&1 &
    disown
end

# Press Ctrl-G to fuzzy-pick a command or recipe and drop it on the prompt.
function _yore_pick
    set -l selected (yore pick)
    if test -n "$selected"
        commandline -r -- $selected
    end
    commandline -f repaint
end
bind \cg _yore_pick
