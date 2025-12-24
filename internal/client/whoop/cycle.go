package whoop

import (
	"context"
	"fmt"
	"net/http"
)

type cycleService struct {
	client *Client
}

func (s *cycleService) Get(ctx context.Context, id int64) (*Cycle, error) {
	const route = "/v2/cycle"
	path := fmt.Sprintf("%s/%d", route, id)

	var cycle Cycle
	if err := s.client.do(ctx, http.MethodGet, path, nil, &cycle); err != nil {
		return nil, err
	}
	return &cycle, nil
}

func (s *cycleService) List(ctx context.Context, params *ListParams) (*PaginatedResponse[Cycle], error) {
	const route = "/v2/cycle"

	var resp PaginatedResponse[Cycle]
	if err := s.client.do(ctx, http.MethodGet, route, params.values(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *cycleService) GetSleep(ctx context.Context, cycleID int64) (*Sleep, error) {
	const route = "/v2/cycle"
	path := fmt.Sprintf("%s/%d/sleep", route, cycleID)

	var sleep Sleep
	if err := s.client.do(ctx, http.MethodGet, path, nil, &sleep); err != nil {
		return nil, err
	}
	return &sleep, nil
}

func (s *cycleService) GetRecovery(ctx context.Context, cycleID int64) (*Recovery, error) {
	const route = "/v2/cycle"
	path := fmt.Sprintf("%s/%d/recovery", route, cycleID)

	var recovery Recovery
	if err := s.client.do(ctx, http.MethodGet, path, nil, &recovery); err != nil {
		return nil, err
	}
	return &recovery, nil
}
