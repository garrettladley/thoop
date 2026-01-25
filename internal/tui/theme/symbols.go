package theme

// Directional symbols
const (
	SymbolArrowUp   = "▲"
	SymbolArrowDown = "▼"
)

// Shape symbols
const (
	SymbolCircleFilled = "●"
	SymbolCircleEmpty  = "○"
	SymbolSquareFilled = "■"
)

// Block characters for progress bars
const (
	SymbolBlockFull = "█"
)

// Vertical block characters (8 levels from empty to full)
var VerticalBlocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Horizontal block characters (8 levels from empty to full)
var HorizontalBlocks = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}
