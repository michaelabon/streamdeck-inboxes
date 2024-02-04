package display

import "fmt"

func PadRight(val string) string {
	switch len(val) {
	case 1:
		return fmt.Sprintf("%-10s", val)
	case 2, 3:
		return fmt.Sprintf("%-9s", val)
	case 4:
		return fmt.Sprintf("%-7s", val)
	}
	return fmt.Sprintf("%-7s", val)
}

func EncodeSVG(svg string) string {
	return fmt.Sprintf("data:image/svg+xml;charset=utf8,%s", svg)
}
