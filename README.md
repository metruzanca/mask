# mask

Wear a different git identity and SSH key per shell, without editing git config.

`mask` sets your git `user.name`/`user.email` and the SSH key `git push` uses,
so you can commit and push as different people from the same machine.

By default it is **automatic**: when you `cd` into a git repository it puts on
the mask you last wore there, and when you leave a repository it takes the mask
off. It remembers each repo independently, handles nested repos (innermost
wins), logs every automatic change, and shows a notice when it acts.

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

## Automatic mode

Automatic mode is on by default after you load the shell wrapper. It works like
this:

- Enter a repo you have worn a mask in before and it puts that mask on.
- Enter a repo with no memory, and it takes the mask off.
- Leave every repo and it takes the mask off.
- Nested repos resolve to the innermost one. Leaving the inner repo restores
  the outer repo's mask.
- A new shell that starts inside a repo syncs immediately.

A repo's memory is written when you run `mask switch <name>` inside it, so
`mask switch work` in a repo means "wear `work` here from now on". `mask off`
both takes the mask off and forgets the repo, so it will not come back.

Every automatic change is written to `~/.config/mask/auto.log` and printed as a
notice like `mask: entered ~/dev/foo -> wearing "work"`.

Turn it all off with `mask settings off`, or in the config:

```toml
[settings]
automatic = false
```

In manual mode `mask` only changes when you run `mask switch` / `mask off`.

## Files

All under `~/.config/mask/` (or `$XDG_CONFIG_HOME/mask/`):

- `config.toml` — masks and settings. Hand-edited or via `mask create` /
  `mask settings`.
- `repos.toml` — the repo-to-mask memory used by automatic mode. Separate on
  purpose, so you can commit it to your dotfiles or ignore it independently.
- `auto.log` — append-only record of automatic changes, capped at 1000 lines.

`repos.toml` and `auto.log` are safe to delete; the memory is rebuilt as you
switch masks in repos.

## Config

Lives at `~/.config/mask/config.toml`. Env vars are hand-edited:

```toml
[mask.personal]
  git_name  = "metru.dev"
  git_email = "metru@zanca.dev"
  ssh_key   = "~/.ssh/id_ed25519"

[mask.personal.env]
  GITHUB_TOKEN = "ghp_x"

[mask.work]
  git_name  = "Metru Zanca"
  git_email = "metru@zanca.dev"
  ssh_key   = "~/.ssh/id_ed25519_terminal_shop"
```

Config also holds settings:

```toml
[settings]
automatic = true
```

## Notes

- In manual mode a mask lasts for the shell session; new terminals start
  unmasked. In automatic mode a new terminal syncs to the repo it starts in.
- Identity and keys are applied through environment variables, so nothing is
  written to your git config.
- Secrets never touch shell history.
- Re-run `mask init <shell>` after upgrading from an older version to install
  the directory hook. Existing shells need a restart to pick it up.
- Worktrees and submodules are tracked by their git directory, so two worktrees
  of the same repository share one memory entry.
