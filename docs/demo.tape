# Demo for yore — render with:  vhs docs/demo.tape   (produces docs/demo.gif)
# Requires: vhs (brew install vhs) and yore on PATH.
#
# It builds a throwaway demo environment (its own temp store + a fake project)
# so the recording shows curated commands, not your real history.

Output docs/demo.gif
Require yore

Set Shell zsh
Set FontSize 22
Set Width 1200
Set Height 720
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 70ms

# --- off-camera setup: isolated store + a fake project with seeded commands ---
Hide
Type "export XDG_DATA_HOME=$(mktemp -d) XDG_CONFIG_HOME=$(mktemp -d) HISTFILE=$(mktemp)" Enter
Type "cd $(mktemp -d) && git init -q" Enter
Type "for i in 1 2 3; do yore record --command 'make test' --dir $PWD --exit 0; done" Enter
Type "yore record --command 'docker compose up -d' --dir $PWD --exit 0" Enter
Type "yore record --command 'kubectl -n staging logs -f deploy/api' --dir $PWD --exit 0" Enter
Type "yore record --command './deploy.sh --prod' --dir $PWD --exit 1" Enter
Type 'eval "$(yore init zsh)"' Enter
Type "clear" Enter
Show

# --- 1. project-aware recall ---
Type "# what actually gets run in THIS project:" Sleep 800ms Enter
Type "yore here" Sleep 800ms Enter Sleep 2500ms

# --- 2. only what worked (exit 0) ---
Type "yore here --ok" Sleep 800ms Enter Sleep 2500ms

# --- 3. save a team recipe with a fill-in-the-blank ---
Type "yore save --project deploy -- ./deploy.sh --env '{env}'" Sleep 600ms Enter Sleep 1800ms

# --- 4. run it: it prompts for the blank, then resolves the command ---
Type "yore run deploy --print" Enter Sleep 1200ms
Type "production" Enter Sleep 2500ms

# --- 5. Ctrl-G: fuzzy-pick onto the prompt ---
Type "# press Ctrl-G to fuzzy-pick a command or recipe:" Sleep 800ms Enter
Ctrl+G Sleep 1500ms
Type "kube" Sleep 1800ms
Enter Sleep 2200ms
Ctrl+C Sleep 1500ms
