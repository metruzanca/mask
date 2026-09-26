# mask

Wear a different git identity and SSH key per shell, without editing git config.

`mask` sets your git `user.name`/`user.email` and the SSH key `git push` uses,
so you can commit and push under different identities from the same machine.
For example, work Sam and personal Sam are two masks with different names,
emails, and SSH keys.

Identity and keys are applied through environment variables, so nothing is
written to your git config, and secrets never touch shell history.

By default it is **automatic**: when you `cd` into a git repository it puts on
the mask you last wore there, and when you leave a repository it takes the mask
off. It works like this:

- Enter a repo you have worn a mask in before and it puts that mask on.
- Enter a repo with no memory, and it takes the mask off.
- Leave every repo and it takes the mask off.
- Nested repos resolve to the innermost one. Leaving the inner repo restores
  the outer repo's mask.
- Worktrees and submodules are tracked by their git directory, so two worktrees
  of the same repository share one memory entry.
- A new shell that starts inside a repo syncs immediately.

A repo's memory is written when you run `mask switch <name>` inside it, so
`mask switch work` in a repo means "wear `work` here from now on". `mask off`
both takes the mask off and forgets the repo, so it will not come back.

Every automatic change prints a notice like
`mask: entered ~/dev/foo -> wearing "work"`.

## Install

```sh
go install github.com/metruzanca/mask@latest
```

Then load the shell wrapper once. For fish, add to `~/.config/fish/config.fish`:

```sh
mask init fish | source
```

bash/zsh, add to `~/.bashrc` or `~/.zshrc`:

```sh
eval "$(mask init bash)"
```

Re-run `mask init <shell>` after upgrading from an older version to install the
directory hook, then restart your shell so it takes effect.

## Use

```sh
mask create          # form: name, git user.name/email, SSH key
mask switch          # toggle between two masks, dropdown for three or more
mask switch work     # or name one directly
mask                 # what you're wearing now
mask list            # all configured masks
mask off             # take it off, and forget the current repo
mask settings        # automatic vs manual control
mask settings off    # same, without the form
```

## Manual mode

To take control, turn automatic mode off with `mask settings off` (or set
`automatic = false` under `[settings]` in the config). The mask then only
changes when you run `mask switch` / `mask off`, and it lasts for the shell
session: a new terminal starts unmasked. The repo memory is still kept up to
date, it just never changes your mask on its own. Turning automatic back on
picks up right where you left off.

## Files

All under `~/.config/mask/` (or `$XDG_CONFIG_HOME/mask/`):

- `config.toml`: masks and settings. Hand-edited or via `mask create` /
  `mask settings`.
- `repos.toml`: the repo-to-mask memory used by automatic mode. Separate on
  purpose, so you can commit it to your dotfiles or ignore it independently.

`repos.toml` is safe to delete; the memory is rebuilt as you switch masks in
repos.

## Config

Lives at `~/.config/mask/config.toml`. Env vars are hand-edited:

```toml
[mask.personal]
  git_name  = "Sam"
  git_email = "sam@personal.example"
  ssh_key   = "~/.ssh/id_ed25519_personal"

[mask.personal.env]
  GITHUB_TOKEN = "ghp_x"

[mask.work]
  git_name  = "Sam Rivera"
  git_email = "sam@work.example"
  ssh_key   = "~/.ssh/id_ed25519_work"
```

Config also holds settings:

```toml
[settings]
automatic = true
```
