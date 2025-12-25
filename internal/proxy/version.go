package proxy

import (
	"fmt"

	"github.com/garrettladley/thoop/internal/version"
)

type VersionError struct {
	ClientVersion string
	ProxyVersion  string
	MinVersion    string
}

func (e VersionError) Error() string {
	return fmt.Sprintf("client version %s incompatible with proxy version %s (requires v%s.x)",
		e.ClientVersion, e.ProxyVersion, e.MinVersion)
}

func CheckVersionCompatibility(clientVersion string) *VersionError {
	proxyVersion := version.Get()

	if version.IsDevelopment(clientVersion) || version.IsDevelopment(proxyVersion) {
		return nil
	}

	var (
		clientMajor = version.ParseMajor(clientVersion)
		proxyMajor  = version.ParseMajor(proxyVersion)
	)

	if clientMajor == proxyMajor {
		return nil
	}

	return &VersionError{
		ClientVersion: clientVersion,
		ProxyVersion:  proxyVersion,
		MinVersion:    proxyMajor,
	}
}
