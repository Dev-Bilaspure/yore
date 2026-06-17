# Contributing to yore

Thanks for hacking on yore. This document is the canonical guide to developing
and testing it locally.

## Prerequisites

- **Go 1.22+** (the module targets 1.22; CI runs `stable`).
- **fzf** — used by the Ctrl-G picker (optional for building/tests; recommended for manual testing).
- **golangci-lint v2** — for linting (`brew install golangci-lint`).
- **vhs** — only if you want to re-render the demo GIF (`brew install vhs`).

```sh
git clone https://github.com/Dev-Bilaspure/yore
cd yore
make build      # → ./bin/yore
```

Run `make help` to see every target.

## Project layout

```
cmd/yore            thin entrypoint
internal/cli        the CLI: flags, subcommands, onboarding, output
pkg/store           the event log (append-only JSONL)
pkg/history         parsing + frecency ranking
pkg/recipe          the .yorefile format + rendering
```
`pkg/*` is the reusable, public surface; `internal/cli` is private.

## The test layers

yore is tested at several levels — run the one that fits your change, and
`make check` before opening a PR.

| Layer | Command | What it covers |
|---|---|---|
| Unit | `make test` | parsers, ranking, recipes, store, CLI logic |
| Race | `make race` | the above under `-race` |
| Coverage | `make cover` | per-package coverage |
| Fuzz | `make fuzz` | parsers never panic on arbitrary input |
| Vet | `make vet` | suspicious constructs |
| Lint | `make lint` | golangci-lint (v2 config in `.golangci.yml`) |
| **Everything** | `make check` | fmt + vet + race + lint — **what CI gates on** |

Tests are pure and hermetic: they use in-memory filesystems, `t.TempDir()`, and
injected `Now`/readers, so they never touch your real history or store.

## Manual / interactive testing — the sandbox

Some behaviour is inherently interactive and can't be unit-tested: the
onboarding journey, the Ctrl-G picker, and the shell recording hooks. For those,
use the **sandbox** — an isolated shell with a freshly-built `yore` that touches
**nothing real** (its own temp data dir, config, history, and `PATH`; all
removed on exit):

```sh
make dev          # seeded sample project + recording already on — poke at everything
make dev-fresh    # clean slate — walk the onboarding journey from the first run
```

Or call the script directly for more control:

```sh
scripts/dev-shell.sh --seed --enable --shell=bash
```

Inside the sandbox:
- `yore`, `yore here`, `yore recipes`, … — run anything against isolated data
- `yenable` — load the recording hook (then press **Ctrl-G**)
- `exit` — leave; temp dirs are cleaned up automatically

Because it's fully isolated, you can reset and replay freely. To re-walk
onboarding within a session: `rm -f "$XDG_DATA_HOME"/yore/onboarding.json`.

### Verifying composability
yore must stay a clean Unix filter — guidance goes to **stderr**, gated on an
interactive TTY. Confirm a change didn't break this:
```sh
yore | cat            # must print ONLY commands — no banners
YORE_NO_HINTS=1 yore  # opt-out of guidance
```

## Rendering the demo GIF

```sh
make demo             # renders docs/demo.gif from docs/demo.tape (needs vhs + fzf)
```

## Adding support for a new shell

- **Parsing** (for `import` / `--file`): implement `history.Source` in `pkg/history`
  with table-driven tests over fixtures in `pkg/history/testdata/`.
- **Recording**: add an `internal/cli/shellinit/<shell>.<ext>` script. It must
  `export YORE_SESSION=1` (so onboarding detects the hook) and call
  `yore record …` after each command.

## Pull requests

1. Branch off `main`.
2. Keep changes focused; add tests for new behaviour.
3. Run `make check` — it must be green.
4. For user-facing changes, update the README and `yore --help` text.
5. Open the PR with a clear description of the why.

## Releasing (maintainers)

Releases are automated by goreleaser on a tag push — see the README's
[Releasing](README.md#releasing-maintainers) section.
