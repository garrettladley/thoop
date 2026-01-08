package version

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/garrettladley/thoop"
)

const Header = "X-Client-Version"

// IsDevelopment returns true for versions that should skip compatibility checks.
func IsDevelopment(v string) bool {
	return strings.HasSuffix(v, "-snapshot")
}

func trimVPrefix(v string) string {
	return strings.TrimPrefix(v, "v")
}

// ParseMajor extracts the major version number from a semver string.
// Returns "0" for unparseable versions.
func ParseMajor(v string) string {
	v = trimVPrefix(v)
	if idx := strings.Index(v, "."); idx > 0 {
		return v[:idx]
	}
	return "0"
}

type VersionError struct {
	ClientVersion string
	ServerVersion string
	MinVersion    string
	IsUnstable    bool
}

func (e VersionError) Error() string {
	msg := fmt.Sprintf("client version %s incompatible with server version %s (requires v%s.x)",
		e.ClientVersion, e.ServerVersion, e.MinVersion)
	if e.IsUnstable {
		msg += " - version 0.x is unstable and does not guarantee compatibility"
	}
	return msg
}

func CheckCompatibility(clientVersion string) *VersionError {
	return CheckCompatibilityBetween(clientVersion, thoop.Version)
}

func CheckCompatibilityBetween(clientVersion, serverVersion string) *VersionError {
	if IsDevelopment(clientVersion) || IsDevelopment(serverVersion) {
		return nil
	}

	var (
		clientMajor = ParseMajor(clientVersion)
		serverMajor = ParseMajor(serverVersion)
	)

	// major version 0 indicates unstable/pre-release - must be exact match
	if (clientMajor == "0" || serverMajor == "0") && trimVPrefix(clientVersion) != trimVPrefix(serverVersion) {
		return &VersionError{
			ClientVersion: clientVersion,
			ServerVersion: serverVersion,
			MinVersion:    serverMajor,
			IsUnstable:    true,
		}
	}

	if clientMajor == serverMajor {
		return nil
	}

	return &VersionError{
		ClientVersion: clientVersion,
		ServerVersion: serverVersion,
		MinVersion:    serverMajor,
	}
}

// IsNewer returns true if latest is newer than current.
// Development versions are never considered outdated.
func IsNewer(current, latest string) bool {
	current = trimVPrefix(current)
	latest = trimVPrefix(latest)

	if IsDevelopment(current) {
		return false
	}

	return current != latest
}

// IsHomebrew returns true if the binary appears to be installed via homebrew.
func IsHomebrew() bool {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return false
	}

	execPath, err := os.Executable()
	if err != nil {
		return false
	}

	return strings.Contains(execPath, "homebrew") || strings.Contains(execPath, "Cellar")
}
