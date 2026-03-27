package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/analogrelay/wt/internal/config"
	gitpkg "github.com/analogrelay/wt/internal/git"
	"github.com/analogrelay/wt/internal/tmux"
	"github.com/analogrelay/wt/internal/tui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show health overview of all worktrees",
	Long: `Show a table with the status of all worktrees.

Columns: REPO, WORKTREE, BRANCH, CLEAN, SYNC, SESSION

Each worktree is checked for uncommitted changes and
sync status with its upstream tracking branch.`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

type worktreeRow struct {
	repo    string
	wt      string
	branch  string
	clean   string
	sync    string
	session string
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, err := os.Stat(cfg.DevRoot); os.IsNotExist(err) {
		return fmt.Errorf("~/dev does not exist")
	}

	// Collect active tmux sessions
	activeSessions := make(map[string]bool)
	for _, s := range tmux.ListSessions() {
		activeSessions[s] = true
	}

	// Find all worktree directories
	dirs := gitpkg.ListWorktreeDirs(cfg.DevRoot, cfg.SpecialDirs)

	if len(dirs) == 0 {
		fmt.Println("wt: no worktrees found")
		return nil
	}

	// Phase 1: Fetch once per repo in parallel
	sp := tui.StartSpinner(fmt.Sprintf("Scanning %d worktrees", len(dirs)))
	gitpkg.ParallelFetchRepos(dirs, cfg.DevRoot, cfg.SpecialDirs)

	// Phase 2: Compute status per worktree in parallel (no network I/O)
	rows := make([]worktreeRow, len(dirs))
	var wg sync.WaitGroup
	wg.Add(len(dirs))

	for i, d := range dirs {
		go func(idx int, dir string) {
			defer wg.Done()
			rows[idx] = buildStatusRow(cfg, dir, activeSessions)
		}(i, d)
	}
	wg.Wait()
	sp.Stop()

	// Track claimed sessions
	claimedSessions := make(map[string]bool)
	for _, r := range rows {
		if r.session == tui.IndicatorSession {
			sname := tmux.SessionName(findDirForRow(r, dirs, rows))
			claimedSessions[sname] = true
		}
	}

	// Build table rows
	var tableRows [][]string
	for _, r := range rows {
		tableRows = append(tableRows, []string{
			r.repo, r.wt, r.branch, r.clean, r.sync, r.session,
		})
	}

	// Add orphan tmux sessions
	for _, sess := range tmux.ListSessions() {
		if claimedSessions[sess] {
			continue
		}
		orphanLabel := fmt.Sprintf("< Unknown tmux session: %s >", sess)
		tableRows = append(tableRows, []string{
			orphanLabel, "", tui.IndicatorNone, tui.IndicatorNone, tui.IndicatorNone, tui.IndicatorSession,
		})
	}

	fmt.Print(tui.RenderStatusTable(tableRows))
	return nil
}

func buildStatusRow(cfg config.Config, dir string, activeSessions map[string]bool) worktreeRow {
	var row worktreeRow

	// Determine display names
	isSpecial := false
	for _, sd := range cfg.SpecialDirs {
		if dir == sd {
			isSpecial = true
			break
		}
	}

	if isSpecial {
		row.repo = filepath.Base(dir)
		row.wt = tui.IndicatorNone
	} else {
		row.repo = filepath.Base(filepath.Dir(dir))
		row.wt = filepath.Base(dir)
	}

	// Branch
	branch := gitpkg.CurrentBranch(dir)
	if branch == "" {
		sha := gitpkg.ShortHEAD(dir)
		branch = fmt.Sprintf("(%s)", sha)
	}
	row.branch = branch

	// Dirty check
	dirty := gitpkg.DirtyCount(dir)
	if dirty == 0 {
		row.clean = tui.IndicatorClean
	} else {
		row.clean = fmt.Sprintf("%s %d", tui.IndicatorDirty, dirty)
	}

	// Sync check (fetch already done by ParallelFetchRepos)
	ahead, behind := gitpkg.AheadBehind(dir)
	if ahead == -1 {
		row.sync = tui.IndicatorNone
	} else if ahead == 0 && behind == 0 {
		row.sync = tui.IndicatorClean
	} else {
		syncParts := ""
		if ahead > 0 {
			syncParts += fmt.Sprintf("↑%d", ahead)
		}
		if behind > 0 {
			if syncParts != "" {
				syncParts += " "
			}
			syncParts += fmt.Sprintf("↓%d", behind)
		}
		row.sync = syncParts
	}

	// Tmux session
	sname := tmux.SessionName(dir)
	if activeSessions[sname] {
		row.session = tui.IndicatorSession
	} else {
		row.session = tui.IndicatorNone
	}

	return row
}

func findDirForRow(r worktreeRow, dirs []string, rows []worktreeRow) string {
	for i, row := range rows {
		if row == r && i < len(dirs) {
			return dirs[i]
		}
	}
	return ""
}
