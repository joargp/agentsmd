# agentsmd

Small Go CLI that enforces `AGENTS.md` as the single source of truth for agent instructions, with `CLAUDE.md` and `GEMINI.md` as symlinks to it.

## Build & Test

```sh
go test ./...
go build -o agentsmd
```

## Usage

```sh
./agentsmd          # fix state: create AGENTS.md + symlinks as needed
./agentsmd --check  # exit 1 if state is wrong, change nothing
```

Expected state:

```
AGENTS.md              # regular file
CLAUDE.md -> AGENTS.md # symlink
GEMINI.md -> AGENTS.md # symlink
```

## Architecture

- `main.go` — parses `--check` flag and delegates to `run()`
- `agentsmd.go` — core logic:
  - `classify()` — checks if a file is missing, regular, or a symlink (and to what target)
  - `audit()` — classifies all three files
  - `run()` — handles all cases:
    - No files exist → create empty `AGENTS.md`
    - Only `AGENTS.md` exists → create symlinks
    - Only one non-AGENTS file exists → rename it to `AGENTS.md`
    - Multiple identical files → pick `AGENTS.md` (or rename alphabetically first)
    - Multiple files with different content → interactive conflict resolution
  - `resolveConflict()` — prompts user to pick which file becomes the source of truth
  - `ensureSymlinks()` — creates/fixes symlinks for `CLAUDE.md` and `GEMINI.md`
- `agentsmd_test.go` — table-driven tests using temp dirs and stdout capture

## Agent Conventions

- Keep `AGENTS.md` as the only regular file. Do not create separate `CLAUDE.md` or `GEMINI.md` with different content — the tool will detect the conflict and either prompt or, in `--check` mode, fail.
- Symlinks are relative (`AGENTS.md`), not absolute paths.
- The tool never modifies the content of `AGENTS.md`; it only manages file presence and symlink targets.
