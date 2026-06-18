# yore

**The commands you rely on, learned from your shell history.**

yore quietly surfaces the commands you reach for most, and lets you promote the
keepers into reusable recipes.

*For developers who live in the terminal and keep re-searching or re-typing the
same commands.*

[![CI](https://github.com/Dev-Bilaspure/yore/actions/workflows/ci.yml/badge.svg)](https://github.com/Dev-Bilaspure/yore/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Dev-Bilaspure/yore?sort=semver)](https://github.com/Dev-Bilaspure/yore/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/Dev-Bilaspure/yore.svg)](https://pkg.go.dev/github.com/Dev-Bilaspure/yore)
[![Go Report Card](https://goreportcard.com/badge/github.com/Dev-Bilaspure/yore)](https://goreportcard.com/report/github.com/Dev-Bilaspure/yore)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

<p align="center"><img src="docs/demo.gif" alt="yore demo: project-aware recall, a saved recipe, and the Ctrl-G picker" width="820"></p>

Your shell records everything you type but understands none of it. The fiddly
command you worked out three months ago is in there — buried among thousands of
one-offs you'll never run again. So you re-derive it. Nothing ever marked it as
worth keeping.

Most tools here ask you to curate first — to write the commands worth keeping
down before they help. `yore` works the other way around: it starts from what
you already run and learns which commands you actually rely on. The ones that
matter then have a natural next step — promote a command into a reusable,
parameterized recipe, and commit it when it's worth sharing.

Frecency, project scope, and exit status are just how yore reads "what you rely
on" from real usage. The structure emerges from your history; you don't author it
up front.

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

yore is a single local binary; the commands you keep are plain text you can
commit, so they travel with the repo when you want them to.

---

## Contents

- [Why yore, if I already use atuin / fzf / just?](#why-yore-if-i-already-use-atuin--fzf--just)
- [Install](#install)
- [Use it](#use-it) · [Recall](#recall) · [Interactive pick](#interactive-pick-ctrl-g) · [Recipes](#recipes--promote-what-you-rely-on) · [Suggestions](#suggestions--let-yore-promote-for-you)
- [How it works](#how-it-works)
- [Privacy](#privacy)
- [Commands](#commands)
- [Library](#library)
- [Roadmap](#roadmap) · [Contributing](#contributing) · [License](#license)

## Why yore, if I already use `atuin` / `fzf` / `just`?

`fzf` and `atuin` are about **recall** — searching what you've already run
(atuin durably, and synced). They're excellent at it, and that's where they stop.

`just`, `make`, `pet` and `navi` are about **curated commands** — but you write
them by hand, up front, separate from what you actually do, and only if you
remember to.

yore's bet is that you shouldn't have to curate up front. It learns which commands
you rely on from real usage, and lets the ones that matter grow into reusable
commands you can keep and share. It isn't a better history search than atuin or a
better task runner than just — it's the part in between: turning real usage into
kept knowledge without writing it down first.

## Install

### 1. Install the binary

**Homebrew** (macOS / Linux) — also installs completions, the man page, and
`fzf` (used by the Ctrl-G picker):

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

Open a new terminal (or `source ~/.zshrc`) for it to take effect, then run
`yore` — it guides you from there.

> **That's it.** You don't need to import anything: `yore` already reads your
> existing shell history, so recall works from the first run. If you'd like to
> make that history a permanent part of yore's own store (so it survives history
> rotation), you can optionally run `yore import` once.

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
history at once. It uses [`fzf`](https://github.com/junegunn/fzf) for the picker
(installed automatically with the Homebrew package); without fzf it falls back to
a simple numbered menu, so install fzf if you used `go install` or a raw binary.

### Recipes — promote what you rely on

Once a command has earned its place in your history, promote it: name it, mark
the parts that vary. Keep it for yourself, or save it with `--project` to commit
it alongside the code.

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

### Suggestions — let yore promote for you

You don't have to remember to save anything. As you work, yore notices the
commands you run repeatedly and offers to turn them into recipes — parameters
and all, inferred from how you actually used them:

```sh
yore suggest
```

Each candidate comes with the evidence behind its blanks (`{env} = staging, prod`),
so the inference is legible, not magic. Save it, rename it, or dismiss it —
dismissed patterns never come back.

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

Issues and PRs welcome — see [CONTRIBUTING.md](CONTRIBUTING.md) for the full
local dev + test workflow. The short version: `make check` before a PR, and
`make dev` drops you into an isolated sandbox shell to manually try interactive
features (onboarding, Ctrl-G, recording) without touching your real setup.

## License

[MIT](LICENSE) © Dev Bilaspure
