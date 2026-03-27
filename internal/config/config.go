package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
)

// Provider defines a git hosting provider with URL templates.
// Templates use Go text/template syntax with {{.Owner}} and {{.Repo}} fields.
type Provider struct {
	CloneURL    string `toml:"clone_url"`
	UpstreamURL string `toml:"upstream_url"`
}

type Config struct {
	DevRoot              string              `toml:"dev_root"`
	LegacyRoots          []string            `toml:"legacy_roots"`
	SpecialDirs          []string            `toml:"special_dirs"`
	SessionNameOverrides map[string]string   `toml:"session_name_overrides"`
	Providers            map[string]Provider `toml:"providers"`
	DefaultProvider      string              `toml:"default_provider"`
}

func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DevRoot:              filepath.Join(home, "dev"),
		LegacyRoots:          []string{filepath.Join(home, "code")},
		SpecialDirs:          []string{filepath.Join(home, ".config", "fleet")},
		SessionNameOverrides: map[string]string{},
		DefaultProvider:      "github",
		Providers: map[string]Provider{
			"github": {
				CloneURL:    "git@github.com:{{.Owner}}/{{.Repo}}.git",
				UpstreamURL: "git@github.com:{{.Owner}}/{{.Repo}}.git",
			},
		},
	}
}

func Load() (Config, error) {
	cfg := Default()

	path := configPath()
	if path == "" {
		return cfg, nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	// Save built-in providers before decoding (TOML decode replaces the map)
	builtinProviders := cfg.Providers

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, err
	}

	// Merge: built-in providers are preserved unless explicitly overridden
	if cfg.Providers == nil {
		cfg.Providers = builtinProviders
	} else {
		for name, p := range builtinProviders {
			if _, exists := cfg.Providers[name]; !exists {
				cfg.Providers[name] = p
			}
		}
	}

	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = "github"
	}

	// Expand ~ in paths
	cfg.DevRoot = expandHome(cfg.DevRoot)
	for i, r := range cfg.LegacyRoots {
		cfg.LegacyRoots[i] = expandHome(r)
	}
	for i, d := range cfg.SpecialDirs {
		cfg.SpecialDirs[i] = expandHome(d)
	}

	return cfg, nil
}

// MacOSDirs returns extra candidate directories on macOS.
func MacOSDirs() []string {
	if runtime.GOOS != "darwin" {
		return nil
	}
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Documents"),
	}
}

// repoTemplateData holds the fields available to provider URL templates.
type repoTemplateData struct {
	Owner string
	Repo  string
}

// ResolveCloneURL executes the named provider's clone_url template.
func (c *Config) ResolveCloneURL(provider, owner, repo string) (string, error) {
	p, ok := c.Providers[provider]
	if !ok {
		return "", fmt.Errorf("unknown provider: %q", provider)
	}
	return executeTemplate(p.CloneURL, owner, repo)
}

// ResolveUpstreamURL executes the named provider's upstream_url template.
// Falls back to clone_url if upstream_url is not set.
func (c *Config) ResolveUpstreamURL(provider, owner, repo string) (string, error) {
	p, ok := c.Providers[provider]
	if !ok {
		return "", fmt.Errorf("unknown provider: %q", provider)
	}
	tmplStr := p.UpstreamURL
	if tmplStr == "" {
		tmplStr = p.CloneURL
	}
	return executeTemplate(tmplStr, owner, repo)
}

func executeTemplate(tmplStr, owner, repo string) (string, error) {
	t, err := template.New("url").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL template %q: %w", tmplStr, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, repoTemplateData{Owner: owner, Repo: repo}); err != nil {
		return "", fmt.Errorf("executing URL template: %w", err)
	}
	return buf.String(), nil
}

// RepoRef holds the parsed components of a [provider:]owner/repo input.
type RepoRef struct {
	Provider string
	Owner    string
	Repo     string
	IsURL    bool   // true if the input was a full URL (bypass provider resolution)
	RawURL   string // the original URL when IsURL is true
}

// ParseRepoRef parses user input into a RepoRef.
//
// Accepted formats:
//   - owner/repo           → default provider
//   - provider:owner/repo  → named provider
//   - git@host:owner/repo  → raw URL (IsURL=true)
//   - https://host/owner/repo → raw URL (IsURL=true)
func (c *Config) ParseRepoRef(input string) (RepoRef, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return RepoRef{}, fmt.Errorf("empty input")
	}

	// Full URL detection: contains :// or starts with git@
	if strings.Contains(input, "://") || strings.HasPrefix(input, "git@") {
		return RepoRef{IsURL: true, RawURL: input}, nil
	}

	// Check for provider:owner/repo (but not a bare owner/repo with colon in owner)
	if idx := strings.Index(input, ":"); idx > 0 && !strings.Contains(input[:idx], "/") {
		provider := input[:idx]
		rest := input[idx+1:]
		owner, repo, err := splitSlug(rest)
		if err != nil {
			return RepoRef{}, fmt.Errorf("invalid repo reference %q: %w", input, err)
		}
		return RepoRef{Provider: provider, Owner: owner, Repo: repo}, nil
	}

	// Plain owner/repo
	owner, repo, err := splitSlug(input)
	if err != nil {
		return RepoRef{}, fmt.Errorf("invalid repo reference %q: %w", input, err)
	}
	return RepoRef{Provider: c.DefaultProvider, Owner: owner, Repo: repo}, nil
}

func splitSlug(s string) (owner, repo string, err error) {
	s = strings.TrimSuffix(s, ".git")
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected owner/repo, got %q", s)
	}
	return parts[0], parts[1], nil
}

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "wt", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "wt", "config.toml")
}

func expandHome(path string) string {
	if len(path) == 0 {
		return path
	}
	if path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
