# hatch

`hatch` is a small Go CLI for creating, cloning, and browsing throwaway projects in `~/hatchery`.

## Why this exists

`tobi/try` inspired this workflow. We wanted the same fast project incubation loop with a single static binary and no Ruby runtime.

## Features

- `hatch <name>`: create `~/hatchery/<yyyy-mm-dd>-<name>`
- `hatch <path> <name>`: copy a source directory into `~/hatchery/<yyyy-mm-dd>-<name>`
- `hatch` interactive browser:
- Type to fuzzy filter projects
- Arrow keys to move
- `Enter` to open
- `Ctrl+A` to archive into `~/hatchery/archive/`
- `Ctrl+R` to remove
- Shell hook for auto-`cd`

## Install

### Prebuilt binary

Install directly from a GitHub release:

```bash
VERSION=v0.1.0
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "unsupported arch: $ARCH" && exit 1 ;;
esac

curl -fsSL "https://github.com/nayeemzen/hatch/releases/download/${VERSION}/hatch_${VERSION}_${OS}_${ARCH}.tar.gz" | tar -xz
install -m 0755 hatch ~/.local/bin/hatch
```

If `~/.local/bin` is not in your `PATH`, add it:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Release files are available at:
`https://github.com/nayeemzen/hatch/releases`

### Go install

```bash
go install github.com/nayeemzen/hatch@latest
```

## Shell setup

`hatch` uses a shell hook so the parent shell can `cd` after selection.

### zsh

```bash
eval "$(hatch --init zsh)"
```

### bash

```bash
eval "$(hatch --init bash)"
```

### fish

```fish
hatch --init fish | source
```

## Usage

```bash
hatch --help
```

Examples:

```bash
hatch spike-auth
hatch ~/templates/service-base payment-service
hatch
```

## Development

```bash
go test ./...
```

## License

MIT
