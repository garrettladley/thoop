package whoop

import (
	"net/http"
	"testing"
	"time"
)

func TestParseRateLimitValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "simple integer",
			input:   "100",
			want:    100,
			wantErr: false,
		},
		{
			name:    "complex format with windows",
			input:   "100, 100;window=60, 10000;window=86400",
			want:    100,
			wantErr: false,
		},
		{
			name:    "day limit complex format",
			input:   "10000, 100;window=60, 10000;window=86400",
			want:    10000,
			wantErr: false,
		},
		{
			name:    "with whitespace",
			input:   "  100  ",
			want:    100,
			wantErr: false,
		},
		{
			name:    "with semicolon attributes",
			input:   "100;window=60",
			want:    100,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "non-numeric",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseRateLimitValue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRateLimitValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseRateLimitValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRateLimitHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		headers http.Header
		want    *RateLimitInfo
		wantErr bool
	}{
		{
			name: "simple headers",
			headers: http.Header{
				limitHeaderKey:     []string{"100"},
				remainingHeaderKey: []string{"99"},
				resetHeaderKey:     []string{"60"},
			},
			want: &RateLimitInfo{
				Limit:     100,
				Remaining: 99,
				Reset:     60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "WHOOP docs example",
			headers: http.Header{
				limitHeaderKey:     []string{"100, 100;window=60, 10000;window=86400"},
				remainingHeaderKey: []string{"98"},
				resetHeaderKey:     []string{"3"},
			},
			want: &RateLimitInfo{
				Limit:     100,
				Remaining: 98,
				Reset:     3 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "complex WHOOP format with remaining windows",
			headers: http.Header{
				limitHeaderKey:     []string{"100, 100;window=60, 10000;window=86400"},
				remainingHeaderKey: []string{"95, 95;window=60, 9950;window=86400"},
				resetHeaderKey:     []string{"45"},
			},
			want: &RateLimitInfo{
				Limit:     100,
				Remaining: 95,
				Reset:     45 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "missing headers returns nil",
			headers: http.Header{},
			want:    nil,
			wantErr: false,
		},
		{
			name: "partial headers returns nil",
			headers: http.Header{
				limitHeaderKey: []string{"100"},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "invalid limit returns error",
			headers: http.Header{
				limitHeaderKey:     []string{"invalid"},
				remainingHeaderKey: []string{"99"},
				resetHeaderKey:     []string{"1735100000"},
			},
			wantErr: true,
		},
		{
			name: "invalid remaining returns error",
			headers: http.Header{
				limitHeaderKey:     []string{"100"},
				remainingHeaderKey: []string{"invalid"},
				resetHeaderKey:     []string{"1735100000"},
			},
			wantErr: true,
		},
		{
			name: "invalid reset returns error",
			headers: http.Header{
				limitHeaderKey:     []string{"100"},
				remainingHeaderKey: []string{"99"},
				resetHeaderKey:     []string{"invalid"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseRateLimitHeaders(tt.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRateLimitHeaders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if tt.want == nil && got != nil {
				t.Errorf("ParseRateLimitHeaders() = %v, want nil", got)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("ParseRateLimitHeaders() = nil, want %v", tt.want)
				return
			}
			if tt.want != nil && got != nil {
				if got.Limit != tt.want.Limit || got.Remaining != tt.want.Remaining || got.Reset != tt.want.Reset {
					t.Errorf("ParseRateLimitHeaders() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}
