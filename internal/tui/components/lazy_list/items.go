package lazy_list

import (
	"strings"
)

// Ensure all item types implement the Item interface.
var (
	_ Item = (*ChartItem)(nil)
	_ Item = (*StaticItem)(nil)
	_ Item = (*SpacerItem)(nil)
	_ Item = (*FuncItem)(nil)
)

// ChartItem wraps a chart render function with built-in caching.
type ChartItem struct {
	id         string
	renderFunc func(width int) string
	cached     string
	cacheWidth int
}

// NewChartItem creates a new chart item with the given ID and render function.
func NewChartItem(id string, renderFunc func(width int) string) *ChartItem {
	return &ChartItem{
		id:         id,
		renderFunc: renderFunc,
	}
}

// Render renders the chart at the given width, using cache if available.
func (c *ChartItem) Render(width int) string {
	if c.cacheWidth == width && c.cached != "" {
		return c.cached
	}
	c.cached = c.renderFunc(width)
	c.cacheWidth = width
	return c.cached
}

// Height returns the height of the chart when rendered at the given width.
func (c *ChartItem) Height(width int) int {
	content := c.Render(width)
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

// Invalidate clears the cache, forcing a re-render on next call.
func (c *ChartItem) Invalidate() {
	c.cached = ""
	c.cacheWidth = 0
}

// StaticItem wraps pre-rendered content that doesn't change.
type StaticItem struct {
	content string
	height  int
}

// NewStaticItem creates a new static item with the given content.
func NewStaticItem(content string) *StaticItem {
	height := 0
	if content != "" {
		height = strings.Count(content, "\n") + 1
	}
	return &StaticItem{
		content: content,
		height:  height,
	}
}

// Render returns the static content (width is ignored).
func (s *StaticItem) Render(_ int) string {
	return s.content
}

// Height returns the height of the static content.
func (s *StaticItem) Height(_ int) int {
	return s.height
}

// SpacerItem adds vertical spacing between items.
type SpacerItem struct {
	lines int
}

// NewSpacerItem creates a new spacer with the given number of lines.
func NewSpacerItem(lines int) *SpacerItem {
	return &SpacerItem{lines: lines}
}

// Render returns empty lines for spacing.
func (s *SpacerItem) Render(_ int) string {
	if s.lines <= 0 {
		return ""
	}
	return strings.Repeat("\n", s.lines-1)
}

// Height returns the number of lines this spacer occupies.
func (s *SpacerItem) Height(_ int) int {
	return s.lines
}

// FuncItem wraps a function that renders content dynamically.
// Unlike ChartItem, FuncItem does not cache its output.
type FuncItem struct {
	renderFunc func(width int) string
}

// NewFuncItem creates a new function item with the given render function.
func NewFuncItem(renderFunc func(width int) string) *FuncItem {
	return &FuncItem{renderFunc: renderFunc}
}

// Render calls the render function with the given width.
func (f *FuncItem) Render(width int) string {
	return f.renderFunc(width)
}

// Height returns the height of the rendered content.
func (f *FuncItem) Height(width int) int {
	content := f.Render(width)
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}
