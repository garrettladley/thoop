package oauth

import (
	"github.com/garrettladley/thoop/internal/config"
	"golang.org/x/oauth2"
)

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

func NewConfig(whoop config.Whoop) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     whoop.ClientID,
		ClientSecret: whoop.ClientSecret,
		RedirectURL:  whoop.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}
