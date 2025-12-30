//go:build !release

package xslog

func DefaultLevel() Level {
	return LevelDebug
}
