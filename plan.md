# Plan: Migrate `wt` from Zsh to Go

## Problem Statement

The `wt` worktree manager is a ~1000-line zsh function that manages git worktrees with tmux session integration, fzf-based fuzzy navigation, and GitHub CLI integration. It has grown beyond what's comfortable to maintain as shell script. Migrate it to a standalone Go application with a polished charmbracelet TUI.

## Decisions (from interview)

| Decision | Choice |
|---|---|
| Shell integration | Tmux session-switching only (always inside tmux) |
| Interactive UI | Built-in TUI — charmbracelet (bubbletea, bubbles, lipgloss) |
| Initial scope | `go` + `status` subcommands; `sync`/`prune` added later |
| Project location | New standalone GitHub repo |
| Configuration | XDG config file (`~/.config/wt/config.toml`) with `~/dev` as default |
| Binary name | `wt` |
| Platforms | Linux + macOS + Windows (WSL) |
| Distribution | Nix flake + goreleaser |

## Trade-offs Accepted

- **No shell `cd`** — tmux session switching replaces directory changing. The binary execs `tmux` directly.
- **No fzf dependency** — replaced by built-in bubbletea fuzzy picker. More code, but zero external TUI deps.
- **Startup cost** — Go binary has ~10ms startup vs instant shell function. Negligible in practice.
- **gh CLI remains external** — for PR listing and fork detection. Could be replaced with go-gh library later.

## Architecture

```
wt/
├── main.go                    # Entry point
├── go.mod / go.sum
├── flake.nix                  # Nix flake
├── .goreleaser.yaml           # Release automation
├── cmd/
│   ├── root.go                # Root cobra command + version
│   ├── go.go                  # `wt go` subcommand
│   └── status.go              # `wt status` subcommand
├── internal/
│   ├── config/
│   │   └── config.go          # XDG config loading (TOML)
│   ├── git/
│   │   ├── worktree.go        # Worktree scanning, primary detection
│   │   ├── remote.go          # Remote URL → slug parsing
│   │   └── status.go          # Dirty count, ahead/behind
│   ├── tmux/
│   │   └── session.go         # Session name derivation, create/switch
│   ├── github/
│   │   └── gh.go              # gh CLI wrapper (PR list, fork detection)
│   ├── tui/
│   │   ├── picker.go          # Fuzzy picker model (bubbletea)
│   │   ├── table.go           # Status table view (lipgloss)
│   │   ├── spinner.go         # Spinner component
│   │   ├── prompt.go          # Confirmation prompts (y/N)
│   │   └── styles.go          # Shared lipgloss styles
│   └── workspace/
│       └── scan.go            # ~/dev directory scanning, candidate collection
├── README.md
└── LICENSE
```

## Phases

### Phase 1 — Project Scaffold
- Initialize Go module (`github.com/ashleyst/wt` or chosen org)
- Set up cobra CLI with `root`, `go`, and `status` subcommands (stubs)
- XDG config system with TOML (`~/.config/wt/config.toml`)
  - `dev_root` (default: `~/dev`)
  - `legacy_roots` (default: `["~/code"]`)
  - `special_dirs` (default: `["~/.config/fleet"]`)
  - `session_name_overrides` (map of path → custom session name)
- Basic Nix flake for building
- `.goreleaser.yaml` for Linux/macOS/Windows(WSL) builds

### Phase 2 — Core Domain Layer
- **git/worktree.go**: Scan `~/dev/*/` for worktree directories, detect primary worktree (`.git` dir vs file), `git worktree add`, `git worktree remove`
- **git/remote.go**: Parse SSH/HTTPS URLs to `owner/repo` slug, detect upstream vs origin, `git fetch`
- **git/status.go**: Dirty file count (`git status --porcelain`), ahead/behind counts (`git rev-list --left-right --count`)
- **tmux/session.go**: Derive session name from path (same logic as `_wt_session_name`), create session, switch client, list sessions, list pane processes
- **github/gh.go**: List open PRs (`gh pr list`), detect fork + parent (`gh repo view`), add upstream remote
- **workspace/scan.go**: Collect candidate directories from configured roots, annotate with tmux session status

### Phase 3 — TUI Components
- **tui/picker.go**: Bubbletea model implementing fuzzy filtered list with:
  - Text input for filtering
  - Scrollable list with keyboard navigation
  - Annotation support (e.g., `[session_name]` suffix, `(origin, PR #42)` metadata)
  - `--query` pre-fill support + auto-select-1 behavior
- **tui/table.go**: Lipgloss-styled table for status output
  - Dynamic column widths
  - Color-coded status indicators (✓/✗/●/—)
  - Terminal width awareness
- **tui/spinner.go**: Spinner for long operations (git fetch, scanning)
- **tui/prompt.go**: Simple y/N confirmation prompt
- **tui/styles.go**: Shared color palette and lipgloss styles

### Phase 4 — `wt go` Subcommand
- Scan workspace candidates (~/dev, ~/code, special dirs, macOS dirs on Darwin)
- Annotate candidates with active tmux sessions
- Add `[+] new worktree` synthetic entry
- Launch picker → dispatch selection:
  - Existing directory → `tmux switch-client`
  - `[+] new worktree` → new worktree flow:
    1. Repo picker (list ~/dev repos + typed owner/repo or URL to clone)
    2. Branch picker (local + remote + PRs, sorted with provenance annotations)
    3. `[+] new branch` → prompt for name + base branch picker
    4. Create worktree + switch tmux session
- Clone flow: `git clone`, detect default branch, detect fork → add upstream
- Match all existing `_wt_go` and `_wt_new_flow` behavior

### Phase 5 — `wt status` Subcommand
- Scan all worktree directories
- Parallel git operations (status, fetch, rev-list) using goroutines
- Build table rows: REPO | WORKTREE | BRANCH | CLEAN | SYNC | SESSION
- Detect orphan tmux sessions
- Render with lipgloss table
- Spinner while scanning

### Phase 6 — Distribution & CI
- GitHub Actions workflow: test, lint (golangci-lint), build
- goreleaser: multi-arch builds, GitHub Releases
- Nix flake: `packages.${system}.default`, dev shell with Go toolchain
- README with installation instructions, usage, screenshots
- Update fleet config to install `wt` binary (replace zsh function)

## Future Work (not in this plan)
- `wt sync` subcommand
- `wt prune` subcommand
- Replace `gh` CLI calls with `go-gh` library for zero external deps
- Shell completions (cobra generates these)
- `wt clone` as standalone subcommand (extract from `go` flow)

## Key Libraries

| Library | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/bubbles` | TUI components (textinput, list, spinner) |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `github.com/BurntSushi/toml` | Config file parsing |
| `github.com/sahilm/fuzzy` | Fuzzy matching (used by bubbles/list) |

## Notes

- The Go binary will exec `git`, `tmux`, and `gh` as subprocesses — same as the shell version. No need for libgit2 bindings.
- The config file is optional — all defaults match current hardcoded behavior, so the binary works out-of-the-box with zero config.
- `_wt_session_name` logic must be exactly preserved for compatibility with existing tmux sessions.
