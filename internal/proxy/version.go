package proxy

import (
	"fmt"

	"github.com/garrettladley/thoop/internal/version"
)

// VersionError contains information about version incompatibility.
type VersionError struct {
	ClientVersion string
	ProxyVersion  string
	MinVersion    string
}

func (e VersionError) Error() string {
	return fmt.Sprintf("client version %s incompatible with proxy version %s (requires v%s.x)",
		e.ClientVersion, e.ProxyVersion, e.MinVersion)
}

// CheckVersionCompatibility validates that client and proxy have the same major version.
// Development versions (devel, dirty, go install timestamps) are always allowed.
func CheckVersionCompatibility(clientVersion string) *VersionError {
	proxyVersion := version.Get()

	if version.IsDevelopment(clientVersion) || version.IsDevelopment(proxyVersion) {
		return nil
	}

	clientMajor := version.ParseMajor(clientVersion)
	proxyMajor := version.ParseMajor(proxyVersion)

	if clientMajor == proxyMajor {
		return nil
	}

	return &VersionError{
		ClientVersion: clientVersion,
		ProxyVersion:  proxyVersion,
		MinVersion:    proxyMajor,
	}
}
