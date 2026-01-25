package braille

// Is returns true if the rune is a braille character.
func Is(r rune) bool {
	return r >= 0x2800 && r <= 0x28FF
}
