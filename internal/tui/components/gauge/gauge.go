package gauge

import (
	"fmt"
	"image/color"
	"strings"
	"unicode"

	drawille "github.com/exrook/drawille-go"

	"charm.land/lipgloss/v2"

	"github.com/garrettladley/thoop/internal/tui/theme"
)

const (
	// gauge dimensions in braille dots (2 dots per char width, 4 dots per char height)
	// large enough to have hollow center for the value text
	gaugeDotsWidth  = 52 // 26 chars wide
	gaugeDotsHeight = 52 // 13 chars tall
)

// Gauge represents a circular progress gauge with a value displayed in the center.
type Gauge struct {
	Value     *float64 // Current value (nil = no data)
	Max       float64  // Maximum value (100 for %, 21 for strain)
	Label     string
	Color     color.Color // Arc fill color
	BgColor   color.Color // Background arc color (unfilled portion)
	TextColor color.Color // Value text color
}

type Option func(*Gauge)

func WithBgColor(c color.Color) Option {
	return func(g *Gauge) {
		g.BgColor = c
	}
}

func WithTextColor(c color.Color) Option {
	return func(g *Gauge) {
		g.TextColor = c
	}
}

func New(value *float64, max float64, label string, c color.Color, opts ...Option) Gauge {
	g := Gauge{
		Value:     value,
		Max:       max,
		Label:     label,
		Color:     c,
		BgColor:   theme.ColorBgLight,
		TextColor: theme.ColorWhite,
	}
	for _, opt := range opts {
		opt(&g)
	}
	return g
}

func (g Gauge) Render() string {
	canvas := drawille.NewCanvas()

	var (
		centerX = float64(gaugeDotsWidth) / 2
		centerY = float64(gaugeDotsHeight) / 2
		radius  = float64(gaugeDotsWidth)/2 - 1
	)

	var percentage float64
	if g.Value != nil && g.Max > 0 {
		percentage = *g.Value / g.Max
		if percentage > 1 {
			percentage = 1
		}
		if percentage < 0 {
			percentage = 0
		}
	}

	// draw background arc (full arc sweep in dim color)
	drawFullArc(&canvas, centerX, centerY, radius)
	bgArcStr := getCanvasString(&canvas, gaugeDotsWidth, gaugeDotsHeight)

	// clear and draw filled arc (clockwise from start by percentage of sweep)
	canvas.Clear()
	if percentage > 0 {
		drawFilledArc(&canvas, centerX, centerY, radius, percentage)
	}
	filledArcStr := getCanvasString(&canvas, gaugeDotsWidth, gaugeDotsHeight)

	// combine arcs with colors
	combinedArc := overlayArcsRaw(bgArcStr, filledArcStr, g.BgColor, g.Color)

	var valueStr string
	if g.Value == nil {
		valueStr = "--"
	} else if g.Max == 100 {
		valueStr = fmt.Sprintf("%.0f%%", *g.Value)
	} else {
		valueStr = fmt.Sprintf("%.1f", *g.Value)
	}

	valueStyle := lipgloss.NewStyle().
		Foreground(g.TextColor).
		Bold(true)

	var (
		arcHeight = lipgloss.Height(combinedArc)
		arcWidth  = lipgloss.Width(combinedArc)
	)

	styledValue := valueStyle.Render(valueStr)
	centeredValue := lipgloss.Place(
		arcWidth,
		arcHeight,
		lipgloss.Center,
		lipgloss.Center,
		styledValue,
	)

	combined := overlayWithBackground(combinedArc, centeredValue)

	labelStyle := lipgloss.NewStyle().
		Foreground(g.TextColor).
		Bold(true).
		Width(arcWidth).
		Align(lipgloss.Center)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		combined,
		labelStyle.Render(g.Label),
	)
}

// getCanvasString extracts the canvas as a string with consistent dimensions.
func getCanvasString(canvas *drawille.Canvas, width, height int) string {
	// canvas uses Frame(minX, minY, maxX, maxY)
	// each braille char is 2 dots wide, 4 dots tall
	charWidth := width / 2
	charHeight := height / 4

	rows := canvas.Rows(0, 0, width, height)

	// ensure we have exactly the right number of rows
	var lines []string
	for i := range charHeight {
		if i < len(rows) {
			// pad or truncate to exact width
			line := rows[i]
			runeCount := len([]rune(line))
			if runeCount < charWidth {
				line += strings.Repeat(" ", charWidth-runeCount)
			} else if runeCount > charWidth {
				line = string([]rune(line)[:charWidth])
			}
			lines = append(lines, line)
		} else {
			lines = append(lines, strings.Repeat(" ", charWidth))
		}
	}

	return strings.Join(lines, "\n")
}

const (
	emptyBraille rune = '\u2800'
	ansiEscape   rune = '\x1b'
)

// overlayArcsRaw combines background and filled arcs with their respective colors.
// for braille characters, we combine dots (OR them together) so filled arc adds to background.
func overlayArcsRaw(bgStr, fillStr string, bgColor, fillColor color.Color) string {
	var (
		bgLines   = strings.Split(bgStr, "\n")
		fillLines = strings.Split(fillStr, "\n")
		result    []string
		bgStyle   = lipgloss.NewStyle().Foreground(bgColor)
		fillStyle = lipgloss.NewStyle().Foreground(fillColor)
	)

	for i := range len(bgLines) {
		bgRunes := []rune(bgLines[i])
		var fillRunes []rune
		if i < len(fillLines) {
			fillRunes = []rune(fillLines[i])
		}

		var lineBuilder strings.Builder
		for j := range len(bgRunes) {
			bgChar := bgRunes[j]
			fillChar := ' '
			if j < len(fillRunes) {
				fillChar = fillRunes[j]
			}

			bgIsBraille := isBraille(bgChar)
			// only consider fill as having content if it has actual dots (not empty braille)
			fillHasDots := isBraille(fillChar) && fillChar != emptyBraille

			if fillHasDots && bgIsBraille {
				// combine braille dots: filled arc on top of background arc
				combined := combineBraille(bgChar, fillChar)
				// use fill color for the combined result (filled takes precedence visually)
				lineBuilder.WriteString(fillStyle.Render(string(combined)))
			} else if fillHasDots {
				lineBuilder.WriteString(fillStyle.Render(string(fillChar)))
			} else if bgIsBraille {
				lineBuilder.WriteString(bgStyle.Render(string(bgChar)))
			} else {
				lineBuilder.WriteRune(' ')
			}
		}
		result = append(result, lineBuilder.String())
	}

	return strings.Join(result, "\n")
}

// isBraille returns true if the rune is a braille character (U+2800 to U+28FF)
func isBraille(r rune) bool {
	return r >= 0x2800 && r <= 0x28FF
}

// combineBraille ORs the dots of two braille characters together
func combineBraille(a, b rune) rune {
	// braille characters are U+2800 + dot pattern
	// OR the patterns together to combine dots
	patternA := a - emptyBraille
	patternB := b - emptyBraille
	return emptyBraille + (patternA | patternB)
}

// overlayWithBackground overlays foreground on background.
// the foreground text has a solid background that covers the background content.
// we preserve the arc on either side of the centered value.
func overlayWithBackground(background, foreground string) string {
	var (
		bgLines  = strings.Split(background, "\n")
		fgLines  = strings.Split(foreground, "\n")
		maxLines = max(len(bgLines), len(fgLines))
		result   = make([]string, maxLines)
	)

	for i := range maxLines {
		var (
			bgLine string
			fgLine string
		)
		if i < len(bgLines) {
			bgLine = bgLines[i]
		}
		if i < len(fgLines) {
			fgLine = fgLines[i]
		}

		// check if foreground line has any visible content
		fgVisible := stripAnsi(fgLine)
		fgStart := -1
		fgEnd := -1
		for idx, r := range []rune(fgVisible) {
			if r != ' ' {
				if fgStart == -1 {
					fgStart = idx
				}
				fgEnd = idx + 1
			}
		}

		if fgStart == -1 {
			// no visible foreground content, use background
			result[i] = bgLine
			continue
		}

		// extract the styled foreground content from fgStart to fgEnd (preserve spacing)
		fgContent := extractStyledSegment(fgLine, fgStart, fgEnd)

		// build result: bg left + fg center + bg right
		// for background, we need to preserve its ANSI styling
		// but bgLine has per-character ANSI, so we'll build segment by segment
		bgVisible := stripAnsi(bgLine)
		bgRunes := []rune(bgVisible)

		var lineBuilder strings.Builder

		// left portion of background (before foreground starts)
		leftEnd := min(fgStart, len(bgRunes))

		// re-render left portion with original colors by extracting styled segments
		lineBuilder.WriteString(extractStyledSegment(bgLine, 0, leftEnd))

		// pad if needed
		for j := len(bgRunes); j < fgStart; j++ {
			lineBuilder.WriteRune(' ')
		}

		// foreground content (with its ANSI codes and original spacing)
		lineBuilder.WriteString(fgContent)

		// right portion of background (after foreground ends)
		if fgEnd < len(bgRunes) {
			lineBuilder.WriteString(extractStyledSegment(bgLine, fgEnd, len(bgRunes)))
		}

		result[i] = lineBuilder.String()
	}

	return strings.Join(result, "\n")
}

// extractStyledSegment extracts characters from start to end position from a styled string,
// preserving the ANSI styling for those characters.
func extractStyledSegment(styledStr string, start, end int) string {
	var (
		result         strings.Builder
		visibleIdx     = 0
		inEscape       = false
		pendingEscapes strings.Builder // accumulates ALL escapes between characters
	)

	for _, r := range styledStr {
		if r == ansiEscape {
			inEscape = true
			pendingEscapes.WriteRune(r)
			continue
		}

		if inEscape {
			pendingEscapes.WriteRune(r)
			if unicode.IsLetter(r) {
				inEscape = false
			}
			continue
		}

		// regular character
		if visibleIdx >= start && visibleIdx < end {
			// include all escape sequences since last character
			if pendingEscapes.Len() > 0 {
				result.WriteString(pendingEscapes.String())
			}
			result.WriteRune(r)
		}
		// clear accumulated escapes after each character
		pendingEscapes.Reset()
		visibleIdx++
	}

	return result.String()
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var (
		result   strings.Builder
		inEscape = false
	)

	for _, r := range s {
		if r == ansiEscape {
			inEscape = true
			continue
		}
		if inEscape {
			if unicode.IsLetter(r) {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
