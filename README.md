# mask

[![Release](https://github.com/metruzanca/mask/actions/workflows/release.yml/badge.svg)](https://github.com/metruzanca/mask/actions/workflows/release.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/metruzanca/mask)](go.mod)
[![Latest release](https://img.shields.io/github/v/release/metruzanca/mask)](https://github.com/metruzanca/mask/releases)

`mask` lets you wear a different git identity in each repository. It sets your
git `user.name`/`user.email` and the SSH key `git push` uses, so you can commit
and push as different people from the same machine without editing config.

Work Sam and personal Sam are two masks. Each carries its own name, email, SSH
key, and any extra environment variables you attach to it.

Everything is applied through environment variables. Nothing is written to your
git config, and secrets never touch shell history.

By default it is **automatic**: enter a repository and it puts on the mask you
last wore there; leave and it takes the mask off.

[Installation](#installation) • [Quick start](#quick-start) • [Automatic mode](#automatic-mode) • [Configuration](#configuration) • [Troubleshooting](#troubleshooting)

## Features

* **Per-repository memory.** A repo remembers the mask you last wore in it, and
  restores it whenever you `cd` back in.
* **Nothing written to git config.** Identity and SSH key are exported as
  environment variables, so a read-only or managed `~/.gitconfig` is fine.
* **SSH key per identity.** `git push` uses the key belonging to the mask, not
  whichever key happens to be first in your agent.
* **Extra environment variables.** Attach `GITHUB_TOKEN`, `AWS_PROFILE`, or
  anything else to a mask and have it appear and disappear with it.
* **Nested repos and worktrees.** The innermost repository wins. Worktrees and
  submodules of one repository share a single memory entry.
* **No dependencies.** One static Go binary, plus a small shell wrapper for
  fish, bash, or zsh.
* **Interactive when you want it.** `mask create` and `mask switch` use forms
  and dropdowns, and every command also works with an explicit name.

## Quick start

Install the binary and load the shell wrapper once:

```sh
go install github.com/metruzanca/mask@latest
```

Fish, in `~/.config/fish/config.fish`:

```sh
mask init fish | source
```

bash or zsh, in `~/.bashrc` or `~/.zshrc`:

```sh
eval "$(mask init bash)"
```

Now create your first mask:

```sh
$ mask create
```

The form asks for a name, git name, email, and SSH key. Then pick a repository
and choose a mask there:

```sh
$ cd ~/dev/work-site
$ mask switch work
$ mask
Wearing mask: work
Git identity: Sam Rivera <sam@work.example>
Mode: automatic
Repo: /home/sam/dev/work-site
Remembered here: work
```

From that point on, entering `~/dev/work-site` puts `work` on automatically:

```sh
$ cd ~/dev/work-site
mask: entered /home/sam/dev/work-site -> wearing "work"

$ cd ~/dev/personal-blog
mask: entered /home/sam/dev/personal-blog -> no mask remembered, mask off
```

Wear a mask in `personal-blog` once, and it will be restored on every later
visit to that repository.

Check what is available and what is worn:

```sh
$ mask list
* work      Sam Rivera <sam@work.example>
  personal  Sam <sam@personal.example>
```

## How it works

`mask` never edits `~/.gitconfig`. When a mask is applied, the shell wrapper
sources the output of `mask switch`, which exports:

* `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`
* `GIT_COMMITTER_NAME`, `GIT_COMMITTER_EMAIL`
* `GIT_SSH_COMMAND` (the mask's SSH key, with `IdentitiesOnly=yes`)
* `GIT_CONFIG_*` entries for `user.name`, `user.email`, and `core.sshCommand`
* `MASK_NAME`, plus any environment variables defined on the mask

Because these are environment variables, they apply to the current shell and
everything it starts. A new shell starts clean and syncs itself when the
directory hook runs.

## Usage

```sh
mask create          # form: name, git user.name/email, SSH key
mask switch          # toggle between two masks, dropdown for three or more
mask switch work     # or name one directly
mask                 # what you're wearing now
mask list            # all configured masks
mask off             # take it off, and remember that here
mask on              # put the last mask back on
mask settings        # automatic vs manual control
mask settings off    # same, without the form
```

### Switching

* With two masks, `mask switch` toggles to the other one.
* With three or more, it opens a dropdown. Pass a name to skip the dropdown,
  which also works in scripts and non-interactive terminals.
* `mask switch <name>` inside a repository is remembered for that repository
  when automatic mode is on.

### Turning a mask off, and back on

`mask off` clears the identity and remembers "no mask here" for the current
repository, so it stays off. `mask on` reverses that: it restores the mask
remembered for the repository, or the last mask worn in the shell when the
repository has no memory.

## Automatic mode

When the shell wrapper is loaded, `mask` reacts to directory changes:

* Enter a repo you have worn a mask in before and it puts that mask on.
* Enter a repo with no memory and it takes the mask off.
* Leave every repo and it takes the mask off.
* Nested repos resolve to the innermost one. Leaving the inner repo restores
  the outer repo's mask.
* Worktrees and submodules are tracked by their git directory, so two worktrees
  of the same repository share one memory entry.
* A new shell that starts inside a repo syncs immediately.

Every automatic change prints a notice like
`mask: entered ~/dev/foo -> wearing "work"`.

Repo memory is written when you run `mask switch <name>` inside a repository.
`mask off` inside a repository records an explicit "no mask here", so the
absence of a mask is also a decision that sticks.

## Manual mode

To take control, turn automatic mode off:

```sh
mask settings off
```

Or set `automatic = false` under `[settings]` in the config file. The mask then
only changes when you run `mask switch` or `mask off`, and it lasts for the
shell session. A new terminal starts unmasked.

Repo memory is not touched while manual, so `switch` and `off` change only the
current shell. Turning automatic back on picks up wherever the memory was left.

## Configuration

The config lives at `~/.config/mask/config.toml` (or
`$XDG_CONFIG_HOME/mask/config.toml`). `mask create` writes the basics; edit the
file to add environment variables or fine-tune settings.

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

[settings]
automatic = true
```

### Choosing a default

How you split things up is up to you:

* **Mask as your default.** Create a mask for your main identity and wear it in
  your everyday repos. Everything is set explicitly, so you never depend on the
  machine's global git config.
* **Global config as your default.** Leave your main identity in `~/.gitconfig`
  and create masks only for other identities, such as `work`. Repos that should
  use the default stay unmasked; run `mask switch work` in the rest.

Either way, automatic mode applies the mask a repo was last given, and a repo
with no memory falls back to your global git config.

## Files

All files live under `~/.config/mask/` (or `$XDG_CONFIG_HOME/mask/`):

* `config.toml`: masks and settings. Hand-edited or managed with `mask create`
  and `mask settings`.
* `repos.toml`: the repository-to-mask memory used by automatic mode. Kept
  separate on purpose, so you can commit it to your dotfiles or ignore it
  independently.
* `auto.log`: a bounded log of automatic changes, useful when debugging why a
  mask was applied.

`repos.toml` is safe to delete; the memory is rebuilt as you switch masks in
repositories. `auto.log` is safe to delete too.

## Installation

### With Go

```sh
go install github.com/metruzanca/mask@latest
```

Go 1.26 or later is required.

### Prebuilt binaries

Download Linux, macOS, or Windows binaries from the
[releases page](https://github.com/metruzanca/mask/releases).

### From source

```sh
git clone https://github.com/metruzanca/mask
cd mask
go install .
```

### Upgrading

After upgrading from an older version, re-run `mask init <shell>` to install the
current directory hook, then restart your shell:

```sh
eval "$(mask init bash)"
```

## Troubleshooting

### The mask changes when I `cd`, but not in a new shell

The wrapper runs the hook at startup, so this usually means the wrapper is not
loaded. Check that `mask init <shell>` is evaluated in the right startup file,
then restart the shell.

### `mask switch` asks for a terminal

With three or more masks, `mask switch` needs a dropdown. Pass a name to choose
without one:

```sh
mask switch work
```

### A repository does not get a mask automatically

Automatic mode may be off, or the repository may have an explicit "no mask"
memory. Check with:

```sh
mask
mask settings
```

Then run `mask switch <name>` inside the repository to remember a mask there.

### `git push` uses the wrong SSH key

Print what is currently exported:

```sh
env | grep -E 'GIT_SSH_COMMAND|GIT_CONFIG'
```

The mask sets `GIT_SSH_COMMAND` to `ssh -i <key> -o IdentitiesOnly=yes`. If you
override it elsewhere, the mask's value can lose. Also make sure the key exists
and is readable.

### Nested repositories wear the wrong mask

Nested repos resolve to the innermost repository. A worktree or submodule is
identified by its git directory, so all of its worktrees share one memory
entry. If that is not what you want, wear the mask you need after entering the
inner repository.

## Development

```sh
git clone https://github.com/metruzanca/mask
cd mask

# Build
go build

# Run tests
go test ./...

# Install the working copy
go install .
```

Releases are built by GoReleaser for Linux, macOS, and Windows on `amd64` and
`arm64`. Tag a commit with `vMAJOR.MINOR.PATCH` to publish one.
