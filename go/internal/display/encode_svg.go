package display

func EncodeSVG(svg string) string {
	return "data:image/svg+xml;charset=utf8," + svg
}
