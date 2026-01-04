//go:build !dev

package footer

func (f Footer) leftContent() string {
	return ""
}
