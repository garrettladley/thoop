//go:build release

package footer

// leftContent returns empty string for release builds - no version in footer.
func (f Footer) leftContent() string {
	return ""
}
