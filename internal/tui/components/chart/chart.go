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
