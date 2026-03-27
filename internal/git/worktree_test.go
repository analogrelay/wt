package git

import "testing"

func TestBranchToDir(t *testing.T) {
	tests := []struct {
		branch string
		want   string
	}{
		{"main", "main"},
		{"feature/login", "feature-login"},
		{"release/v1.0", "release-v1.0"},
		{"bugfix/fix-auth/token", "bugfix-fix-auth-token"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := BranchToDir(tt.branch)
			if got != tt.want {
				t.Errorf("BranchToDir(%q) = %q, want %q", tt.branch, got, tt.want)
			}
		})
	}
}
