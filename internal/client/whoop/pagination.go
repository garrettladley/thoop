package whoop

import (
	"net/url"
	"strconv"
	"time"
)

type ListParams struct {
	Limit     int
	Start     *time.Time
	End       *time.Time
	NextToken *string
}

func (p *ListParams) values() url.Values {
	if p == nil {
		return nil
	}

	v := make(url.Values)

	if p.Limit > 0 {
		v.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Start != nil {
		v.Set("start", p.Start.Format(time.RFC3339))
	}
	if p.End != nil {
		v.Set("end", p.End.Format(time.RFC3339))
	}
	if p.NextToken != nil {
		v.Set("nextToken", *p.NextToken)
	}

	return v
}

type PaginatedResponse[T any] struct {
	Records   []T     `json:"records"`
	NextToken *string `json:"next_token,omitempty"`
}

func (p *PaginatedResponse[T]) HasMore() bool {
	return p.NextToken != nil && *p.NextToken != ""
}
