package version

import (
	"runtime/debug"
	"strings"
	"sync"
)

const Header = "X-Client-Version"

const (
	versionDevel   = "devel"
	versionUnknown = "unknown"
)

// version is set via ldflags at build time.
// falls back to debug.ReadBuildInfo for go install.
var version = versionDevel

var once sync.Once

func Get() string {
	once.Do(func() {
		if version != versionDevel {
			return
		}
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}
		if v := info.Main.Version; v != "" && v != "("+versionDevel+")" {
			version = v
		}
	})
	return version
}

// IsDevelopment returns true for versions that should skip compatibility checks.
func IsDevelopment(v string) bool {
	return v == versionDevel || v == versionUnknown || v == "" ||
		strings.Contains(v, "dirty") ||
		strings.Contains(v, "-0.")
}

// ParseMajor extracts the major version number from a semver string.
// Returns "0" for unparseable versions.
func ParseMajor(v string) string {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.Index(v, "."); idx > 0 {
		return v[:idx]
	}
	return "0"
}
