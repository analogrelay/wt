package git

import "testing"

func TestRemoteSlug(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "SSH URL",
			url:  "git@github.com:owner/repo.git",
			want: "owner/repo",
		},
		{
			name: "SSH URL without .git",
			url:  "git@github.com:owner/repo",
			want: "owner/repo",
		},
		{
			name: "HTTPS URL",
			url:  "https://github.com/owner/repo.git",
			want: "owner/repo",
		},
		{
			name: "HTTPS URL without .git",
			url:  "https://github.com/owner/repo",
			want: "owner/repo",
		},
		{
			name: "HTTPS URL with trailing slash",
			url:  "https://github.com/owner/repo/",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoteSlug(tt.url)
			if got != tt.want {
				t.Errorf("RemoteSlug(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
