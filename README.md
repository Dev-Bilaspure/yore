# yore

**Your shell history as project-aware, frecency-ranked, team-shareable memory.**

[![CI](https://github.com/Dev-Bilaspure/yore/actions/workflows/ci.yml/badge.svg)](https://github.com/Dev-Bilaspure/yore/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Dev-Bilaspure/yore?sort=semver)](https://github.com/Dev-Bilaspure/yore/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/Dev-Bilaspure/yore.svg)](https://pkg.go.dev/github.com/Dev-Bilaspure/yore)
[![Go Report Card](https://goreportcard.com/badge/github.com/Dev-Bilaspure/yore)](https://goreportcard.com/report/github.com/Dev-Bilaspure/yore)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

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

## Contents

- [Why not just atuin / fzf / navi?](#why-not-just-atuin--fzf--navi)
- [Install](#install)
- [Use it](#use-it) · [Recall](#recall) · [Interactive pick](#interactive-pick-ctrl-g) · [Recipes](#recipes--save-the-keepers-share-with-your-team)
- [How it works](#how-it-works)
- [Privacy](#privacy)
- [Commands](#commands)
- [Library](#library)
- [Roadmap](#roadmap) · [Contributing](#contributing) · [License](#license)

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

### 1. Install the binary

**Homebrew** (macOS / Linux) — also installs completions and the man page:

```sh
brew tap Dev-Bilaspure/tap
brew trust dev-bilaspure/tap   # one-time: Homebrew requires trusting third-party taps
brew install yore
```

> The `brew trust` step is a standard, one-time-per-machine confirmation that
> Homebrew now requires for any tap outside its official catalog. After it,
> upgrades (`brew upgrade yore`) and any future tools from this tap need no
> further trust.

Or with **Go**: `go install github.com/Dev-Bilaspure/yore/cmd/yore@latest` —
or grab a **prebuilt binary** from the
[Releases](https://github.com/Dev-Bilaspure/yore/releases) page.

### 2. Enable it — required

Add one line to your shell's startup file:

```sh
eval "$(yore init zsh)"     # in ~/.zshrc   (bash and fish have their own line)
```

This installs the hook that records each command's directory, exit status, and
duration, and binds **Ctrl-G** to the interactive picker. **It's the on-switch:**
without it `yore` still works for basic recall from your existing shell history,
but project scoping (`yore here`), what-worked filtering (`--ok`), and Ctrl-G
won't — those need the recording this turns on.

Open a new terminal (or `source ~/.zshrc`) for it to take effect.

### 3. Seed from your existing history (recommended)

```sh
yore import
```

Backfills past commands from `~/.zsh_history` / `~/.bash_history` so yore is
useful immediately instead of only learning from here on.

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
    "github.com/Dev-Bilaspure/yore/pkg/store"
    "github.com/Dev-Bilaspure/yore/pkg/history"
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

Releases are fully automated by [goreleaser](https://goreleaser.com) via GitHub
Actions. One-time setup (already done): a `Dev-Bilaspure/homebrew-tap` repo and a
`HOMEBREW_TAP_GITHUB_TOKEN` secret with write access to it.

To cut a release, just push a version tag:

```sh
git tag v0.2.0
git push origin v0.2.0
```

The workflow builds binaries for linux/darwin/windows × amd64/arm64, publishes a
GitHub Release, and updates the Homebrew cask in the tap. Users get it with
`brew upgrade yore` — no re-tap or re-trust needed. To validate config locally
before tagging: `goreleaser check` (or a dry run with
`goreleaser release --snapshot --clean --skip=publish`).

## Contributing

Issues and PRs welcome. Run `make check` (build + vet + lint + race tests) before
submitting. New shell parsers implement `history.Source`; new shells for
recording add an `internal/cli/shellinit/*` script.

## License

[MIT](LICENSE) © Dev Bilaspure
