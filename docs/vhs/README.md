# Demo recordings

The GIFs in [`docs/gifs`](../gifs) are recorded with
[VHS](https://github.com/charmbracelet/vhs) from the tapes in this directory.

Each tape sources [`demo.sh`](demo.sh), which builds the working copy and sets
up a throwaway `HOME` with two temporary repositories and two masks:

| Mask | Git identity | SSH key | Environment |
| --- | --- | --- | --- |
| `huddlesurety` | Sam Rivera `<sam@huddlesurety.dev>` | `~/.ssh/id_ed25519_huddlesurety` | `AWS_PROFILE=huddlesurety` |
| `fuelmap` | Sam `<sam@fuelmap.dev>` | `~/.ssh/id_ed25519_fuelmap` | `AWS_PROFILE=fuelmap` |

The demo never reads or writes your real `~/.config/mask`, `~/.gitconfig`, or
`~/.ssh`. The throwaway tree lives at `/tmp/opencode/mask-demo` by default;
override it with `MASK_DEMO_ROOT`.

| Tape | GIF | Shows |
| --- | --- | --- |
| `auto-switch.tape` | `auto-switch.gif` | Repository memory, automatic switching, and the Starship mask module |
| `commit-identity.tape` | `commit-identity.gif` | Empty commits recorded with the worn mask's git user |
| `env-vars.tape` | `env-vars.gif` | Per-mask environment variables appearing and disappearing |
| `manual-switch.tape` | `manual-switch.gif` | `mask list`, toggling with `mask switch`, and `mask` status |
| `ssh-key.tape` | `ssh-key.gif` | `GIT_SSH_COMMAND` switching to the mask's SSH key |

## Requirements

`vhs` 0.12 or newer, including `ttyd` and `ffmpeg`, plus `bash`, `starship`,
`git`, `go`, and `ssh-keygen`. The demos use the Fira Code Nerd Font, but VHS
falls back to another font when it is missing.

## Recording

Run from the repository root:

```sh
./docs/vhs/record.sh                 # record every tape
./docs/vhs/record.sh auto-switch     # or one tape by name
```

The `Output` path in each tape is relative to the repository root, so vhs must
be run from there. Re-record the affected GIFs whenever the prompt, notices, or
CLI output changes, then commit the updated GIFs with the tapes.
