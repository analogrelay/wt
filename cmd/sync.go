package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	gosync "sync"

	"github.com/analogrelay/wt/internal/config"
	gitpkg "github.com/analogrelay/wt/internal/git"
	"github.com/analogrelay/wt/internal/tui"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Safely pull/push all worktrees",
	Long: `Fetch all remotes, analyze sync status, and offer to pull/push.

Protected branches (main, master, release/*) are never auto-pushed.
Dirty or diverged worktrees are reported but skipped.`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

type syncAction string

const (
	actionOK      syncAction = "ok"
	actionPull    syncAction = "pull"
	actionPush    syncAction = "push"
	actionProtect syncAction = "protect"
	actionDiverge syncAction = "diverge"
	actionDirty   syncAction = "dirty"
)

type syncEntry struct {
	action syncAction
	dir    string
	label  string
	detail string
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, err := os.Stat(cfg.DevRoot); os.IsNotExist(err) {
		return fmt.Errorf("~/dev does not exist")
	}

	// Phase 1: Collect worktree directories
	dirs := gitpkg.ListWorktreeDirs(cfg.DevRoot, cfg.SpecialDirs)

	if len(dirs) == 0 {
		fmt.Println("wt: no worktrees found")
		return nil
	}

	// Phase 2: Fetch once per repo in parallel
	sp := tui.StartSpinner("Fetching remotes")
	gitpkg.ParallelFetchRepos(dirs, cfg.DevRoot, cfg.SpecialDirs)
	sp.Stop()

	// Phase 3: Analyze each worktree in parallel
	entries := make([]syncEntry, len(dirs))
	var wg gosync.WaitGroup
	wg.Add(len(dirs))

	for i, d := range dirs {
		go func(idx int, dir string) {
			defer wg.Done()
			entries[idx] = analyzeSyncEntry(cfg, dir)
		}(i, d)
	}
	wg.Wait()

	// Filter out zero-value entries (detached HEAD worktrees)
	var plan []syncEntry
	for _, e := range entries {
		if e.action != "" {
			plan = append(plan, e)
		}
	}

	// Phase 4: Present sync plan
	if len(plan) == 0 {
		fmt.Println("wt: no worktrees to sync")
		return nil
	}

	hasActions := false
	fmt.Println()
	fmt.Println("wt sync plan:")
	fmt.Println()

	for _, e := range plan {
		switch e.action {
		case actionOK:
			fmt.Printf("  ✓ ok      %-35s (%s)\n", e.label, e.detail)
		case actionPull:
			fmt.Printf("  ↓ pull    %-35s (%s)\n", e.label, e.detail)
			hasActions = true
		case actionPush:
			fmt.Printf("  ↑ push    %-35s (%s)\n", e.label, e.detail)
			hasActions = true
		case actionProtect:
			fmt.Printf("  ⚠ protect %-35s (%s)\n", e.label, e.detail)
		case actionDiverge:
			fmt.Printf("  ⚠ diverge %-35s (%s)\n", e.label, e.detail)
		case actionDirty:
			fmt.Printf("  ⚠ dirty   %-35s (%s)\n", e.label, e.detail)
		}
	}

	if !hasActions {
		fmt.Println()
		fmt.Println("wt: nothing to sync")
		return nil
	}

	fmt.Println()
	if !tui.Confirm("Proceed with pull/push actions?") {
		return nil
	}

	// Phase 5: Execute pull/push in parallel
	fmt.Println()

	type syncResult struct {
		label   string
		action  syncAction
		success bool
	}

	var actionEntries []syncEntry
	for _, e := range plan {
		if e.action == actionPull || e.action == actionPush {
			actionEntries = append(actionEntries, e)
		}
	}

	results := make([]syncResult, len(actionEntries))
	var execWg gosync.WaitGroup
	execWg.Add(len(actionEntries))

	for i, e := range actionEntries {
		go func(idx int, entry syncEntry) {
			defer execWg.Done()
			var err error
			switch entry.action {
			case actionPull:
				err = gitpkg.Pull(entry.dir)
			case actionPush:
				err = gitpkg.Push(entry.dir)
			}
			results[idx] = syncResult{
				label:   entry.label,
				action:  entry.action,
				success: err == nil,
			}
		}(i, e)
	}
	execWg.Wait()

	for _, r := range results {
		if r.success {
			verb := "pulled"
			if r.action == actionPush {
				verb = "pushed"
			}
			fmt.Printf("  ✓ %-7s %s\n", verb, r.label)
		} else {
			verb := "pull --ff-only"
			if r.action == actionPush {
				verb = "push"
			}
			fmt.Fprintf(os.Stderr, "  ✗ failed  %s (%s failed)\n", r.label, verb)
		}
	}

	fmt.Println()
	fmt.Println("wt: sync complete")
	return nil
}

func analyzeSyncEntry(cfg config.Config, dir string) syncEntry {
	isSpecial := false
	for _, sd := range cfg.SpecialDirs {
		if dir == sd {
			isSpecial = true
			break
		}
	}

	var label string
	if isSpecial {
		label = filepath.Base(dir)
	} else {
		repoName := filepath.Base(filepath.Dir(dir))
		wtName := filepath.Base(dir)
		label = repoName + "/" + wtName
	}

	branch := gitpkg.CurrentBranch(dir)
	if branch == "" {
		// Detached HEAD — skip
		return syncEntry{}
	}

	dirty := gitpkg.DirtyCount(dir)
	if dirty > 0 {
		return syncEntry{
			action: actionDirty,
			dir:    dir,
			label:  label,
			detail: fmt.Sprintf("%d uncommitted changes", dirty),
		}
	}

	remote := gitpkg.UpstreamRemote(dir)
	ahead, behind := gitpkg.AheadBehind(dir)
	if ahead == -1 {
		// No tracking branch
		return syncEntry{}
	}

	if ahead == 0 && behind == 0 {
		return syncEntry{action: actionOK, dir: dir, label: label, detail: "in sync"}
	}

	if ahead == 0 && behind > 0 {
		return syncEntry{
			action: actionPull,
			dir:    dir,
			label:  label,
			detail: fmt.Sprintf("%d behind %s", behind, remote),
		}
	}

	if ahead > 0 && behind == 0 {
		if isProtectedBranch(branch) {
			return syncEntry{
				action: actionProtect,
				dir:    dir,
				label:  label,
				detail: fmt.Sprintf("%d ahead of %s — push manually", ahead, remote),
			}
		}
		return syncEntry{
			action: actionPush,
			dir:    dir,
			label:  label,
			detail: fmt.Sprintf("%d ahead of %s", ahead, remote),
		}
	}

	return syncEntry{
		action: actionDiverge,
		dir:    dir,
		label:  label,
		detail: fmt.Sprintf("%d ahead, %d behind — needs manual rebase/merge", ahead, behind),
	}
}

func isProtectedBranch(branch string) bool {
	switch {
	case branch == "main" || branch == "master":
		return true
	case strings.HasPrefix(branch, "release/") || strings.HasPrefix(branch, "release-"):
		return true
	default:
		return false
	}
}
