# yore

**Your shell history as project-aware, frecency-ranked, team-shareable memory.**

Your shell records everything you type but understands none of it. The 40-character
`kubectl`/`docker`/`gcloud` incantation you figured out three months ago is now
buried in flat history, a stale README, or your teammate's head — so you re-derive
it. Again.

`yore` fixes that. It ranks the commands you actually reuse by **frecency**
(frequency + recency), scopes them to the **project you're in**, knows which ones
**actually worked**, and lets you save the keepers as **named, parameterized
recipes you can commit and share with your team**.

```console
$ yore here                 # what actually gets run in THIS repo
make test
docker compose up -d
kubectl -n staging logs -f deploy/api

$ yore run deploy           # a saved recipe, prompts for the variable parts
env [staging]: prod
tag: v2.1.0
→ ./deploy.sh --env prod --tag v2.1.0
```

No daemon. No database. No `Ctrl-R` hijack. Plain-text, git-friendly storage, and
a single dependency-free binary.

---

## Why not just `atuin` / `fzf` / `navi`?

| | what it does | the gap yore fills |
| --- | --- | --- |
| `fzf` + Ctrl-R | fuzzy-search raw history | no ranking, no context, no recipes |
| `atuin` / `mcfly` | recorded history search (SQLite, sync) | personal-only, binary store, replaces Ctrl-R |
| `just` / `make` | named project commands | hand-authored, not learned from real use |
| `navi` / `tldr` | cheatsheets | manually maintained |

**Nobody else does: auto-learned from real usage + scoped to this project +
version-controlled so your team shares it.** That's yore.

## Install

```sh
# Homebrew (macOS / Linux) — installs completions + man page
brew install indihood/tap/yore

# Go
go install github.com/indihood/yore/cmd/yore@latest

# Or grab a prebuilt binary from the Releases page.
```

Then turn on recording (one line in your shell rc):

```sh
eval "$(yore init zsh)"     # ~/.zshrc       (also: bash, fish)
```

That installs a lightweight hook that records each command's directory, exit
status, and duration — and binds **Ctrl-G** to the interactive picker. Seed it
with your existing history once:

```sh
yore import
```

## Use it

### Recall

```sh
yore                  # every command, best-first (frecency)
yore -n 20            # top 20
yore here             # commands used in the current project
yore here --ok        # ...only ones that exited successfully
yore --sort recent    # most recent first  (also: frequent, raw)
yore --count          # show how often each was used
yore | fzf            # still a clean, composable pipe
```

### Interactive pick (Ctrl-G)

After `yore init`, press **Ctrl-G** to fuzzy-pick a command or recipe and drop it
straight onto your prompt. It searches your recipes *and* your frecency-ranked
history at once (powered by `fzf` if installed, with a numbered fallback if not).

### Recipes — save the keepers, share with your team

```sh
# Save a command (write {placeholders} for the parts that vary):
yore save deploy -- ./deploy.sh --env '{env}' --tag '{tag}'
yore save --last "fix the flaky test"      # capture the command you just ran

# Save to the project so your team gets it (writes ./.yorefile — commit it):
yore save --project test -- make test

yore recipes          # list them
yore run deploy       # prompts for {env}/{tag}, then runs it
yore run deploy --print   # just print the resolved command
```

A `.yorefile` is plain text and meant to be committed — it's your repo's runbook,
learned from what the team actually runs:

```ini
[test]
desc: run the test suite
cmd: make test

[deploy]
desc: build and ship to a cluster
cmd: ./deploy.sh --env {env} --tag {tag}
param: env=staging
```

A new teammate clones the repo, runs `yore recipes`, and instantly knows how to
build, test, and deploy it.

> **Tip:** quote `'{placeholders}'` when saving from the shell so it doesn't
> expand the braces.

## How it works

- **Recording** (`yore init`) appends each command — with `cwd`, exit code, and
  duration — to a plain JSON-Lines log at `~/.local/share/yore/history.jsonl`.
- **Ranking** is exponential-decay frecency: `score = count × 2^(−age / half-life)`
  (default half-life 30 days, tune with `--half-life`). A command used a lot last
  year loses to one used a few times this week.
- **Project scope** (`yore here`) keeps only commands run in the current project
  (the nearest ancestor with a `.git`, `.yorefile`, `go.mod`, …).
- **Recipes** live in `~/.config/yore/recipes` (personal) and `./.yorefile`
  (per-project; project recipes override personal ones).

## Privacy

yore is local-only: **no telemetry, no network, no sync.** Commands that look
like they contain secrets (passwords, tokens, API keys) are **never written to
the store**, and `--redact` also strips them from output. Your history file is
yours; `.yorefile` recipes are the only thing meant to be shared, and only when
you commit them.

## Commands

```text
yore [flags]          recall commands, best-first
yore here [flags]     recall commands used in the current project
yore pick             fuzzy-pick a command/recipe (Ctrl-G after init)
yore save <name> -- <cmd>   save a reusable recipe (--project, --last)
yore run <name>       run a recipe (prompts for {params}; --print, --yes)
yore recipes          list saved recipes
yore init <shell>     print shell integration (eval it)
yore import           seed memory from existing shell history
yore completion <sh>  shell completion script
```

Key flags: `-n/--top`, `--here`, `--ok`, `--sort`, `--half-life`, `--min-count`,
`--count`, `--redact`, `--file` (read a specific history file / `-` for stdin),
`-0/--null`. Run `yore --help` for the full list.

## Library

The engine is a small, dependency-free Go API:

```go
import (
    "github.com/indihood/yore/pkg/store"
    "github.com/indihood/yore/pkg/history"
)

st, _ := store.Default()
events, _ := st.Load()
entries := history.RankEvents(events, history.EventQuery{
    Sort: history.SortFrecency, Dir: "/path/to/project", Limit: 20,
})
```

## Roadmap

- Optional encrypted sync for recipes across machines.
- A built-in TUI (today the interactive picker uses `fzf`).
- Richer recipe params (choices, validation).

## Releasing (maintainers)

Automated with [goreleaser](https://goreleaser.com):

1. Create the tap repo `indihood/homebrew-tap` and a token in
   `HOMEBREW_TAP_GITHUB_TOKEN` (+ `GITHUB_TOKEN` for the release).
2. `goreleaser check` → `git tag v0.1.0 && goreleaser release --clean`.

This builds binaries for linux/darwin/windows × amd64/arm64 and publishes a
Homebrew cask so users can `brew install indihood/tap/yore`.

## Contributing

Issues and PRs welcome. Run `make check` (build + vet + lint + race tests) before
submitting. New shell parsers implement `history.Source`; new shells for
recording add an `internal/cli/shellinit/*` script.

## License

[MIT](LICENSE) © Dev Bilaspure
