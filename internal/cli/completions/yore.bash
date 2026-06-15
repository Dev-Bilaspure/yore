# bash completion for yore
_yore() {
    local cur prev words cword
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    local subcommands="here pick save run recipes init import completion"
    local flags="-n --top --here --ok --sort --half-life --min-count --count \
--redact --file --shell --no-dedup -0 --null --version -h --help"

    case "$prev" in
        --sort)  COMPREPLY=( $(compgen -W "frecency recent frequent raw" -- "$cur") ); return ;;
        --shell) COMPREPLY=( $(compgen -W "auto zsh bash all" -- "$cur") ); return ;;
        --file)  COMPREPLY=( $(compgen -f -- "$cur") ); return ;;
        init|completion) COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") ); return ;;
        run|recipes)
            COMPREPLY=( $(compgen -W "$(yore recipes 2>/dev/null | awk '{print $1}')" -- "$cur") )
            return ;;
        -n|--top|--min-count|--half-life) return ;;
    esac

    if [[ "$cur" == -* ]]; then
        COMPREPLY=( $(compgen -W "$flags" -- "$cur") )
        return
    fi
    if [[ "$COMP_CWORD" -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "$subcommands" -- "$cur") )
    fi
}
complete -F _yore yore
