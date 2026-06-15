# fish completion for yore
set -l subcommands here pick save run recipes init import completion

# Subcommands (only as the first argument).
complete -c yore -n '__fish_use_subcommand' -a here       -d 'recall commands in this project'
complete -c yore -n '__fish_use_subcommand' -a pick       -d 'fuzzy-pick a command or recipe'
complete -c yore -n '__fish_use_subcommand' -a save       -d 'save a reusable recipe'
complete -c yore -n '__fish_use_subcommand' -a run        -d 'run a saved recipe'
complete -c yore -n '__fish_use_subcommand' -a recipes    -d 'list saved recipes'
complete -c yore -n '__fish_use_subcommand' -a init       -d 'print shell integration'
complete -c yore -n '__fish_use_subcommand' -a import     -d 'seed from existing history'
complete -c yore -n '__fish_use_subcommand' -a completion -d 'output a completion script'

# Argument values for specific subcommands.
complete -c yore -n '__fish_seen_subcommand_from init completion' -xa 'bash zsh fish'
complete -c yore -n '__fish_seen_subcommand_from run' \
    -xa '(yore recipes 2>/dev/null | string split -f1 " ")'

# Flags.
complete -c yore -s n -l top       -d 'limit to top N commands' -r
complete -c yore       -l here      -d 'scope to the current project'
complete -c yore       -l ok        -d 'only commands that succeeded'
complete -c yore       -l sort      -d 'output ordering' -xa 'frecency recent frequent raw'
complete -c yore       -l half-life -d 'frecency decay half-life' -r
complete -c yore       -l min-count -d 'only commands used at least N times' -r
complete -c yore       -l count     -d 'show usage counts'
complete -c yore       -l redact    -d 'drop commands that look like secrets'
complete -c yore       -l file      -d 'read a specific history file (- for stdin)' -r -F
complete -c yore       -l shell     -d 'with --file: which shell' -xa 'auto zsh bash all'
complete -c yore -s 0  -l null      -d 'NUL-separated output'
complete -c yore       -l version   -d 'print version'
complete -c yore -s h  -l help      -d 'show help'
