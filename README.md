# agentsmd

Keeps `AGENTS.md`, `CLAUDE.md`, and `GEMINI.md` in sync by making `AGENTS.md` the source of truth and the others symlinks to it.

## Install

```sh
go install github.com/joargp/agentsmd@latest
```

Or download a binary from [releases](https://github.com/joargp/agentsmd/releases).

## Usage

```sh
agentsmd          # fix state: create AGENTS.md + symlinks as needed
agentsmd --check  # exit 1 if state is wrong, change nothing
```

Result:

```
AGENTS.md              # regular file
CLAUDE.md -> AGENTS.md # symlink
GEMINI.md -> AGENTS.md # symlink
```
