package gauge

import (
	"embed"
	"image/color"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

//go:embed testdata/*.golden
var goldenFiles embed.FS

func TestExtractStyledSegment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		start    int
		end      int
		expected string
	}{
		{
			name:     "extract middle segment with ANSI codes",
			input:    "\x1b[31mA\x1b[0m\x1b[31mB\x1b[0m\x1b[31mC\x1b[0m",
			start:    0,
			end:      2,
			expected: "\x1b[31mA\x1b[0m\x1b[31mB",
		},
		{
			name:     "extract single character with ANSI",
			input:    "\x1b[31mA\x1b[0m\x1b[31mB\x1b[0m\x1b[31mC\x1b[0m",
			start:    1,
			end:      2,
			expected: "\x1b[0m\x1b[31mB",
		},
		{
			name:     "extract without ANSI codes",
			input:    "ABC",
			start:    0,
			end:      2,
			expected: "AB",
		},
		{
			name:     "extract from middle to end",
			input:    "\x1b[31mA\x1b[0m\x1b[31mB\x1b[0m\x1b[31mC\x1b[0m",
			start:    1,
			end:      3,
			expected: "\x1b[0m\x1b[31mB\x1b[0m\x1b[31mC",
		},
		{
			name:     "extract entire string",
			input:    "\x1b[31mABC\x1b[0m",
			start:    0,
			end:      3,
			expected: "\x1b[31mABC",
		},
		{
			name:     "extract empty range",
			input:    "\x1b[31mABC\x1b[0m",
			start:    1,
			end:      1,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractStyledSegment(tt.input, tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("extractStyledSegment() = %q, want %q", result, tt.expected)
				t.Logf("input: %q, start: %d, end: %d", tt.input, tt.start, tt.end)
			}
		})
	}
}

func TestStripAnsi(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove single ANSI code",
			input:    "\x1b[31mHello\x1b[0m",
			expected: "Hello",
		},
		{
			name:     "remove multiple ANSI codes",
			input:    "\x1b[31mH\x1b[0m\x1b[32me\x1b[0m\x1b[33ml\x1b[0m",
			expected: "Hel",
		},
		{
			name:     "no ANSI codes",
			input:    "Hello",
			expected: "Hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only ANSI codes",
			input:    "\x1b[31m\x1b[0m",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := stripAnsi(tt.input)
			if result != tt.expected {
				t.Errorf("stripAnsi() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestOverlayWithBackground(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		background string
		foreground string
		wantLines  int
	}{
		{
			name:       "simple overlay",
			background: "AAAAA\nBBBBB\nCCCCC",
			foreground: "     \n  X  \n     ",
			wantLines:  3,
		},
		{
			name:       "foreground smaller than background",
			background: "AAAAA\nBBBBB\nCCCCC",
			foreground: "  X  ",
			wantLines:  3,
		},
		{
			name:       "empty foreground",
			background: "AAAAA\nBBBBB\nCCCCC",
			foreground: "     \n     \n     ",
			wantLines:  3,
		},
		{
			name:       "styled background with centered foreground",
			background: "\x1b[31mAAAAA\x1b[0m\n\x1b[31mBBBBB\x1b[0m\n\x1b[31mCCCCC\x1b[0m",
			foreground: "     \n\x1b[1m  X  \x1b[0m\n     ",
			wantLines:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := overlayWithBackground(tt.background, tt.foreground)
			lines := strings.Split(result, "\n")
			if len(lines) != tt.wantLines {
				t.Errorf("overlayWithBackground() returned %d lines, want %d", len(lines), tt.wantLines)
			}
		})
	}
}

func TestOverlayArcsRaw_Coloring(t *testing.T) {
	t.Parallel()

	// helper to check if string contains ANSI color sequence
	containsColor := func(s string, colorCode string) bool {
		return strings.Contains(s, colorCode)
	}

	tests := []struct {
		name          string
		bgStr         string
		fillStr       string
		expectBgParts bool // should have bg-colored parts
		expectFill    bool // should have fill-colored parts
	}{
		{
			name:          "full fill covers background",
			bgStr:         "⣿⣿⣿",
			fillStr:       "⣿⣿⣿",
			expectBgParts: false, // all fill, no bg-only
			expectFill:    true,
		},
		{
			name:          "no fill shows background",
			bgStr:         "⣿⣿⣿",
			fillStr:       "   ",
			expectBgParts: true,
			expectFill:    false,
		},
		{
			name:          "partial fill shows both colors",
			bgStr:         "⣿⣿⣿",
			fillStr:       "⣿  ",
			expectBgParts: true,
			expectFill:    true,
		},
		{
			name:          "empty braille in fill uses bg color",
			bgStr:         "⣿⣿⣿",
			fillStr:       "⠀⠀⠀", // empty braille (U+2800)
			expectBgParts: true,
			expectFill:    false,
		},
	}

	bgColor := color.RGBA{R: 100, G: 100, B: 100, A: 255} // gray
	fillColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}   // red

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := overlayArcsRaw(tt.bgStr, tt.fillStr, bgColor, fillColor)

			// result should contain ANSI escape sequences
			hasAnsi := containsColor(result, "\x1b[")
			if !hasAnsi {
				t.Error("expected ANSI color codes in output")
			}

			// count distinct escape sequences to verify different colors
			parts := strings.Split(result, "\x1b[")
			distinctCodes := make(map[string]bool)
			for _, p := range parts[1:] { // skip first empty part
				if idx := strings.Index(p, "m"); idx != -1 {
					distinctCodes[p[:idx+1]] = true
				}
			}

			// if we expect both bg and fill, we should have multiple distinct color codes
			if tt.expectBgParts && tt.expectFill {
				if len(distinctCodes) < 2 {
					t.Errorf("expected multiple distinct color codes for mixed bg/fill, got %d", len(distinctCodes))
				}
			}

			// verify visible content is preserved
			stripped := stripAnsi(result)
			expectedStripped := stripAnsi(tt.bgStr) // bg forms the base shape
			if !strings.Contains(stripped, strings.TrimSpace(expectedStripped)) && strings.TrimSpace(expectedStripped) != "" {
				t.Errorf("stripped output should contain background braille pattern")
			}
		})
	}
}

func TestGaugeRender_Golden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		value      float64
		max        float64
		label      string
		goldenFile string
	}{
		{"sleep_0", 0, 100, "SLEEP", "testdata/sleep_0.golden"},
		{"sleep_25", 25, 100, "SLEEP", "testdata/sleep_25.golden"},
		{"sleep_50", 50, 100, "SLEEP", "testdata/sleep_50.golden"},
		{"sleep_75", 75, 100, "SLEEP", "testdata/sleep_75.golden"},
		{"sleep_100", 100, 100, "SLEEP", "testdata/sleep_100.golden"},
		{"recovery_0", 0, 100, "RECOVERY", "testdata/recovery_0.golden"},
		{"recovery_25", 25, 100, "RECOVERY", "testdata/recovery_25.golden"},
		{"recovery_50", 50, 100, "RECOVERY", "testdata/recovery_50.golden"},
		{"recovery_75", 75, 100, "RECOVERY", "testdata/recovery_75.golden"},
		{"recovery_100", 100, 100, "RECOVERY", "testdata/recovery_100.golden"},
		{"strain_0.0", 0, 21, "STRAIN", "testdata/strain_0.0.golden"},
		{"strain_10.5", 10.5, 21, "STRAIN", "testdata/strain_10.5.golden"},
		{"strain_21.0", 21, 21, "STRAIN", "testdata/strain_21.0.golden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			value := tt.value
			g := New(&value, tt.max, tt.label, nil)
			result := g.Render()
			stripped := stripAnsi(result)

			golden, err := goldenFiles.ReadFile(tt.goldenFile)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}

			expected := strings.TrimSuffix(string(golden), "\n")
			if diff := cmp.Diff(normalizeLines(expected), normalizeLines(stripped)); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// normalizeLines trims trailing whitespace from each line for stable comparison
func normalizeLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}
