# Demo for yore — render with:  ZDOTDIR=$(mktemp -d) vhs docs/demo.tape
# Requires: vhs + fzf + yore on PATH. Renders against a throwaway store + fake
# project so the GIF shows curated commands, not your real history.

Output docs/demo.gif
Require yore

Set Shell zsh
Set FontSize 22
Set Width 1200
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms

# --- off-camera setup: isolated store, clean prompt, a fake "api" project with
#     seeded commands (no recording hook, so recall shows only this data). ---
Hide
Type "export XDG_DATA_HOME=$(mktemp -d) XDG_CONFIG_HOME=$(mktemp -d) HISTFILE=$(mktemp)" Enter
Type 'PROMPT="%F{6}~/api%f %F{2}❯%f "' Enter
Type "RPROMPT=''" Enter
Type "cd $(mktemp -d) && git init -q" Enter
Type "for i in 1 2 3; do yore record --command 'make test' --dir $PWD --exit 0; done" Enter
Type "yore record --command 'docker compose up -d' --dir $PWD --exit 0" Enter
Type "yore record --command 'kubectl -n staging logs -f deploy/api' --dir $PWD --exit 0" Enter
Type "yore record --command './deploy.sh --prod' --dir $PWD --exit 1" Enter
Type "clear" Enter
Show

# commands the project actually uses, ranked
Sleep 700ms
Type "yore here" Sleep 400ms Enter Sleep 2200ms

# ...only the ones that succeeded (the failed deploy drops off)
Type "yore here --ok" Sleep 400ms Enter Sleep 2200ms

# save a reusable recipe with a fill-in-the-blank
Type "yore save deploy -- ./deploy.sh --env '{env}'" Sleep 400ms Enter Sleep 1800ms

# run it: yore prompts for {env} and resolves the command
Type "yore run deploy --print" Sleep 300ms Enter Sleep 900ms
Type "production" Enter Sleep 2200ms

# Ctrl-G fuzzy picker (bound inline, keybinding only — no recording)
Hide
Type "yore-pick-widget() { local s; s=$(yore pick) || { zle reset-prompt; return }; [[ -n $s ]] && LBUFFER=$s; zle reset-prompt }" Enter
Type "zle -N yore-pick-widget" Enter
Type "bindkey '^G' yore-pick-widget" Enter
Type "clear" Enter
Show
Sleep 500ms
Ctrl+G Sleep 1500ms
Type "kube" Sleep 1600ms
Enter Sleep 2000ms
Ctrl+C Sleep 800ms
