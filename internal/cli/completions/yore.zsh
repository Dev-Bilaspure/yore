#compdef yore
# zsh completion for yore
_yore() {
    local -a subcommands
    subcommands=(
        'here:recall commands used in the current project'
        'pick:fuzzy-pick a command or recipe'
        'save:save a command as a reusable recipe'
        'run:run a saved recipe'
        'recipes:list saved recipes'
        'init:print shell integration'
        'import:seed memory from existing history'
        'completion:output a shell completion script'
    )

    _arguments -C \
        '(-n --top)'{-n,--top}'[limit to top N commands]:N:' \
        '--here[scope to the current project]' \
        '--ok[only commands that succeeded]' \
        '--sort[output ordering]:mode:(frecency recent frequent raw)' \
        '--half-life[frecency decay half-life]:duration:' \
        '--min-count[only commands used at least N times]:N:' \
        '--count[show usage counts]' \
        '--redact[drop commands that look like secrets]' \
        '*--file[read a specific history file (- for stdin)]:file:_files' \
        '--shell[with --file: which shell]:shell:(auto zsh bash all)' \
        '-0[NUL-separated output]' \
        '--null[NUL-separated output]' \
        '--version[print version]' \
        '1: :->cmds' \
        '*:: :->args'

    case $state in
        cmds) _describe 'subcommand' subcommands ;;
        args)
            case $words[1] in
                init|completion) _values 'shell' bash zsh fish ;;
                run) compadd $(yore recipes 2>/dev/null | awk '{print $1}') ;;
            esac ;;
    esac
}

# Call _yore only when this file is sourced by the completion system (as the
# autoloaded _yore function), not when sourced directly via eval.
if [ "$funcstack[1]" = "_yore" ]; then
    _yore "$@"
fi
