package version

import "testing"

func TestCheckCompatibilityBetween(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		clientVersion string
		serverVersion string
		wantError     bool
		wantUnstable  bool
	}{
		{
			name:          "exact same v0.x versions are compatible",
			clientVersion: "0.0.6",
			serverVersion: "0.0.6",
			wantError:     false,
		},
		{
			name:          "exact same v0.x versions with v prefix are compatible",
			clientVersion: "v0.0.6",
			serverVersion: "v0.0.6",
			wantError:     false,
		},
		{
			name:          "same v0.x versions with mixed v prefix are compatible",
			clientVersion: "0.0.6",
			serverVersion: "v0.0.6",
			wantError:     false,
		},
		{
			name:          "different v0.x versions are incompatible",
			clientVersion: "0.0.5",
			serverVersion: "0.0.6",
			wantError:     true,
			wantUnstable:  true,
		},
		{
			name:          "v0.x client with v1.x server is incompatible",
			clientVersion: "0.0.6",
			serverVersion: "1.0.0",
			wantError:     true,
			wantUnstable:  true,
		},
		{
			name:          "v1.x client with v0.x server is incompatible",
			clientVersion: "1.0.0",
			serverVersion: "0.0.6",
			wantError:     true,
			wantUnstable:  true,
		},
		{
			name:          "same major stable versions are compatible",
			clientVersion: "1.0.0",
			serverVersion: "1.2.3",
			wantError:     false,
		},
		{
			name:          "different major stable versions are incompatible",
			clientVersion: "1.0.0",
			serverVersion: "2.0.0",
			wantError:     true,
			wantUnstable:  false,
		},
		{
			name:          "devel client skips check",
			clientVersion: "devel",
			serverVersion: "1.0.0",
			wantError:     false,
		},
		{
			name:          "devel server skips check",
			clientVersion: "1.0.0",
			serverVersion: "devel",
			wantError:     false,
		},
		{
			name:          "dirty version skips check",
			clientVersion: "1.0.0-dirty",
			serverVersion: "2.0.0",
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckCompatibilityBetween(tt.clientVersion, tt.serverVersion)
			if tt.wantError {
				if err == nil {
					t.Errorf("CheckCompatibilityBetween(%q, %q) = nil, want error", tt.clientVersion, tt.serverVersion)
					return
				}
				if err.IsUnstable != tt.wantUnstable {
					t.Errorf("CheckCompatibilityBetween(%q, %q).IsUnstable = %v, want %v", tt.clientVersion, tt.serverVersion, err.IsUnstable, tt.wantUnstable)
				}
			} else {
				if err != nil {
					t.Errorf("CheckCompatibilityBetween(%q, %q) = %v, want nil", tt.clientVersion, tt.serverVersion, err)
				}
			}
		})
	}
}

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
