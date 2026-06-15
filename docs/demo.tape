# Demo for yore — render with:  ZDOTDIR=$(mktemp -d) vhs docs/demo.tape
# Requires: vhs + fzf + yore on PATH. The clean ZDOTDIR keeps your prompt theme
# out of the recording; it renders against a throwaway store + fake project so
# the GIF shows curated commands, not your real history.

Output docs/demo.gif
Require yore

Set Shell zsh
Set FontSize 22
Set Width 1200
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

# --- off-camera setup: isolated store, clean prompt, fake project, seeded
#     commands. No recording hook is enabled, so the recall demos below show
#     only this curated data. ---
Hide
Type "export XDG_DATA_HOME=$(mktemp -d) XDG_CONFIG_HOME=$(mktemp -d) HISTFILE=$(mktemp)" Enter
Type "setopt interactive_comments" Enter
Type 'PROMPT="%F{6}~/demo%f %F{2}❯%f "' Enter
Type "RPROMPT=''" Enter
Type "cd $(mktemp -d) && git init -q" Enter
Type "for i in 1 2 3; do yore record --command 'make test' --dir $PWD --exit 0; done" Enter
Type "yore record --command 'docker compose up -d' --dir $PWD --exit 0" Enter
Type "yore record --command 'kubectl -n staging logs -f deploy/api' --dir $PWD --exit 0" Enter
Type "yore record --command './deploy.sh --prod' --dir $PWD --exit 1" Enter
Type "clear" Enter
Show

# 1. project-aware recall
Type "# what actually gets run in THIS project:" Enter Sleep 400ms
Type "yore here" Enter Sleep 1800ms

# 2. only the commands that worked
Type "# ...only the ones that exited cleanly:" Enter Sleep 300ms
Type "yore here --ok" Enter Sleep 1800ms

# 3. save the keeper as a team recipe (committed to .yorefile)
Type "# save it as a team recipe -> ./.yorefile (commit it):" Enter Sleep 300ms
Type "yore save --project deploy -- ./deploy.sh --env '{env}'" Enter Sleep 1500ms

# 4. run it: yore prompts for the blank, then resolves the command
Type "# run it - yore fills in the blank:" Enter Sleep 300ms
Type "yore run deploy --print" Enter Sleep 900ms
Type "production" Enter Sleep 1800ms

# 5. Ctrl-G fuzzy picker. Bind it inline (keybinding only, no recording) so the
#    picker stays clean and nothing extra is shown.
Type "# or hit Ctrl-G to fuzzy-pick onto your prompt:" Enter Sleep 1000ms
Hide
Type "yore-pick-widget() { local s; s=$(yore pick) || { zle reset-prompt; return }; [[ -n $s ]] && LBUFFER=$s; zle reset-prompt }" Enter
Type "zle -N yore-pick-widget" Enter
Type "bindkey '^G' yore-pick-widget" Enter
Type "clear" Enter
Show
Ctrl+G Sleep 1500ms
Type "kube" Sleep 1500ms
Enter Sleep 1800ms
Ctrl+C Sleep 1000ms
