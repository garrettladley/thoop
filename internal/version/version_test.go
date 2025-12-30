package version

import "testing"

func TestIsNewer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{
			name:    "same version",
			current: "1.0.0",
			latest:  "1.0.0",
			want:    false,
		},
		{
			name:    "same version with v prefix on current",
			current: "v1.0.0",
			latest:  "1.0.0",
			want:    false,
		},
		{
			name:    "same version with v prefix on latest",
			current: "1.0.0",
			latest:  "v1.0.0",
			want:    false,
		},
		{
			name:    "same version with v prefix on both",
			current: "v1.0.0",
			latest:  "v1.0.0",
			want:    false,
		},
		{
			name:    "newer version available",
			current: "1.0.0",
			latest:  "1.1.0",
			want:    true,
		},
		{
			name:    "major version bump",
			current: "1.0.0",
			latest:  "2.0.0",
			want:    true,
		},
		{
			name:    "patch version bump",
			current: "1.0.0",
			latest:  "1.0.1",
			want:    true,
		},
		{
			name:    "devel version never outdated",
			current: "devel",
			latest:  "1.0.0",
			want:    false,
		},
		{
			name:    "unknown version never outdated",
			current: "unknown",
			latest:  "1.0.0",
			want:    false,
		},
		{
			name:    "dirty version never outdated",
			current: "1.0.0-dirty",
			latest:  "1.1.0",
			want:    false,
		},
		{
			name:    "empty version never outdated",
			current: "",
			latest:  "1.0.0",
			want:    false,
		},
		{
			name:    "prerelease version never outdated",
			current: "1.0.0-0.abc123",
			latest:  "1.1.0",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsNewer(tt.current, tt.latest); got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
