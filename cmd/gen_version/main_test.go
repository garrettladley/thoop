//go:build dev

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSemverString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sv   semver
		want string
	}{
		{
			name: "simple version",
			sv:   semver{Major: 1, Minor: 2, Patch: 3},
			want: "1.2.3",
		},
		{
			name: "zero version",
			sv:   semver{Major: 0, Minor: 0, Patch: 0},
			want: "0.0.0",
		},
		{
			name: "large numbers",
			sv:   semver{Major: 10, Minor: 20, Patch: 30},
			want: "10.20.30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.sv.String(); got != tt.want {
				t.Errorf("semver.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSemverSnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sv   semver
		want semver
	}{
		{
			name: "increments patch",
			sv:   semver{Major: 1, Minor: 2, Patch: 3},
			want: semver{Major: 1, Minor: 2, Patch: 4},
		},
		{
			name: "from zero patch",
			sv:   semver{Major: 0, Minor: 0, Patch: 0},
			want: semver{Major: 0, Minor: 0, Patch: 1},
		},
		{
			name: "preserves major and minor",
			sv:   semver{Major: 10, Minor: 20, Patch: 30},
			want: semver{Major: 10, Minor: 20, Patch: 31},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.sv.Snapshot()

			if got != tt.want {
				t.Errorf("semver.Snapshot() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSemverSnapshotString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sv   semver
		want string
	}{
		{
			name: "adds snapshot suffix with incremented patch",
			sv:   semver{Major: 0, Minor: 0, Patch: 11},
			want: "0.0.12-snapshot",
		},
		{
			name: "from 1.2.3",
			sv:   semver{Major: 1, Minor: 2, Patch: 3},
			want: "1.2.4-snapshot",
		},
		{
			name: "from zero",
			sv:   semver{Major: 0, Minor: 0, Patch: 0},
			want: "0.0.1-snapshot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.sv.SnapshotString(); got != tt.want {
				t.Errorf("semver.SnapshotString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		want    *semver
		wantErr bool
	}{
		{
			name:    "simple semver",
			version: "1.2.3",
			want:    &semver{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:    "with v prefix",
			version: "v1.2.3",
			want:    &semver{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:    "zero version",
			version: "0.0.0",
			want:    &semver{Major: 0, Minor: 0, Patch: 0},
		},
		{
			name:    "pre-1.0 version",
			version: "0.0.11",
			want:    &semver{Major: 0, Minor: 0, Patch: 11},
		},
		{
			name:    "large numbers",
			version: "10.20.30",
			want:    &semver{Major: 10, Minor: 20, Patch: 30},
		},
		{
			name:    "with snapshot suffix",
			version: "1.2.3-snapshot",
			want:    &semver{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:    "with prerelease suffix",
			version: "1.2.3-alpha.1",
			want:    &semver{Major: 1, Minor: 2, Patch: 3},
		},
		{
			name:    "major only",
			version: "1",
			wantErr: true,
		},
		{
			name:    "major and minor only",
			version: "1.2",
			wantErr: true,
		},
		{
			name:    "empty string",
			version: "",
			wantErr: true,
		},
		{
			name:    "v prefix only",
			version: "v",
			wantErr: true,
		},
		{
			name:    "non-numeric major",
			version: "abc.1.2",
			wantErr: true,
		},
		{
			name:    "non-numeric minor",
			version: "1.abc.2",
			wantErr: true,
		},
		{
			name:    "non-numeric patch",
			version: "1.2.abc",
			wantErr: true,
		},
		{
			name:    "with build metadata",
			version: "1.2.3+build.123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseVersion(tt.version)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseVersion(%q) expected error, got nil", tt.version)
				}
				return
			}

			if err != nil {
				t.Errorf("parseVersion(%q) unexpected error: %v", tt.version, err)
				return
			}

			if got.Major != tt.want.Major {
				t.Errorf("parseVersion(%q) Major = %d, want %d", tt.version, got.Major, tt.want.Major)
			}
			if got.Minor != tt.want.Minor {
				t.Errorf("parseVersion(%q) Minor = %d, want %d", tt.version, got.Minor, tt.want.Minor)
			}
			if got.Patch != tt.want.Patch {
				t.Errorf("parseVersion(%q) Patch = %d, want %d", tt.version, got.Patch, tt.want.Patch)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		version     string
		snapshot    bool
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "release version without snapshot",
			version:     "1.2.3",
			snapshot:    false,
			wantVersion: "1.2.3",
		},
		{
			name:        "release version with snapshot",
			version:     "1.2.3",
			snapshot:    true,
			wantVersion: "1.2.4-snapshot",
		},
		{
			name:        "snapshot version without snapshot flag",
			version:     "1.2.4-snapshot",
			snapshot:    false,
			wantVersion: "1.2.4-snapshot",
		},
		{
			name:        "snapshot version with snapshot flag - no double increment",
			version:     "1.2.4-snapshot",
			snapshot:    true,
			wantVersion: "1.2.4-snapshot",
		},
		{
			name:        "zero version with snapshot",
			version:     "0.0.0",
			snapshot:    true,
			wantVersion: "0.0.1-snapshot",
		},
		{
			name:    "invalid version",
			version: "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, versionStr, err := generate(tt.version, tt.snapshot)

			if tt.wantErr {
				if err == nil {
					t.Errorf("generate(%q, %v) expected error, got nil", tt.version, tt.snapshot)
				}
				return
			}

			if err != nil {
				t.Errorf("generate(%q, %v) unexpected error: %v", tt.version, tt.snapshot, err)
				return
			}

			if versionStr != tt.wantVersion {
				t.Errorf("generate(%q, %v) version = %q, want %q", tt.version, tt.snapshot, versionStr, tt.wantVersion)
			}

			if !strings.Contains(content, tt.wantVersion) {
				t.Errorf("generate(%q, %v) content doesn't contain version %q", tt.version, tt.snapshot, tt.wantVersion)
			}
		})
	}
}

func TestDoubleSnapshotIsIdempotent(t *testing.T) {
	t.Parallel()

	// start with a release version
	version := "1.2.3"

	// first snapshot
	_, firstVersion, err := generate(version, true)
	if err != nil {
		t.Fatalf("first generate failed: %v", err)
	}

	if firstVersion != "1.2.4-snapshot" {
		t.Errorf("first snapshot version = %q, want %q", firstVersion, "1.2.4-snapshot")
	}

	// second snapshot (should not increment again)
	_, secondVersion, err := generate(firstVersion, true)
	if err != nil {
		t.Fatalf("second generate failed: %v", err)
	}

	if secondVersion != firstVersion {
		t.Errorf("double snapshot changed version: got %q, want %q", secondVersion, firstVersion)
	}

	// third snapshot (still should not increment)
	_, thirdVersion, err := generate(secondVersion, true)
	if err != nil {
		t.Fatalf("third generate failed: %v", err)
	}

	if thirdVersion != firstVersion {
		t.Errorf("triple snapshot changed version: got %q, want %q", thirdVersion, firstVersion)
	}
}

func TestGeneratorRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		manifest    string
		snapshot    bool
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "valid manifest without snapshot",
			manifest:    `{".": "1.2.3"}`,
			snapshot:    false,
			wantVersion: "1.2.3",
		},
		{
			name:        "valid manifest with snapshot",
			manifest:    `{".": "1.2.3"}`,
			snapshot:    true,
			wantVersion: "1.2.4-snapshot",
		},
		{
			name:        "manifest with snapshot version - no double increment",
			manifest:    `{".": "1.2.4-snapshot"}`,
			snapshot:    true,
			wantVersion: "1.2.4-snapshot",
		},
		{
			name:     "invalid json",
			manifest: `{invalid}`,
			wantErr:  true,
		},
		{
			name:     "missing version key",
			manifest: `{"other": "1.2.3"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := strings.NewReader(tt.manifest)
			var output bytes.Buffer

			g := &generator{
				manifestReader: reader,
				outputWriter:   &output,
			}

			version, err := g.run(tt.snapshot)

			if tt.wantErr {
				if err == nil {
					t.Errorf("run() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("run() unexpected error: %v", err)
				return
			}

			if version != tt.wantVersion {
				t.Errorf("run() version = %q, want %q", version, tt.wantVersion)
			}

			if !strings.Contains(output.String(), tt.wantVersion) {
				t.Errorf("run() output doesn't contain version %q", tt.wantVersion)
			}
		})
	}
}
