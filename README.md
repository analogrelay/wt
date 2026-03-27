# wt

Worktree manager with tmux session integration.

`wt` manages git worktrees with tmux session switching, a fuzzy picker for navigation, and a status dashboard — all in a polished terminal UI built with [charmbracelet](https://charm.sh).

## Install

### From source

```bash
go install github.com/analogrelay/wt@latest
```

### Nix flake

```bash
nix profile install github:analogrelay/wt
```

Or add to your flake inputs:

```nix
{
  inputs.wt.url = "github:analogrelay/wt";
}
```

### GitHub Releases

Download pre-built binaries from [Releases](https://github.com/analogrelay/wt/releases).

## Usage

### `wt` / `wt status`

Show a health overview of all worktrees:

```
REPO        WORKTREE  BRANCH          CLEAN  SYNC  SESSION
──────────  ────────  ──────────────  ─────  ────  ───────
wt          main      main            ✓      ✓     ●
dotfiles    main      main            ✗ 3    ✓     —
myproject   main      main            ✓      ↑2    ●
myproject   feature   feature/login   ✓      ✓     —
```

Columns:
- **REPO** — repository directory name
- **WORKTREE** — worktree directory name
- **BRANCH** — current branch (or short SHA if detached)
- **CLEAN** — ✓ if clean, ✗ N if dirty with N uncommitted changes
- **SYNC** — ✓ if in sync, ↑N ahead, ↓N behind, — if no tracking branch
- **SESSION** — ● if tmux session is active, — otherwise

### `wt go [query]`

Navigate to a project directory with a fuzzy picker:

```bash
wt go           # open picker with all candidates
wt go myproj    # pre-filter picker; auto-select if one match
```

The picker shows:
- All worktree directories under `~/dev/*/*` and `~/code/*/*`
- Special directories (e.g. `~/.config/fleet`)
- Active tmux sessions annotated with `[session_name]`
- `[+] new worktree` to create a new worktree

#### New worktree flow

Selecting `[+] new worktree` launches an interactive flow:

1. **Repo picker** — select from existing repos or type `owner/repo` / URL to clone
2. **Branch picker** — select from local, remote, and PR branches (with provenance annotations)
3. **New branch** — optionally create a new branch from a selected base

## Configuration

Optional config at `~/.config/wt/config.toml`:

```toml
# Root directory for worktrees (default: ~/dev)
dev_root = "~/dev"

# Legacy roots to also scan (default: ["~/code"])
legacy_roots = ["~/code"]

# Special directories to include (default: ["~/.config/fleet"])
special_dirs = ["~/.config/fleet"]

# Override session names for specific paths
[session_name_overrides]
# "~/dev/myrepo" = "custom-name"
```

All settings are optional — `wt` works with zero configuration using sensible defaults.

## How it works

`wt` assumes you use git worktrees organized as:

```
~/dev/
├── repo-name/
│   ├── main/          ← primary worktree (.git directory)
│   ├── feature-foo/   ← linked worktree (.git file)
│   └── bugfix-bar/    ← linked worktree (.git file)
└── other-repo/
    └── main/
```

Each worktree gets a tmux session named after its path (e.g. `repo-name_main`). The `go` command switches between tmux sessions instead of `cd`-ing, so each worktree keeps its own shell state.

## Dependencies

- **git** — for worktree operations
- **tmux** — for session management
- **gh** (optional) — for PR listing and fork detection

## License

MIT
