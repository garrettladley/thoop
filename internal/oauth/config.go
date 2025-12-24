package oauth

import "golang.org/x/oauth2"

type ConfigProvider interface {
	GetClientID() string
	GetClientSecret() string
	GetRedirectURL() string
}

const (
	authURL  = "https://api.prod.whoop.com/oauth/oauth2/auth"
	tokenURL = "https://api.prod.whoop.com/oauth/oauth2/token" //nolint:gosec // not credentials, just endpoint URL
)

var scopes = []string{
	"offline",
	"read:recovery",
	"read:cycles",
	"read:sleep",
	"read:workout",
	"read:profile",
	"read:body_measurement",
}

func NewConfig(provider ConfigProvider) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     provider.GetClientID(),
		ClientSecret: provider.GetClientSecret(),
		RedirectURL:  provider.GetRedirectURL(),
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}
