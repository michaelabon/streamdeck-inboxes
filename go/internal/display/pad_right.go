package display

import "fmt"

// PadRight uses the length of the string val to determine
// how many space characters are needed to pad it.
// This happens because we're trying to get the number
// in a specific spot in the upper-left corner of the icon,
// but numeral characters have more width than space characters.
func PadRight(val string) string {
	switch len(val) {
	case 1:
		return fmt.Sprintf("%-10s", val)
	case 2, 3:
		return fmt.Sprintf("%-9s", val)
	case 4:
		return fmt.Sprintf("%-7s", val)
	default:
		return fmt.Sprintf("%-7s", val)
	}
}
