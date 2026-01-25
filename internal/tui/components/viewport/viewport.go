package viewport

import (
	"strings"
)

// Viewport manages scrolling for content that exceeds the visible area.
type Viewport struct {
	width  int
	height int
}

// Option configures a Viewport.
type Option func(*Viewport)

// WithSize sets the viewport dimensions.
func WithSize(width, height int) Option {
	return func(v *Viewport) {
		v.width = width
		v.height = height
	}
}

// New creates a new Viewport.
func New(opts ...Option) *Viewport {
	v := &Viewport{}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// SetSize updates the viewport dimensions.
func (v *Viewport) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Width returns the viewport width.
func (v *Viewport) Width() int {
	return v.width
}

// Height returns the viewport height.
func (v *Viewport) Height() int {
	return v.height
}

// Render renders the content with the given scroll offset.
// offset is the number of lines to skip from the top.
// Returns the visible portion of the content.
func (v *Viewport) Render(content string, offset int) string {
	if v.height <= 0 {
		return ""
	}

	lines := strings.Split(content, "\n")

	// clamp offset
	maxOffset := len(lines) - v.height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}

	// extract visible lines
	endLine := offset + v.height
	if endLine > len(lines) {
		endLine = len(lines)
	}

	visible := lines[offset:endLine]

	// pad to fill viewport if content is shorter
	for len(visible) < v.height {
		visible = append(visible, "")
	}

	return strings.Join(visible, "\n")
}

// ContentHeight returns the number of lines in the content.
func ContentHeight(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

// MaxOffset returns the maximum scroll offset for the given content.
func (v *Viewport) MaxOffset(content string) int {
	contentLines := ContentHeight(content)
	maxOffset := contentLines - v.height
	if maxOffset < 0 {
		return 0
	}
	return maxOffset
}

// ClampOffset clamps the offset to valid bounds for the given content.
func (v *Viewport) ClampOffset(content string, offset int) int {
	if offset < 0 {
		return 0
	}
	maxOffset := v.MaxOffset(content)
	if offset > maxOffset {
		return maxOffset
	}
	return offset
}

// AtTop returns true if the offset is at the top.
func AtTop(offset int) bool {
	return offset <= 0
}

// AtBottom returns true if at the bottom for the given content and viewport height.
func (v *Viewport) AtBottom(content string, offset int) bool {
	return offset >= v.MaxOffset(content)
}
