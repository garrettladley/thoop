package units

import (
	"fmt"
	"strings"
)

const kjPerKcal = 4.184

// KilojoulesToCalories converts kilojoules to kilocalories (dietary calories).
func KilojoulesToCalories(kj float64) float64 {
	return kj / kjPerKcal
}

func FormatWithCommas(n float64) string {
	intPart := int(n)
	s := fmt.Sprintf("%d", intPart)

	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
