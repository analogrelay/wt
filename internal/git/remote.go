package git

import (
	"os/exec"
	"regexp"
	"strings"
)

var slugRe = regexp.MustCompile(`[:/]([^/]+/[^/]+?)(?:\.git)?$`)

// RemoteSlug parses a git remote URL (SSH or HTTPS) into an owner/repo slug.
func RemoteSlug(url string) string {
	m := slugRe.FindStringSubmatch(url)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// RemoteURL returns the URL for the named remote in the given directory.
func RemoteURL(dir, remote string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", remote).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// UpstreamRemote returns "upstream" if that remote exists, otherwise "origin".
func UpstreamRemote(dir string) string {
	if _, err := RemoteURL(dir, "upstream"); err == nil {
		return "upstream"
	}
	return "origin"
}

// RepoSlug returns the owner/repo slug for a given remote in the directory.
func RepoSlug(dir, remote string) string {
	url, err := RemoteURL(dir, remote)
	if err != nil {
		return ""
	}
	return RemoteSlug(url)
}

// FetchAll runs git fetch --all --quiet in the given directory.
func FetchAll(dir string) error {
	cmd := exec.Command("git", "-C", dir, "fetch", "--all", "--quiet")
	return cmd.Run()
}

// Fetch runs git fetch for a specific remote in the given directory.
func Fetch(dir, remote string) error {
	cmd := exec.Command("git", "-C", dir, "fetch", remote, "--quiet")
	return cmd.Run()
}

// Remotes lists all remote names for the directory.
func Remotes(dir string) ([]string, error) {
	out, err := exec.Command("git", "-C", dir, "remote").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}
	return result, nil
}

// AddRemote adds a named remote with the given URL.
func AddRemote(dir, name, url string) error {
	return exec.Command("git", "-C", dir, "remote", "add", name, url).Run()
}
