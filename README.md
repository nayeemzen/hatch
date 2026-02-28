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

Download from GitHub Releases and place `hatch` in your `PATH`.

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
