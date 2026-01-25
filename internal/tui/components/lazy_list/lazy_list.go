package lazy_list

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Item represents a renderable section (chart, metric row, etc.)
type Item interface {
	// Render renders the item at the given width.
	Render(width int) string
	// Height returns the height of the item when rendered at the given width.
	// This may trigger a render if not cached.
	Height(width int) int
}

// renderedItem caches the render result and height for an item.
type renderedItem struct {
	content string
	height  int
}

// List manages lazy rendering of items with viewport scrolling.
type List struct {
	width  int
	height int
	items  []Item
	gap    int // vertical gap between items

	// scroll state: item-based offset
	offsetIdx  int // first visible item index
	offsetLine int // ;ines hidden in first visible item

	// cache of rendered items (cleared on width change)
	cache      []renderedItem
	cacheWidth int
}

// New creates a new lazy list with the given items.
func New(items []Item) *List {
	return &List{
		items: items,
		gap:   0,
		cache: make([]renderedItem, len(items)),
	}
}

// WithGap sets the gap between items (in lines).
func (l *List) WithGap(gap int) *List {
	l.gap = gap
	return l
}

// SetSize sets the viewport dimensions.
func (l *List) SetSize(width, height int) {
	if l.cacheWidth != width {
		// width changed, invalidate cache
		l.cache = make([]renderedItem, len(l.items))
		l.cacheWidth = width
	}
	l.width = width
	l.height = height
}

// getItem renders an item lazily and returns its content and height.
func (l *List) getItem(idx int) (string, int) {
	if idx < 0 || idx >= len(l.items) {
		return "", 0
	}

	if l.cache[idx].content != "" {
		return l.cache[idx].content, l.cache[idx].height
	}

	// render and cache
	content := l.items[idx].Render(l.width)
	height := strings.Count(content, "\n") + 1
	if content == "" {
		height = 0
	}
	l.cache[idx] = renderedItem{content: content, height: height}
	return content, height
}

// TotalHeight returns the total height of all items plus gaps.
func (l *List) TotalHeight() int {
	total := 0
	for i := range l.items {
		_, h := l.getItem(i)
		total += h
		if i < len(l.items)-1 && h > 0 {
			total += l.gap
		}
	}
	return total
}

// SetOffset sets the scroll offset from a line-based offset.
// Converts line offset to item-based offset.
func (l *List) SetOffset(lineOffset int) {
	if lineOffset <= 0 {
		l.offsetIdx = 0
		l.offsetLine = 0
		return
	}

	currentLine := 0
	for i := range l.items {
		_, h := l.getItem(i)
		itemEnd := currentLine + h
		if i < len(l.items)-1 && h > 0 {
			itemEnd += l.gap
		}

		if lineOffset < itemEnd {
			l.offsetIdx = i
			l.offsetLine = lineOffset - currentLine
			return
		}
		currentLine = itemEnd
	}

	// past end, clamp to last item
	if len(l.items) > 0 {
		l.offsetIdx = len(l.items) - 1
		_, h := l.getItem(l.offsetIdx)
		l.offsetLine = max(h-1, 0)
	}
}

// ScrollBy scrolls by the given number of lines (positive = down, negative = up).
func (l *List) ScrollBy(lines int) {
	if len(l.items) == 0 {
		return
	}

	totalHeight := l.TotalHeight()
	maxOffset := max(totalHeight-l.height, 0)

	// convert current state to line offset
	currentLineOffset := l.LineOffset()

	// apply scroll
	newOffset := max(currentLineOffset+lines, 0)
	newOffset = min(newOffset, maxOffset)

	// convert back to item-based offset
	l.SetOffset(newOffset)
}

// LineOffset returns the current scroll offset in lines.
func (l *List) LineOffset() int {
	lineOffset := 0
	for i := range l.offsetIdx {
		_, h := l.getItem(i)
		lineOffset += h
		if i < len(l.items)-1 && h > 0 {
			lineOffset += l.gap
		}
	}
	return lineOffset + l.offsetLine
}

// MaxOffset returns the maximum line offset.
func (l *List) MaxOffset() int {
	totalHeight := l.TotalHeight()
	return max(totalHeight-l.height, 0)
}

// ClampOffset clamps a line offset to valid bounds.
func (l *List) ClampOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	maxOffset := l.MaxOffset()
	if offset > maxOffset {
		return maxOffset
	}
	return offset
}

// AtTop returns true if scrolled to the top.
func (l *List) AtTop() bool {
	return l.offsetIdx == 0 && l.offsetLine == 0
}

// AtBottom returns true if scrolled to the bottom.
func (l *List) AtBottom() bool {
	return l.LineOffset() >= l.MaxOffset()
}

// VisibleItemIndices returns the range of visible item indices [start, end).
func (l *List) VisibleItemIndices() (start, end int) {
	if len(l.items) == 0 || l.height <= 0 {
		return 0, 0
	}

	start = l.offsetIdx
	visibleLines := l.height
	end = start

	for i := start; i < len(l.items) && visibleLines > 0; i++ {
		_, h := l.getItem(i)
		if i == start {
			// first visible item - subtract hidden lines
			h -= l.offsetLine
		}
		visibleLines -= h
		if i < len(l.items)-1 && h > 0 {
			visibleLines -= l.gap
		}
		end = i + 1
	}

	return start, end
}

// Render renders only the visible items within the viewport.
func (l *List) Render() string {
	if len(l.items) == 0 || l.height <= 0 || l.width <= 0 {
		return ""
	}

	var result []string
	linesRendered := 0

	for i := l.offsetIdx; i < len(l.items) && linesRendered < l.height; i++ {
		content, h := l.getItem(i)
		if h == 0 {
			continue
		}

		lines := strings.Split(content, "\n")

		startLine := 0
		if i == l.offsetIdx {
			// first visible item - skip hidden lines
			startLine = l.offsetLine
		}

		for j := startLine; j < len(lines) && linesRendered < l.height; j++ {
			result = append(result, lines[j])
			linesRendered++
		}

		// add gap between items (if not last item and there's room)
		if i < len(l.items)-1 && linesRendered < l.height {
			for range min(l.gap, l.height-linesRendered) {
				result = append(result, "")
				linesRendered++
			}
		}
	}

	// pad to fill viewport if content is shorter
	for linesRendered < l.height {
		result = append(result, "")
		linesRendered++
	}

	return strings.Join(result, "\n")
}

// RenderCentered renders the list centered horizontally within the given width.
func (l *List) RenderCentered(totalWidth int) string {
	content := l.Render()
	if totalWidth <= l.width {
		return content
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < totalWidth {
			padding := (totalWidth - lineWidth) / 2
			lines[i] = strings.Repeat(" ", padding) + line
		}
	}
	return strings.Join(lines, "\n")
}

// OffsetIdx returns the current first visible item index.
func (l *List) OffsetIdx() int {
	return l.offsetIdx
}

// OffsetLine returns the number of hidden lines in the first visible item.
func (l *List) OffsetLine() int {
	return l.offsetLine
}

// ItemCount returns the number of items in the list.
func (l *List) ItemCount() int {
	return len(l.items)
}

// InvalidateCache clears the render cache for all items.
func (l *List) InvalidateCache() {
	l.cache = make([]renderedItem, len(l.items))
}
