package display

import "fmt"

func PadRight(val string) string {
	return fmt.Sprintf("%-7s", val)
}
