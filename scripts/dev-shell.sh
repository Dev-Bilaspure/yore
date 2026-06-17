#!/usr/bin/env bash
#
# dev-shell.sh — launch an isolated sandbox shell with a freshly-built yore.
#
# Lets you manually exercise interactive behaviour (onboarding, Ctrl-G, the
# recording hook) WITHOUT touching your real install, shell config, history, or
# stored data. Everything lives in temp dirs that are removed on exit.
#
# Usage:
#   scripts/dev-shell.sh [--seed] [--enable] [--shell=zsh|bash]
#     --seed     pre-populate a sample project with recorded commands + a recipe
#     --enable   load the recording hook immediately (Ctrl-G works on launch)
#     --shell=   which shell to launch (default: $SHELL, falling back to zsh)
#
# Inside the sandbox:
#   yore        run it (recall / onboarding)
#   yenable     load the recording hook, then press Ctrl-G
#   exit        leave the sandbox (temp dirs are cleaned up automatically)
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

seed=false
enable=false
shell_name=$(basename "${SHELL:-zsh}")
for arg in "$@"; do
  case "$arg" in
    --seed) seed=true ;;
    --enable) enable=true ;;
    --shell=*) shell_name="${arg#*=}" ;;
    -h | --help)
      sed -n '3,18p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "dev-shell: unknown argument: $arg" >&2
      exit 2
      ;;
  esac
done
case "$shell_name" in zsh | bash) ;; *) shell_name=zsh ;; esac

echo "▶ building yore…"
bin_dir=$(mktemp -d)
data_dir=$(mktemp -d)
config_dir=$(mktemp -d)
hist_file=$(mktemp)
zdot_dir=$(mktemp -d)
cleanup() { rm -rf "$bin_dir" "$data_dir" "$config_dir" "$hist_file" "$zdot_dir"; }
trap cleanup EXIT

go build -o "$bin_dir/yore" ./cmd/yore

export PATH="$bin_dir:$PATH"
export XDG_DATA_HOME="$data_dir"
export XDG_CONFIG_HOME="$config_dir"
export HISTFILE="$hist_file"

# A small fallback history so recall has something to show even un-seeded.
cat >"$hist_file" <<'EOF'
git status
git push origin main
make test
docker compose up -d
kubectl -n staging logs -f deploy/api
EOF

start_dir="$repo_root"
if $seed; then
  proj="$data_dir/sample-project"
  mkdir -p "$proj/.git"
  (
    cd "$proj"
    for _ in 1 2 3; do yore record --command 'make test' --dir "$proj" --exit 0; done
    yore record --command 'docker compose up -d' --dir "$proj" --exit 0
    yore record --command './deploy.sh --prod' --dir "$proj" --exit 1
    yore save --project deploy -- ./deploy.sh --env '{env}'
  ) >/dev/null
  start_dir="$proj"
fi

# Allow automated checks (CI) to validate setup without an interactive shell.
if [ "${YORE_DEV_NO_SHELL:-}" = "1" ]; then
  echo "✓ sandbox prepared (YORE_DEV_NO_SHELL=1; not launching a shell)"
  yore --version
  exit 0
fi

banner() {
  cat <<EOF

── yore sandbox ── isolated; nothing here touches your real setup.
  yore        run it (recall / onboarding)
  yenable     load the recording hook (then press Ctrl-G)
  exit        leave the sandbox
EOF
}

cd "$start_dir"
case "$shell_name" in
zsh)
  {
    echo "PROMPT='%F{yellow}[yore-sandbox]%f %1~ %# '"
    echo "alias yenable='eval \"\$(yore init zsh)\"'"
    $enable && echo 'eval "$(yore init zsh)"'
    declare -f banner
    echo "banner"
  } >"$zdot_dir/.zshrc"
  ZDOTDIR="$zdot_dir" exec zsh -i
  ;;
bash)
  rc="$zdot_dir/bashrc"
  {
    echo "PS1='[yore-sandbox] \w \$ '"
    echo "alias yenable='eval \"\$(yore init bash)\"'"
    $enable && echo 'eval "$(yore init bash)"'
    declare -f banner
    echo "banner"
  } >"$rc"
  exec bash --rcfile "$rc" -i
  ;;
esac
