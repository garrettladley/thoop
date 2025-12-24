package whoop

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
)

type Client struct {
	client *http.Client
}

func New(src oauth2.TokenSource) *Client {
	return &Client{
		client: oauth2.NewClient(context.Background(), src),
	}
}
