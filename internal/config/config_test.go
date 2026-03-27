package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	home, _ := os.UserHomeDir()

	if cfg.DevRoot != filepath.Join(home, "dev") {
		t.Errorf("DevRoot = %q, want %q", cfg.DevRoot, filepath.Join(home, "dev"))
	}

	if len(cfg.LegacyRoots) != 1 || cfg.LegacyRoots[0] != filepath.Join(home, "code") {
		t.Errorf("LegacyRoots = %v, want [%s]", cfg.LegacyRoots, filepath.Join(home, "code"))
	}

	if len(cfg.SpecialDirs) != 1 || cfg.SpecialDirs[0] != filepath.Join(home, ".config", "fleet") {
		t.Errorf("SpecialDirs = %v, want [%s]", cfg.SpecialDirs, filepath.Join(home, ".config", "fleet"))
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"~/dev", filepath.Join(home, "dev")},
		{"/absolute/path", "/absolute/path"},
		{"", ""},
	}

	for _, tt := range tests {
		got := expandHome(tt.input)
		if got != tt.want {
			t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDefaultProviders(t *testing.T) {
	cfg := Default()

	if cfg.DefaultProvider != "github" {
		t.Errorf("DefaultProvider = %q, want %q", cfg.DefaultProvider, "github")
	}

	gh, ok := cfg.Providers["github"]
	if !ok {
		t.Fatal("built-in github provider missing")
	}
	if gh.CloneURL != "git@github.com:{{.Owner}}/{{.Repo}}.git" {
		t.Errorf("github CloneURL = %q", gh.CloneURL)
	}
	if gh.UpstreamURL != "git@github.com:{{.Owner}}/{{.Repo}}.git" {
		t.Errorf("github UpstreamURL = %q", gh.UpstreamURL)
	}
}

func TestParseRepoRef(t *testing.T) {
	cfg := Default()

	tests := []struct {
		name     string
		input    string
		wantProv string
		wantOwn  string
		wantRepo string
		wantURL  bool
		wantErr  bool
	}{
		{
			name:     "plain owner/repo uses default provider",
			input:    "owner/repo",
			wantProv: "github",
			wantOwn:  "owner",
			wantRepo: "repo",
		},
		{
			name:     "provider:owner/repo",
			input:    "gitlab:owner/repo",
			wantProv: "gitlab",
			wantOwn:  "owner",
			wantRepo: "repo",
		},
		{
			name:     "explicit github:owner/repo",
			input:    "github:myorg/myrepo",
			wantProv: "github",
			wantOwn:  "myorg",
			wantRepo: "myrepo",
		},
		{
			name:     "strips .git suffix",
			input:    "owner/repo.git",
			wantProv: "github",
			wantOwn:  "owner",
			wantRepo: "repo",
		},
		{
			name:    "SSH URL is raw",
			input:   "git@github.com:owner/repo.git",
			wantURL: true,
		},
		{
			name:    "HTTPS URL is raw",
			input:   "https://github.com/owner/repo.git",
			wantURL: true,
		},
		{
			name:    "empty input errors",
			input:   "",
			wantErr: true,
		},
		{
			name:    "bare word errors",
			input:   "justrepo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := cfg.ParseRepoRef(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got %+v", ref)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantURL {
				if !ref.IsURL {
					t.Errorf("expected IsURL=true")
				}
				if ref.RawURL != tt.input {
					t.Errorf("RawURL = %q, want %q", ref.RawURL, tt.input)
				}
				return
			}

			if ref.Provider != tt.wantProv {
				t.Errorf("Provider = %q, want %q", ref.Provider, tt.wantProv)
			}
			if ref.Owner != tt.wantOwn {
				t.Errorf("Owner = %q, want %q", ref.Owner, tt.wantOwn)
			}
			if ref.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", ref.Repo, tt.wantRepo)
			}
		})
	}
}

func TestResolveCloneURL(t *testing.T) {
	cfg := Default()
	cfg.Providers["gitlab"] = Provider{
		CloneURL: "git@gitlab.com:{{.Owner}}/{{.Repo}}.git",
	}

	tests := []struct {
		name     string
		provider string
		owner    string
		repo     string
		want     string
		wantErr  bool
	}{
		{
			name:     "github default",
			provider: "github",
			owner:    "myorg",
			repo:     "myrepo",
			want:     "git@github.com:myorg/myrepo.git",
		},
		{
			name:     "gitlab custom",
			provider: "gitlab",
			owner:    "team",
			repo:     "project",
			want:     "git@gitlab.com:team/project.git",
		},
		{
			name:     "unknown provider errors",
			provider: "unknown",
			owner:    "a",
			repo:     "b",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cfg.ResolveCloneURL(tt.provider, tt.owner, tt.repo)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveUpstreamURL_FallsBackToCloneURL(t *testing.T) {
	cfg := Default()
	cfg.Providers["gitlab"] = Provider{
		CloneURL: "git@gitlab.com:{{.Owner}}/{{.Repo}}.git",
		// UpstreamURL intentionally empty
	}

	got, err := cfg.ResolveUpstreamURL("gitlab", "team", "project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "git@gitlab.com:team/project.git"
	if got != want {
		t.Errorf("got %q, want %q (should fall back to CloneURL)", got, want)
	}
}

func TestResolveUpstreamURL_UsesOwnTemplate(t *testing.T) {
	cfg := Default()
	cfg.Providers["custom"] = Provider{
		CloneURL:    "git@git.example.com:{{.Owner}}/{{.Repo}}.git",
		UpstreamURL: "https://git.example.com/{{.Owner}}/{{.Repo}}.git",
	}

	got, err := cfg.ResolveUpstreamURL("custom", "org", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://git.example.com/org/repo.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
