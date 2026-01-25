package chart

// Renderer is the interface for all chart components.
type Renderer interface {
	Render(width int) string
}

// Config holds common chart configuration.
type Config struct {
	Width      int
	Height     int
	ShowGrid   bool
	ShowLegend bool
}

// Option is a functional option for configuring charts.
type Option func(*Config)

// WithWidth sets the chart width.
func WithWidth(w int) Option {
	return func(c *Config) {
		c.Width = w
	}
}

// WithHeight sets the chart height.
func WithHeight(h int) Option {
	return func(c *Config) {
		c.Height = h
	}
}

// WithGrid enables or disables grid lines.
func WithGrid(show bool) Option {
	return func(c *Config) {
		c.ShowGrid = show
	}
}

// WithLegend enables or disables the legend.
func WithLegend(show bool) Option {
	return func(c *Config) {
		c.ShowLegend = show
	}
}
