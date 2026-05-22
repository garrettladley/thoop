package version

import "testing"

const (
	testVersion006         = "0.0.6"
	testVersionV006        = "v0.0.6"
	testVersion100         = "1.0.0"
	testVersionV100        = "v1.0.0"
	testVersion200         = "2.0.0"
	testVersion100Snapshot = "1.0.0-snapshot"
)

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
			clientVersion: testVersion006,
			serverVersion: testVersion006,
			wantError:     false,
		},
		{
			name:          "exact same v0.x versions with v prefix are compatible",
			clientVersion: testVersionV006,
			serverVersion: testVersionV006,
			wantError:     false,
		},
		{
			name:          "same v0.x versions with mixed v prefix are compatible",
			clientVersion: testVersion006,
			serverVersion: testVersionV006,
			wantError:     false,
		},
		{
			name:          "different v0.x versions are incompatible",
			clientVersion: "0.0.5",
			serverVersion: testVersion006,
			wantError:     true,
			wantUnstable:  true,
		},
		{
			name:          "v0.x client with v1.x server is incompatible",
			clientVersion: testVersion006,
			serverVersion: testVersion100,
			wantError:     true,
			wantUnstable:  true,
		},
		{
			name:          "v1.x client with v0.x server is incompatible",
			clientVersion: testVersion100,
			serverVersion: testVersion006,
			wantError:     true,
			wantUnstable:  true,
		},
		{
			name:          "same major stable versions are compatible",
			clientVersion: testVersion100,
			serverVersion: "1.2.3",
			wantError:     false,
		},
		{
			name:          "different major stable versions are incompatible",
			clientVersion: testVersion100,
			serverVersion: testVersion200,
			wantError:     true,
			wantUnstable:  false,
		},
		{
			name:          "snapshot client skips check",
			clientVersion: testVersion100Snapshot,
			serverVersion: testVersion200,
			wantError:     false,
		},
		{
			name:          "snapshot server skips check",
			clientVersion: testVersion100,
			serverVersion: "2.0.0-snapshot",
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
			current: testVersion100,
			latest:  testVersion100,
			want:    false,
		},
		{
			name:    "same version with v prefix on current",
			current: testVersionV100,
			latest:  testVersion100,
			want:    false,
		},
		{
			name:    "same version with v prefix on latest",
			current: testVersion100,
			latest:  testVersionV100,
			want:    false,
		},
		{
			name:    "same version with v prefix on both",
			current: testVersionV100,
			latest:  testVersionV100,
			want:    false,
		},
		{
			name:    "newer version available",
			current: testVersion100,
			latest:  "1.1.0",
			want:    true,
		},
		{
			name:    "major version bump",
			current: testVersion100,
			latest:  testVersion200,
			want:    true,
		},
		{
			name:    "patch version bump",
			current: testVersion100,
			latest:  "1.0.1",
			want:    true,
		},
		{
			name:    "snapshot version never outdated",
			current: testVersion100Snapshot,
			latest:  testVersion200,
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

func TestIsDevelopment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		want    bool
	}{
		{testVersion100, false},
		{"0.0.11", false},
		{testVersionV100, false},
		{testVersion100Snapshot, true},
		{"0.0.12-snapshot", true},
		{"v1.0.0-snapshot", true},
		{"1.0.0-SNAPSHOT", false}, // case sensitive
		{"snapshot", false},       // must be suffix
		{"1.0.0-snapshot-extra", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			t.Parallel()
			if got := IsDevelopment(tt.version); got != tt.want {
				t.Errorf("IsDevelopment(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
