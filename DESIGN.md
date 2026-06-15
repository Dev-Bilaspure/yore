# yore — design & architecture

## Vision

Your shell records everything you type but understands none of it. **yore turns
your command history into reusable, project-aware, team-shareable knowledge** —
so you (and your teammates) stop re-deriving commands you've already figured out.

## The problem we actually solve

Operational command knowledge is *trapped*: in flat history (searchable but
dumb), in your head (decays), in stale READMEs, and in high-friction cheatsheets.
"How do I build / test / deploy *this* repo?" has no good home. Existing tools
each solve a slice:

- `fzf`+Ctrl-R / `atuin` / `mcfly` — search history (personal, raw text).
- `just` / `make` / `task` — named commands, but hand-authored.
- `navi` / `tldr` — cheatsheets, manually maintained.

**Nobody nails: auto-learned from real usage + scoped to the current project +
version-controlled so a team shares it.** That is yore's wedge.

## Principles (the soul we keep)

1. **Fast, single binary.** Core engine has no heavy deps; no required daemon, no
   mandatory SQLite, no Ctrl-R hijack.
2. **Composable.** Everything works as a pipe *and* interactively.
3. **Plain-text & git-friendly.** Curated knowledge lives in human-readable files
   you can commit. (This is what binary-SQLite tools can't do.)
4. **Privacy-first.** Recording is opt-in and local; secrets are redacted; nothing
   syncs without explicit action.
5. **Augment, don't replace.** Plays nice with existing history and fzf.

## Architecture

```
            ┌──────────────────────────────────────────────┐
   shell ──▶│ yore record  (hook: cmd, cwd, exit, dur, time)│
   hook     └───────────────┬──────────────────────────────┘
                            ▼
              raw memory  ~/.local/share/yore/history.jsonl   (private, local)
                            │
              import  ◀─────┤  (seed from ~/.zsh_history, ~/.bash_history)
                            ▼
            ┌──────────────────────────────┐     recipes (human-readable):
   yore  ──▶│ recall: frecency + project   │◀──   ~/.config/yore/recipes      (personal)
   yore here│ scope + exit-code awareness  │      <repo>/.yorefile            (team, committed)
            └──────────────────────────────┘
```

### Data model

`store.Event` — one recorded execution:
`Time, Command, Dir, Exit (-1 = unknown), Duration, Shell`.

Stored as append-only JSONL (one event per line). Aggregation/ranking happens in
memory; an index can be added later if logs grow huge (keeps us dependency-free
now).

`recipe.Recipe` — a curated, named command:
`Name, Command (with {param} placeholders), Description, Params, Tags`.

### Interfaces

- `yore`                 interactive / list recall (frecency, deduped).
- `yore here`            commands that matter in the current project/dir.
- `yore init <shell>`    print shell hook + key-binding for eval.
- `yore record ...`      called by the hook; appends an event (best-effort, fast).
- `yore import`          seed the store from existing shell history files.
- `yore save`            promote a command into a recipe (auto-detect {params}).  [phase 2]
- `yore run <name>`      run a recipe, prompting for params.                      [phase 2]
- `yore completion`      shell completion scripts.

All non-interactive output stays pipe-friendly (`yore --print | fzf`).

## Build phases

1. **Foundation** — event model, store, `record`, `init`, `import`, project-aware
   recall. Turns yore context-aware.
2. **Recipes** — `save`/`run`, params, project `.yorefile`, team sharing.
3. **TUI** — first-class interactive picker (intent search + param prompting).
4. **Polish** — redaction in capture, docs, Homebrew/completions refresh.

## Compatibility

The `Event` store supersedes direct HISTFILE reading, but `yore` auto-bootstraps
from existing history on first run (zero-config), and the existing zsh/bash
parsers are reused by `import`. Frecency ranking (`pkg/history`) is unchanged and
becomes the engine under recall.
