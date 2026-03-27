package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/analogrelay/wt/internal/config"
	gitpkg "github.com/analogrelay/wt/internal/git"
	"github.com/analogrelay/wt/internal/github"
	"github.com/analogrelay/wt/internal/tmux"
	"github.com/analogrelay/wt/internal/tui"
	"github.com/analogrelay/wt/internal/workspace"
	"github.com/spf13/cobra"
)

var goCmd = &cobra.Command{
	Use:   "go [query]",
	Short: "Navigate to a project directory",
	Long: `Navigate to a project directory using a fuzzy picker.

Select from existing worktrees, switch tmux sessions, or
create new worktrees with the interactive flow.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGo,
}

func init() {
	rootCmd.AddCommand(goCmd)
}

func runGo(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	// Build candidate list
	candidates := workspace.ScanCandidates(cfg)

	// Build picker items
	var items []tui.PickerItem
	for _, c := range candidates {
		annotation := ""
		if c.HasSession {
			annotation = tui.IndicatorSession
		}
		items = append(items, tui.PickerItem{
			Text:       c.Path,
			Annotation: annotation,
			Value:      c,
		})
	}

	// Add synthetic entry
	items = append(items, tui.PickerItem{
		Text:  "[+] new worktree",
		Value: "new-worktree",
	})

	selected, err := tui.RunPicker(items, "go> ", query)
	if err != nil {
		return err
	}
	if selected == nil {
		return nil
	}

	// Dispatch
	switch v := selected.Value.(type) {
	case string:
		if v == "new-worktree" {
			return newWorktreeFlow(cfg)
		}
	case workspace.Candidate:
		return tmux.Switch(v.Path)
	}

	// Fallback: treat text as path
	return tmux.Switch(selected.Text)
}

func newWorktreeFlow(cfg config.Config) error {
	// Step 1: Repo picker
	repoDir := cfg.DevRoot
	var repoCandidates []tui.PickerItem

	if entries, err := os.ReadDir(repoDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				repoCandidates = append(repoCandidates, tui.PickerItem{
					Text:  e.Name(),
					Value: e.Name(),
				})
			}
		}
	}

	repoPicker := tui.NewPicker(repoCandidates, "repo> ", "")
	repoPicker.SyntheticEntry = func(query string) *tui.PickerItem {
		return &tui.PickerItem{
			Text:  fmt.Sprintf("<new repo: %s>", query),
			Value: "new-repo:" + query,
		}
	}
	repoSelected, err := tui.RunPickerModel(repoPicker)
	if err != nil {
		return err
	}
	if repoSelected == nil {
		return nil
	}

	// Handle new repo creation
	if val, ok := repoSelected.Value.(string); ok && strings.HasPrefix(val, "new-repo:") {
		repoName := strings.TrimPrefix(val, "new-repo:")
		dest := filepath.Join(cfg.DevRoot, repoName, "main")

		if info, err := os.Stat(dest); err == nil && info.IsDir() {
			fmt.Fprintf(os.Stderr, "wt: %s already exists, switching to it\n", dest)
			return tmux.Switch(dest)
		}

		fmt.Fprintf(os.Stderr, "wt: creating new repo %s\n", repoName)
		if err := gitpkg.Init(dest); err != nil {
			return fmt.Errorf("git init failed: %w", err)
		}
		return tmux.Switch(dest)
	}

	repoName := repoSelected.Text

	// Check if it matches an existing repo dir
	if _, err := os.Stat(filepath.Join(repoDir, repoName)); os.IsNotExist(err) {
		// Not an existing repo — treat as owner/repo or URL, clone it
		var clonedName string
		clonedName, err = ensureRepo(cfg, repoName)
		if err != nil {
			return err
		}
		repoName = clonedName
	}

	primaryWT := gitpkg.PrimaryWorktree(cfg.DevRoot, repoName)
	if primaryWT == "" {
		return fmt.Errorf("cannot find primary worktree for %s", repoName)
	}

	// Step 2: Branch picker
	sp := tui.StartSpinner("Collecting branches")
	gitpkg.FetchAll(primaryWT) //nolint:errcheck

	// Collect branches with provenance
	type branchInfo struct {
		sources []string
		prInfo  string
	}
	branches := make(map[string]*branchInfo)

	for _, b := range gitpkg.LocalBranches(primaryWT) {
		if branches[b] == nil {
			branches[b] = &branchInfo{}
		}
		branches[b].sources = append(branches[b].sources, "local")
	}

	for _, ref := range gitpkg.RemoteBranches(primaryWT) {
		if strings.HasSuffix(ref, "/HEAD") {
			continue
		}
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			continue
		}
		remoteName, branchName := parts[0], parts[1]
		if branches[branchName] == nil {
			branches[branchName] = &branchInfo{}
		}
		branches[branchName].sources = append(branches[branchName].sources, remoteName)
	}

	// Fetch PRs
	prRemote := gitpkg.UpstreamRemote(primaryWT)
	prSlug := gitpkg.RepoSlug(primaryWT, prRemote)
	branchPRs := make(map[string]string)
	if prSlug != "" {
		if prs, err := github.ListPRs(prSlug); err == nil {
			for _, pr := range prs {
				branchPRs[pr.HeadRefName] = fmt.Sprintf("PR #%d: %s", pr.Number, pr.Title)
			}
		}
	}
	sp.Stop()

	// Build display items
	sortedBranches := make([]string, 0, len(branches))
	for b := range branches {
		sortedBranches = append(sortedBranches, b)
	}
	sort.Strings(sortedBranches)

	var branchItems []tui.PickerItem
	for _, bname := range sortedBranches {
		info := branches[bname]
		meta := strings.Join(info.sources, ",")
		if prInfo, ok := branchPRs[bname]; ok {
			meta += ", " + prInfo
			info.prInfo = prInfo
		}
		branchItems = append(branchItems, tui.PickerItem{
			Text:       bname,
			Annotation: fmt.Sprintf("(%s)", meta),
			Value:      bname,
		})
	}

	branchItems = append(branchItems, tui.PickerItem{
		Text:  "[+] new branch",
		Value: "+new",
	})

	branchSelected, err := tui.RunPicker(branchItems, "branch> ", "")
	if err != nil {
		return err
	}
	if branchSelected == nil {
		return nil
	}

	selectedBranch := branchSelected.Text
	selectedValue, _ := branchSelected.Value.(string)

	// Determine source info for the selected branch
	var selectedSources string
	if info, ok := branches[selectedBranch]; ok {
		selectedSources = strings.Join(info.sources, ",")
	}

	// Step 3: New branch flow
	if selectedValue == "+new" || selectedSources == "" {
		newBranchName := selectedBranch
		if selectedValue == "+new" {
			newBranchName = tui.Prompt("New branch name: ")
			if newBranchName == "" {
				return nil
			}
		}

		// Base branch picker — prioritize main/master/release/*
		var basePriority, baseOther []tui.PickerItem
		for _, bname := range sortedBranches {
			item := tui.PickerItem{Text: bname, Value: bname}
			switch {
			case bname == "main" || bname == "master" ||
				strings.HasPrefix(bname, "release/") ||
				strings.HasPrefix(bname, "release-"):
				basePriority = append(basePriority, item)
			default:
				baseOther = append(baseOther, item)
			}
		}
		baseItems := append(basePriority, baseOther...)

		baseSelected, err := tui.RunPicker(baseItems, "base branch> ", "")
		if err != nil {
			return err
		}
		if baseSelected == nil {
			return nil
		}

		baseBranch := baseSelected.Text

		// Resolve base to ref (prefer remote tracking)
		baseRef := baseBranch
		if info, ok := branches[baseBranch]; ok {
			srcs := strings.Join(info.sources, ",")
			if strings.Contains(srcs, "upstream") {
				baseRef = "upstream/" + baseBranch
			} else if strings.Contains(srcs, "origin") {
				baseRef = "origin/" + baseBranch
			}
		}

		worktreeDir := gitpkg.BranchToDir(newBranchName)
		dest := filepath.Join(cfg.DevRoot, repoName, worktreeDir)

		if info, err := os.Stat(dest); err == nil && info.IsDir() {
			fmt.Fprintf(os.Stderr, "wt: %s already exists, switching to it\n", dest)
			return tmux.Switch(dest)
		}

		fmt.Fprintf(os.Stderr, "wt: creating branch %s from %s\n", newBranchName, baseBranch)
		if err := gitpkg.WorktreeAddNewBranch(primaryWT, dest, newBranchName, baseRef); err != nil {
			return err
		}
		return tmux.Switch(dest)
	}

	// Create worktree for existing branch
	worktreeDir := gitpkg.BranchToDir(selectedBranch)
	dest := filepath.Join(cfg.DevRoot, repoName, worktreeDir)

	if info, err := os.Stat(dest); err == nil && info.IsDir() {
		fmt.Fprintf(os.Stderr, "wt: %s already exists, switching to it\n", dest)
		return tmux.Switch(dest)
	}

	hasPR := branchPRs[selectedBranch]

	if hasPR != "" &&
		!strings.Contains(selectedSources, "local") &&
		!strings.Contains(selectedSources, "origin") {
		// PR from upstream only — fetch the PR ref
		prNumStr := strings.TrimPrefix(hasPR, "PR #")
		prNumStr = strings.SplitN(prNumStr, ":", 2)[0]
		prNum := 0
		fmt.Sscanf(prNumStr, "%d", &prNum) //nolint:errcheck

		fmt.Fprintf(os.Stderr, "wt: fetching %s → %s\n", hasPR, dest)
		if err := gitpkg.FetchPR(primaryWT, prRemote, prNum, selectedBranch); err != nil {
			return err
		}
		if err := gitpkg.WorktreeAdd(primaryWT, dest, selectedBranch, false); err != nil {
			return err
		}
	} else if strings.Contains(selectedSources, "upstream") &&
		!strings.Contains(selectedSources, "local") {
		fmt.Fprintf(os.Stderr, "wt: adding worktree for upstream branch %s\n", selectedBranch)
		if err := gitpkg.WorktreeAddNewBranch(primaryWT, dest, selectedBranch, "upstream/"+selectedBranch); err != nil {
			return err
		}
	} else if strings.Contains(selectedSources, "origin") &&
		!strings.Contains(selectedSources, "local") {
		fmt.Fprintf(os.Stderr, "wt: adding worktree for origin/%s\n", selectedBranch)
		if err := gitpkg.WorktreeAdd(primaryWT, dest, "origin/"+selectedBranch, false); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(os.Stderr, "wt: adding worktree for %s\n", selectedBranch)
		if err := gitpkg.WorktreeAdd(primaryWT, dest, selectedBranch, false); err != nil {
			return err
		}
	}

	return tmux.Switch(dest)
}

// ensureRepo clones a repo into ~/dev if not already present.
// Returns the repo directory name.
func ensureRepo(cfg config.Config, input string) (string, error) {
	ref, err := cfg.ParseRepoRef(input)
	if err != nil {
		return "", err
	}

	var slug, cloneURL string
	var owner, repo, provider string

	if ref.IsURL {
		// Full URL — extract slug for naming, use URL directly
		slug = gitpkg.RemoteSlug(ref.RawURL)
		if slug == "" {
			return "", fmt.Errorf("could not parse repo slug from %q", ref.RawURL)
		}
		cloneURL = ref.RawURL
		parts := strings.SplitN(slug, "/", 2)
		owner, repo = parts[0], parts[1]
		provider = "" // no provider for raw URLs
	} else {
		owner = ref.Owner
		repo = ref.Repo
		provider = ref.Provider
		slug = owner + "/" + repo
		cloneURL, err = cfg.ResolveCloneURL(provider, owner, repo)
		if err != nil {
			return "", err
		}
	}

	destName := repo
	destDir := filepath.Join(cfg.DevRoot, destName)

	// Collision check
	if info, err := os.Stat(destDir); err == nil && info.IsDir() {
		primaryWT := gitpkg.PrimaryWorktree(cfg.DevRoot, destName)
		if primaryWT != "" {
			existingSlug := gitpkg.RepoSlug(primaryWT, "origin")
			if existingSlug == slug {
				return destName, nil // already cloned
			}
		}

		// Different repo — ask to use owner.repo
		altName := owner + "." + repo
		fmt.Fprintf(os.Stderr, "wt: ~/dev/%s already exists for a different repo.\n", destName)
		if tui.Confirm(fmt.Sprintf("Use %s instead?", altName)) {
			destName = altName
			destDir = filepath.Join(cfg.DevRoot, destName)
		} else {
			return "", fmt.Errorf("cancelled")
		}
	}

	cloneDest := filepath.Join(destDir, "main")
	fmt.Fprintf(os.Stderr, "wt: cloning %s → ~/dev/%s/main ...\n", slug, destName)
	if err := gitpkg.Clone(cloneURL, cloneDest); err != nil {
		return "", fmt.Errorf("clone failed: %w", err)
	}

	// Detect default branch
	defaultBranch := gitpkg.DefaultBranch(cloneDest)

	// Rename clone dir to match default branch if needed
	if defaultBranch != "main" {
		newDest := filepath.Join(destDir, defaultBranch)
		if err := os.Rename(cloneDest, newDest); err != nil {
			return "", fmt.Errorf("rename failed: %w", err)
		}
	}

	// Fork detection — only for github provider (gh CLI only works with GitHub)
	if provider == "github" {
		primaryWT := gitpkg.PrimaryWorktree(cfg.DevRoot, destName)
		if primaryWT != "" {
			info, err := github.ViewRepo(slug)
			if err == nil && info.IsFork {
				parentSlug := github.ParentSlug(info)
				if parentSlug != "" {
					fmt.Fprintf(os.Stderr, "wt: %s is a fork of %s\n", slug, parentSlug)
					if tui.ConfirmDefault(fmt.Sprintf("Add %s as 'upstream' remote?", parentSlug)) {
						// Resolve upstream URL through the provider
						parentParts := strings.SplitN(parentSlug, "/", 2)
						upstreamURL, urlErr := cfg.ResolveUpstreamURL(provider, parentParts[0], parentParts[1])
						if urlErr != nil {
							upstreamURL = fmt.Sprintf("git@github.com:%s.git", parentSlug)
						}
						gitpkg.AddRemote(primaryWT, "upstream", upstreamURL) //nolint:errcheck
						gitpkg.Fetch(primaryWT, "upstream")                  //nolint:errcheck
						fmt.Fprintf(os.Stderr, "wt: added upstream → %s\n", parentSlug)
					}
				}
			}
		}
	}

	return destName, nil
}
