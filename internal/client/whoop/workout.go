package whoop

import (
	"context"
	"net/http"
)

type workoutService struct {
	client *Client
}

func (s *workoutService) Get(ctx context.Context, id string) (*Workout, error) {
	const route = "/v2/activity/workout"
	path := route + "/" + id

	var workout Workout
	if err := s.client.do(ctx, http.MethodGet, path, nil, &workout); err != nil {
		return nil, err
	}
	return &workout, nil
}

func (s *workoutService) List(ctx context.Context, params *ListParams) (*PaginatedResponse[Workout], error) {
	const route = "/v2/activity/workout"

	var resp PaginatedResponse[Workout]
	if err := s.client.do(ctx, http.MethodGet, route, params.values(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
