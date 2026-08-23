package length60exact

// FuncFit: sig length = 60 -> must collapse (inclusive limit).
func FuncFit(
	xxxxxxxxxxxxxxxxxxxxxxxxxxx int,
	second string,
) { // want "Signature can be formatted more compactly"
	_ = second
}

// FuncOver: sig length = 61 -> must stay multi-line.
func FuncOver(
	xxxxxxxxxxxxxxxxxxxxxxxxxxx int,
	second string,
) {
	_ = second
}
