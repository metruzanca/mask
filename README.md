# mask

Wear a different git identity and SSH key per shell, without editing git config.

`mask` sets your git `user.name`/`user.email` and the SSH key `git push` uses,
so you can commit and push as different people from the same machine.

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
mask off             # take it off
```

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

## Notes

- A mask lasts for the shell session. New terminals start unmasked.
- Identity and keys are applied through environment variables, so nothing is
  written to your git config.
- Secrets never touch shell history.
