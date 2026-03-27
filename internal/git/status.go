package git

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// CurrentBranch returns the current branch name, or empty string if detached.
func CurrentBranch(dir string) string {
	out, err := exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ShortHEAD returns the short SHA of HEAD.
func ShortHEAD(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// DirtyCount returns the number of uncommitted changes (from git status --porcelain).
func DirtyCount(dir string) int {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err != nil {
		return 0
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return 0
	}
	return len(strings.Split(text, "\n"))
}

// AheadBehind returns the ahead and behind counts relative to the upstream
// tracking branch. Returns (-1, -1) if there is no tracking branch.
func AheadBehind(dir string) (ahead, behind int) {
	out, err := exec.Command("git", "-C", dir, "rev-list", "--left-right", "--count", "HEAD...@{upstream}").Output()
	if err != nil {
		return -1, -1
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return -1, -1
	}
	a, err1 := strconv.Atoi(parts[0])
	b, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return -1, -1
	}
	return a, b
}

// LocalBranches returns all local branch names.
func LocalBranches(dir string) []string {
	out, err := exec.Command("git", "-C", dir, "branch", "--format=%(refname:short)").Output()
	if err != nil {
		return nil
	}
	return splitLines(string(out))
}

// RemoteBranches returns all remote branch refs (e.g. "origin/main").
func RemoteBranches(dir string) []string {
	out, err := exec.Command("git", "-C", dir, "branch", "-r", "--format=%(refname:short)").Output()
	if err != nil {
		return nil
	}
	return splitLines(string(out))
}

func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	var result []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

// RepoKeyForDir returns the repo identifier for a worktree directory.
// For special dirs it returns the base name; for dev-root dirs it returns the
// parent directory name (the repo directory).
func RepoKeyForDir(dir string, specialDirs []string) string {
	for _, sd := range specialDirs {
		if dir == sd {
			return filepath.Base(dir)
		}
	}
	return filepath.Base(filepath.Dir(dir))
}

// ParallelFetchRepos fetches all remotes once per unique repo, in parallel.
// It deduplicates by repo key so that multiple worktrees sharing the same
// underlying repo only trigger one fetch.
func ParallelFetchRepos(dirs []string, devRoot string, specialDirs []string) {
	// Group dirs by repo key and find a primary worktree (or the dir itself) to fetch from.
	type fetchTarget struct {
		dir string
	}
	targets := make(map[string]fetchTarget)

	for _, d := range dirs {
		key := RepoKeyForDir(d, specialDirs)
		if _, ok := targets[key]; ok {
			continue
		}

		isSpecial := false
		for _, sd := range specialDirs {
			if d == sd {
				isSpecial = true
				break
			}
		}

		if isSpecial {
			targets[key] = fetchTarget{dir: d}
		} else {
			primary := PrimaryWorktree(devRoot, key)
			if primary != "" {
				targets[key] = fetchTarget{dir: primary}
			} else {
				targets[key] = fetchTarget{dir: d}
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(targets))
	for _, t := range targets {
		go func(dir string) {
			defer wg.Done()
			FetchAll(dir) //nolint:errcheck
		}(t.dir)
	}
	wg.Wait()
}

// Pull runs git pull --ff-only in the given directory.
func Pull(dir string) error {
	cmd := exec.Command("git", "-C", dir, "pull", "--ff-only", "--quiet")
	return cmd.Run()
}

// Push runs git push in the given directory.
func Push(dir string) error {
	cmd := exec.Command("git", "-C", dir, "push", "--quiet")
	return cmd.Run()
}
