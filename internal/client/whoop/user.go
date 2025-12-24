package whoop

import (
	"context"
	"net/http"
)

type userService struct {
	client *Client
}

func (s *userService) GetProfile(ctx context.Context) (*UserProfile, error) {
	const route = "/v2/user/profile/basic"

	var profile UserProfile
	if err := s.client.do(ctx, http.MethodGet, route, nil, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *userService) GetBodyMeasurement(ctx context.Context) (*BodyMeasurement, error) {
	const route = "/v2/user/measurement/body"

	var measurement BodyMeasurement
	if err := s.client.do(ctx, http.MethodGet, route, nil, &measurement); err != nil {
		return nil, err
	}
	return &measurement, nil
}

func (s *userService) RevokeAccess(ctx context.Context) error {
	const route = "/v2/user/access"
	return s.client.do(ctx, http.MethodDelete, route, nil, nil)
}
