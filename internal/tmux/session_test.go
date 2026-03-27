package tmux

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionName(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{
			name: "dev root repo worktree",
			dir:  filepath.Join(home, "dev", "myrepo", "main"),
			want: "myrepo_main",
		},
		{
			name: "dev root repo with nested worktree",
			dir:  filepath.Join(home, "dev", "myrepo", "feature-branch"),
			want: "myrepo_feature-branch",
		},
		{
			name: "code root repo worktree",
			dir:  filepath.Join(home, "code", "owner", "repo"),
			want: "owner_repo",
		},
		{
			name: "fleet config dir",
			dir:  filepath.Join(home, ".config", "fleet"),
			want: "fleet",
		},
		{
			name: "arbitrary nested dir",
			dir:  "/tmp/projects/myapp",
			want: "projects.myapp",
		},
		{
			name: "home child dir",
			dir:  filepath.Join(home, "Downloads"),
			want: "Downloads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SessionName(tt.dir)
			if got != tt.want {
				t.Errorf("SessionName(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}
