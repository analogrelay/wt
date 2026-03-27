package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PrimaryWorktree returns the path of the primary worktree (the one with a .git
// directory, not a .git file) for a repo under devRoot.
func PrimaryWorktree(devRoot, repoName string) string {
	repoDir := filepath.Join(devRoot, repoName)

	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return ""
	}

	// First pass: find the directory with a .git directory (primary worktree)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(repoDir, e.Name())
		gitPath := filepath.Join(candidate, ".git")
		info, err := os.Stat(gitPath)
		if err == nil && info.IsDir() {
			return candidate
		}
	}

	// Fallback: return the first directory
	for _, e := range entries {
		if e.IsDir() {
			return filepath.Join(repoDir, e.Name())
		}
	}

	return ""
}

// BranchToDir converts a branch name to a worktree directory name.
// Slashes are replaced with hyphens.
func BranchToDir(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

// WorktreeAdd creates a new worktree at dest for the given branch.
// If newBranch is true, it creates a new branch with -b.
func WorktreeAdd(primaryWT, dest, ref string, newBranch bool) error {
	args := []string{"-C", primaryWT, "worktree", "add"}
	if newBranch {
		args = append(args, "-b", ref, dest)
	} else {
		args = append(args, dest, ref)
	}
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// WorktreeAddNewBranch creates a new worktree at dest with a new branch based on baseRef.
func WorktreeAddNewBranch(primaryWT, dest, branchName, baseRef string) error {
	cmd := exec.Command("git", "-C", primaryWT, "worktree", "add", "-b", branchName, dest, baseRef)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// WorktreeRemove removes a worktree at the given path.
func WorktreeRemove(primaryWT, worktreePath string) error {
	cmd := exec.Command("git", "-C", primaryWT, "worktree", "remove", worktreePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Clone clones a repo into dest.
func Clone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Init creates a new git repository at dest, creating parent directories as needed.
func Init(dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	cmd := exec.Command("git", "init", dest)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DefaultBranch detects the default branch of a repository by inspecting
// refs/remotes/origin/HEAD.
func DefaultBranch(dir string) string {
	out, err := exec.Command("git", "-C", dir, "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err != nil {
		return "main"
	}
	ref := strings.TrimSpace(string(out))
	ref = strings.TrimPrefix(ref, "refs/remotes/origin/")
	if ref == "" {
		return "main"
	}
	return ref
}

// IsGitDir returns true if the given path contains a .git entry (directory or file).
func IsGitDir(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

// IsPrimaryWorktree returns true if the path has a .git directory (not file).
func IsPrimaryWorktree(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ListWorktreeDirs lists all worktree directories under devRoot (depth 2) and
// any special directories that have a .git entry.
func ListWorktreeDirs(devRoot string, specialDirs []string) []string {
	var dirs []string

	// Scan devRoot/*/* for .git entries
	repos, err := os.ReadDir(devRoot)
	if err == nil {
		for _, repo := range repos {
			if !repo.IsDir() {
				continue
			}
			repoPath := filepath.Join(devRoot, repo.Name())
			wts, err := os.ReadDir(repoPath)
			if err != nil {
				continue
			}
			for _, wt := range wts {
				if !wt.IsDir() {
					continue
				}
				wtPath := filepath.Join(repoPath, wt.Name())
				if IsGitDir(wtPath) {
					dirs = append(dirs, wtPath)
				}
			}
		}
	}

	for _, sd := range specialDirs {
		if IsGitDir(sd) {
			dirs = append(dirs, sd)
		}
	}

	return dirs
}

// FetchPR fetches a PR head ref into a local branch name.
func FetchPR(dir, remote string, prNumber int, localBranch string) error {
	ref := fmt.Sprintf("pull/%d/head:%s", prNumber, localBranch)
	cmd := exec.Command("git", "-C", dir, "fetch", remote, ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
