package version

import (
	"fmt"
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

type VersionError struct {
	ClientVersion string
	ServerVersion string
	MinVersion    string
}

func (e VersionError) Error() string {
	return fmt.Sprintf("client version %s incompatible with server version %s (requires v%s.x)",
		e.ClientVersion, e.ServerVersion, e.MinVersion)
}

func CheckCompatibility(clientVersion string) *VersionError {
	serverVersion := Get()

	if IsDevelopment(clientVersion) || IsDevelopment(serverVersion) {
		return nil
	}

	var (
		clientMajor = ParseMajor(clientVersion)
		serverMajor = ParseMajor(serverVersion)
	)

	if clientMajor == serverMajor {
		return nil
	}

	return &VersionError{
		ClientVersion: clientVersion,
		ServerVersion: serverVersion,
		MinVersion:    serverMajor,
	}
}
