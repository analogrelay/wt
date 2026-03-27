package workspace

import (
	"os"
	"path/filepath"

	"github.com/analogrelay/wt/internal/config"
	"github.com/analogrelay/wt/internal/tmux"
)

// Candidate represents a navigable directory.
type Candidate struct {
	Path        string
	SessionName string
	HasSession  bool
}

// ScanCandidates collects all candidate directories from configured roots.
func ScanCandidates(cfg config.Config) []Candidate {
	activeSessions := make(map[string]bool)
	for _, s := range tmux.ListSessions() {
		activeSessions[s] = true
	}

	var candidates []Candidate

	// Dev root: depth-2 directories (~/dev/*/*)
	candidates = appendDepth2(candidates, cfg.DevRoot, activeSessions)

	// Legacy roots: depth-2 directories
	for _, root := range cfg.LegacyRoots {
		candidates = appendDepth2(candidates, root, activeSessions)
	}

	// Special directories
	for _, sd := range cfg.SpecialDirs {
		if info, err := os.Stat(sd); err == nil && info.IsDir() {
			sname := tmux.SessionName(sd)
			candidates = append(candidates, Candidate{
				Path:        sd,
				SessionName: sname,
				HasSession:  activeSessions[sname],
			})
		}
	}

	// Dev root and legacy roots themselves as navigable targets
	for _, root := range append([]string{cfg.DevRoot}, cfg.LegacyRoots...) {
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			sname := tmux.SessionName(root)
			candidates = append(candidates, Candidate{
				Path:        root,
				SessionName: sname,
				HasSession:  activeSessions[sname],
			})
		}
	}

	// macOS extra dirs
	for _, d := range config.MacOSDirs() {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			sname := tmux.SessionName(d)
			candidates = append(candidates, Candidate{
				Path:        d,
				SessionName: sname,
				HasSession:  activeSessions[sname],
			})
		}
	}

	return candidates
}

func appendDepth2(candidates []Candidate, root string, activeSessions map[string]bool) []Candidate {
	if root == "" {
		return candidates
	}
	repos, err := os.ReadDir(root)
	if err != nil {
		return candidates
	}
	for _, repo := range repos {
		if !repo.IsDir() {
			continue
		}
		repoPath := filepath.Join(root, repo.Name())
		wts, err := os.ReadDir(repoPath)
		if err != nil {
			continue
		}
		for _, wt := range wts {
			if !wt.IsDir() {
				continue
			}
			wtPath := filepath.Join(repoPath, wt.Name())
			sname := tmux.SessionName(wtPath)
			candidates = append(candidates, Candidate{
				Path:        wtPath,
				SessionName: sname,
				HasSession:  activeSessions[sname],
			})
		}
	}
	return candidates
}
