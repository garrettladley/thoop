package whoop

import (
	"context"
	"net/http"
)

type recoveryService struct {
	client *Client
}

func (s *recoveryService) List(ctx context.Context, params *ListParams) (*PaginatedResponse[Recovery], error) {
	const route = "/v2/recovery"

	var resp PaginatedResponse[Recovery]
	if err := s.client.do(ctx, http.MethodGet, route, params.values(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
