package dashboard

import (
	"testing"

	"github.com/garrettladley/thoop/internal/tui/components/metric_row"
)

func TestGetDirectionHigherBetter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		current   float64
		avg       float64
		precision int
		expected  metric_row.Direction
	}{
		{
			name:      "no precision: clearly higher",
			current:   10.0,
			avg:       5.0,
			precision: -1, // -1 means no precision option
			expected:  metric_row.DirectionUp,
		},
		{
			name:      "no precision: clearly lower",
			current:   5.0,
			avg:       10.0,
			precision: -1,
			expected:  metric_row.DirectionDown,
		},
		{
			name:      "no precision: equal",
			current:   5.0,
			avg:       5.0,
			precision: -1,
			expected:  metric_row.DirectionNeutral,
		},
		{
			name:      "bug case: 6.75 vs 6.84 both display as 6.8 with precision 1",
			current:   6.75,
			avg:       6.84,
			precision: 1,
			expected:  metric_row.DirectionNeutral, // should be neutral since both round to 6.8
		},
		{
			name:      "bug case: 6.84 vs 6.75 both display as 6.8 with precision 1",
			current:   6.84,
			avg:       6.75,
			precision: 1,
			expected:  metric_row.DirectionNeutral, // should be neutral since both round to 6.8
		},
		{
			name:      "precision 0: 45.2 vs 45.4 both round to 45",
			current:   45.2,
			avg:       45.4,
			precision: 0,
			expected:  metric_row.DirectionNeutral,
		},
		{
			name:      "precision 0: 46.4 vs 45.5 round to 46 vs 46",
			current:   46.4,
			avg:       45.5,
			precision: 0,
			expected:  metric_row.DirectionNeutral,
		},
		{
			name:      "precision 0: 46.5 vs 45.4 round to 47 vs 45 - up",
			current:   46.5,
			avg:       45.4,
			precision: 0,
			expected:  metric_row.DirectionUp,
		},
		{
			name:      "precision 0: 44.4 vs 45.5 round to 44 vs 46 - down",
			current:   44.4,
			avg:       45.5,
			precision: 0,
			expected:  metric_row.DirectionDown,
		},
		{
			name:      "precision 1: 6.85 vs 6.74 round to 6.9 vs 6.7 - up",
			current:   6.85,
			avg:       6.74,
			precision: 1,
			expected:  metric_row.DirectionUp,
		},
		{
			name:      "precision 1: 6.74 vs 6.85 round to 6.7 vs 6.9 - down",
			current:   6.74,
			avg:       6.85,
			precision: 1,
			expected:  metric_row.DirectionDown,
		},
		{
			name:      "zero average returns none",
			current:   10.0,
			avg:       0.0,
			precision: 0,
			expected:  metric_row.DirectionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result metric_row.Direction
			if tt.precision == -1 {
				result = getDirectionHigherBetter(tt.current, tt.avg)
			} else {
				result = getDirectionHigherBetter(tt.current, tt.avg, WithPrecision(tt.precision))
			}

			if result != tt.expected {
				t.Errorf("getDirectionHigherBetter(%v, %v, precision=%d) = %v, want %v",
					tt.current, tt.avg, tt.precision, result, tt.expected)
			}
		})
	}
}

func TestGetDirectionLowerBetter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		current   float64
		avg       float64
		precision int
		expected  metric_row.Direction
	}{
		{
			name:      "no precision: clearly higher (bad)",
			current:   10.0,
			avg:       5.0,
			precision: -1,
			expected:  metric_row.DirectionUpBad,
		},
		{
			name:      "no precision: clearly lower (good)",
			current:   5.0,
			avg:       10.0,
			precision: -1,
			expected:  metric_row.DirectionDownGood,
		},
		{
			name:      "no precision: equal",
			current:   5.0,
			avg:       5.0,
			precision: -1,
			expected:  metric_row.DirectionNeutral,
		},
		{
			name:      "bug case: 54.2 vs 54.4 both round to 54 with precision 0",
			current:   54.2,
			avg:       54.4,
			precision: 0,
			expected:  metric_row.DirectionNeutral,
		},
		{
			name:      "precision 1: 15.22 vs 15.24 both round to 15.2",
			current:   15.22,
			avg:       15.24,
			precision: 1,
			expected:  metric_row.DirectionNeutral,
		},
		{
			name:      "precision 1: 15.35 vs 15.24 round to 15.4 vs 15.2 - up bad",
			current:   15.35,
			avg:       15.24,
			precision: 1,
			expected:  metric_row.DirectionUpBad,
		},
		{
			name:      "precision 1: 15.14 vs 15.25 round to 15.1 vs 15.3 - down good",
			current:   15.14,
			avg:       15.25,
			precision: 1,
			expected:  metric_row.DirectionDownGood,
		},
		{
			name:      "zero average returns none",
			current:   10.0,
			avg:       0.0,
			precision: 0,
			expected:  metric_row.DirectionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result metric_row.Direction
			if tt.precision == -1 {
				result = getDirectionLowerBetter(tt.current, tt.avg)
			} else {
				result = getDirectionLowerBetter(tt.current, tt.avg, WithPrecision(tt.precision))
			}

			if result != tt.expected {
				t.Errorf("getDirectionLowerBetter(%v, %v, precision=%d) = %v, want %v",
					tt.current, tt.avg, tt.precision, result, tt.expected)
			}
		})
	}
}
