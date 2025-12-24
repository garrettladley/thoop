package whoop

import (
	"context"
	"net/http"
)

type sleepService struct {
	client *Client
}

func (s *sleepService) Get(ctx context.Context, id string) (*Sleep, error) {
	const route = "/v2/activity/sleep"
	path := route + "/" + id

	var sleep Sleep
	if err := s.client.do(ctx, http.MethodGet, path, nil, &sleep); err != nil {
		return nil, err
	}
	return &sleep, nil
}

func (s *sleepService) List(ctx context.Context, params *ListParams) (*PaginatedResponse[Sleep], error) {
	const route = "/v2/activity/sleep"

	var resp PaginatedResponse[Sleep]
	if err := s.client.do(ctx, http.MethodGet, route, params.values(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
