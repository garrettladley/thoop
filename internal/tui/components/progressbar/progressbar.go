package progressbar

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

type ProgressBar struct {
	Width      int
	Percentage float64
	FillColor  color.Color
	EmptyColor color.Color
}

type Option func(*ProgressBar)

func WithFillColor(c color.Color) Option {
	return func(p *ProgressBar) {
		p.FillColor = c
	}
}

func WithEmptyColor(c color.Color) Option {
	return func(p *ProgressBar) {
		p.EmptyColor = c
	}
}

func New(width int, percentage float64, opts ...Option) ProgressBar {
	p := ProgressBar{
		Width:      width,
		Percentage: percentage,
		FillColor:  theme.ColorTeal,
		EmptyColor: theme.ColorDim,
	}
	for _, opt := range opts {
		opt(&p)
	}
	return p
}

func (p ProgressBar) Render() string {
	if p.Width <= 0 {
		return ""
	}

	percentage := p.Percentage
	if percentage > 1 {
		percentage = 1
	}
	if percentage < 0 {
		percentage = 0
	}

	filled := int(float64(p.Width) * percentage)
	empty := p.Width - filled

	fillStyle := lipgloss.NewStyle().Foreground(p.FillColor)
	emptyStyle := lipgloss.NewStyle().Foreground(p.EmptyColor)

	bar := fillStyle.Render(strings.Repeat("━", filled)) +
		emptyStyle.Render(strings.Repeat("━", empty))

	return bar
}

// SegmentedProgressBar represents a 3-segment progress bar
// where only the active segment is highlighted based on the percentage range
type SegmentedProgressBar struct {
	Width               int
	Percentage          float64
	PoorColor           color.Color // below SufficientThreshold
	SufficientColor     color.Color // SufficientThreshold to OptimalThreshold
	OptimalColor        color.Color // above OptimalThreshold
	DimColor            color.Color
	SegmentWidth        int
	SufficientThreshold float64 // threshold for sufficient (default 70)
	OptimalThreshold    float64 // threshold for optimal (default 85)
}

type SegmentedOption func(*SegmentedProgressBar)

func WithSegmentWidth(w int) SegmentedOption {
	return func(s *SegmentedProgressBar) {
		s.SegmentWidth = w
	}
}

func WithSegmentedColors(poor, sufficient, optimal color.Color) SegmentedOption {
	return func(s *SegmentedProgressBar) {
		s.PoorColor = poor
		s.SufficientColor = sufficient
		s.OptimalColor = optimal
	}
}

// WithSegmentedThresholds sets custom thresholds for sufficient and optimal.
func WithSegmentedThresholds(sufficient, optimal float64) SegmentedOption {
	return func(s *SegmentedProgressBar) {
		s.SufficientThreshold = sufficient
		s.OptimalThreshold = optimal
	}
}

func NewSegmented(percentage float64, opts ...SegmentedOption) SegmentedProgressBar {
	s := SegmentedProgressBar{
		Percentage:          percentage,
		PoorColor:           theme.ColorOrange,
		SufficientColor:     theme.ColorNeutral,
		OptimalColor:        theme.ColorTeal,
		DimColor:            theme.ColorDim,
		SegmentWidth:        6,
		SufficientThreshold: 70,
		OptimalThreshold:    85,
	}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

func (s SegmentedProgressBar) Render() string {
	// determine which segment is active based on percentage and thresholds
	var poorStyle, suffStyle, optStyle lipgloss.Style
	dimStyle := lipgloss.NewStyle().Foreground(s.DimColor)

	pct := s.Percentage
	if pct > 100 {
		pct = 100
	}

	switch {
	case pct >= s.OptimalThreshold:
		poorStyle = dimStyle
		suffStyle = dimStyle
		optStyle = lipgloss.NewStyle().Foreground(s.OptimalColor)
	case pct >= s.SufficientThreshold:
		poorStyle = dimStyle
		suffStyle = lipgloss.NewStyle().Foreground(s.SufficientColor)
		optStyle = dimStyle
	default:
		poorStyle = lipgloss.NewStyle().Foreground(s.PoorColor)
		suffStyle = dimStyle
		optStyle = dimStyle
	}

	segment := strings.Repeat("━", s.SegmentWidth)
	gap := " "

	return poorStyle.Render(segment) + gap + suffStyle.Render(segment) + gap + optStyle.Render(segment)
}
