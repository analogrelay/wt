package github

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// PR represents an open pull request.
type PR struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	HeadRefName string `json:"headRefName"`
}

// ListPRs lists open pull requests for the given repo slug (owner/repo).
func ListPRs(repoSlug string) ([]PR, error) {
	out, err := exec.Command("gh", "pr", "list",
		"--repo", repoSlug,
		"--json", "number,title,headRefName",
	).Output()
	if err != nil {
		return nil, err
	}
	var prs []PR
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

// RepoInfo holds fork detection information.
type RepoInfo struct {
	IsFork bool       `json:"isFork"`
	Parent *ParentRef `json:"parent"`
}

// ParentRef holds the parent repo reference for a fork.
type ParentRef struct {
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
	Name string `json:"name"`
}

// ViewRepo returns fork information for the given repo slug.
func ViewRepo(slug string) (*RepoInfo, error) {
	out, err := exec.Command("gh", "repo", "view", slug,
		"--json", "isFork,parent",
	).Output()
	if err != nil {
		return nil, err
	}
	var info RepoInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ParentSlug returns the owner/repo slug of the parent repo, or empty string.
func ParentSlug(info *RepoInfo) string {
	if info == nil || !info.IsFork || info.Parent == nil {
		return ""
	}
	owner := strings.TrimSpace(info.Parent.Owner.Login)
	name := strings.TrimSpace(info.Parent.Name)
	if owner == "" || name == "" {
		return ""
	}
	return owner + "/" + name
}
