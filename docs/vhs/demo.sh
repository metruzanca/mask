# Build a throwaway HOME and two temporary repositories for the VHS demos.
#
# Source this from an interactive bash shell while the repository root is the
# working directory (the VHS tapes do this for you):
#
#   source docs/vhs/demo.sh
#
# It never reads or writes the real ~/.config/mask, ~/.gitconfig, or ~/.ssh.
# Override the throwaway location with MASK_DEMO_ROOT.

__mask_demo_setup() {
  set -e

  local demo_root repo_root script_dir
  demo_root=${MASK_DEMO_ROOT:-/tmp/opencode/mask-demo}
  script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
  repo_root=$(cd "$script_dir/../.." && pwd)

  if [ -e "$demo_root" ]; then
    chmod -R u+w "$demo_root"
    rm -rf "$demo_root"
  fi
  mkdir -p "$demo_root/bin" "$demo_root/home/dev" "$demo_root/home/.config"

  # Ignore any mask the recording shell may be wearing, so a real identity
  # cannot leak into the demo.
  if [ -n "${MASK_ENV_MASK:-}" ]; then
    local user_var
    for user_var in $MASK_ENV_MASK; do
      unset "$user_var"
    done
  fi
  unset MASK_NAME MASK_REPO MASK_ENV_MASK MASK_LAST_NAME MASK_LAST_PWD
  unset GIT_AUTHOR_NAME GIT_AUTHOR_EMAIL GIT_COMMITTER_NAME GIT_COMMITTER_EMAIL
  unset GIT_SSH_COMMAND GIT_CONFIG_COUNT SSH_AUTH_SOCK AWS_PROFILE
  local git_var
  while IFS= read -r git_var; do
    unset "$git_var"
  done < <(env | sed -n 's/^\(GIT_CONFIG_\(KEY\|VALUE\)_[0-9][0-9]*\)=.*/\1/p')

  # Build with the real HOME so the Go caches stay out of the demo.
  go build -o "$demo_root/bin/mask" "$repo_root"

  export MASK_DEMO_ROOT=$demo_root
  export HOME=$demo_root/home
  export XDG_CONFIG_HOME=$HOME/.config
  export GIT_CONFIG_NOSYSTEM=1
  export GIT_CONFIG_GLOBAL=$HOME/.gitconfig
  export STARSHIP_CONFIG=$HOME/.config/starship.toml
  export PATH="$demo_root/bin:$PATH"

  cd "$HOME"
  git config --global init.defaultBranch main
  git config --global color.ui auto

  mkdir -p "$HOME/.ssh"
  ssh-keygen -q -t ed25519 -N '' -C sam@huddlesurety.dev -f "$HOME/.ssh/id_ed25519_huddlesurety"
  ssh-keygen -q -t ed25519 -N '' -C sam@fuelmap.dev -f "$HOME/.ssh/id_ed25519_fuelmap"

  mkdir -p "$XDG_CONFIG_HOME/mask"
  cat >"$XDG_CONFIG_HOME/mask/config.toml" <<'EOF'
[mask.huddlesurety]
  git_name  = "Sam Rivera"
  git_email = "sam@huddlesurety.dev"
  ssh_key   = "~/.ssh/id_ed25519_huddlesurety"

[mask.huddlesurety.env]
  AWS_PROFILE = "huddlesurety"

[mask.fuelmap]
  git_name  = "Sam"
  git_email = "sam@fuelmap.dev"
  ssh_key   = "~/.ssh/id_ed25519_fuelmap"

[mask.fuelmap.env]
  AWS_PROFILE = "fuelmap"

[settings]
  automatic = true
  notify    = true
EOF

  cat >"$STARSHIP_CONFIG" <<'EOF'
add_newline = false
format = "$directory$git_branch${custom.mask}$character"

[directory]
truncation_length = 1
style = "bold cyan"

[git_branch]
format = "[$branch]($style) "
style = "bold yellow"

[character]
success_symbol = "[❯](bold green)"
error_symbol = "[❯](bold red)"

[custom.mask]
command = 'printf %s "$MASK_NAME"'
when = 'test -n "$MASK_NAME"'
format = '[🎭 $output](bold purple) '
description = "currently worn mask"
EOF

  # Create both repositories and remember a mask in each, so the tapes can
  # start with repos that already have a per-repo memory.
  local repo
  for repo in huddlesurety fuelmap; do
    mkdir -p "$HOME/dev/$repo"
    cd "$HOME/dev/$repo"
    git init -q
    echo "# $repo" >README.md
    git add README.md
    git -c user.name='Setup' -c user.email='setup@example.com' commit -q -m 'Initial commit'
    "$demo_root/bin/mask" --shell=bash switch "$repo" >/dev/null 2>&1
  done

  cd "$HOME"

  eval "$(starship init bash)"
  eval "$(mask init bash)"

  set +e
}

__mask_demo_setup
unset -f __mask_demo_setup
