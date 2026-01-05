package main

import (
	"context"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "lowercase y",
			input: "y\n",
			want:  true,
		},
		{
			name:  "uppercase Y",
			input: "Y\n",
			want:  true,
		},
		{
			name:  "lowercase yes",
			input: "yes\n",
			want:  true,
		},
		{
			name:  "uppercase YES",
			input: "YES\n",
			want:  true,
		},
		{
			name:  "mixed case Yes",
			input: "Yes\n",
			want:  true,
		},
		{
			name:  "n returns false",
			input: "n\n",
			want:  false,
		},
		{
			name:  "no returns false",
			input: "no\n",
			want:  false,
		},
		{
			name:  "empty input returns false",
			input: "\n",
			want:  false,
		},
		{
			name:  "random input returns false",
			input: "maybe\n",
			want:  false,
		},
		{
			name:  "y with leading whitespace",
			input: "  y\n",
			want:  true,
		},
		{
			name:  "y with trailing whitespace",
			input: "y  \n",
			want:  true,
		},
		{
			name:  "yes with surrounding whitespace",
			input: "  yes  \n",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := strings.NewReader(tt.input)
			if got := confirm(t.Context(), r); got != tt.want {
				t.Errorf("confirm(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConfirm_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// use a reader that blocks forever (empty string, no newline)
	r := strings.NewReader("")
	if got := confirm(ctx, r); got != false {
		t.Errorf("confirm with cancelled context = %v, want false", got)
	}
}
