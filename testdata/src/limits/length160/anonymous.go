package length160

// --- Anonymous Functions & Identifiers ---

// BlankShort checks blank identifiers.
func BlankShort(
	_ int,
	_ string,
) { // want "Multi-line signature can be collapsed to one line"
}

// MethodWithBlanksInterface checks blank identifiers in interface.
type MethodWithBlanksInterface interface {
	MethodWithBlanks(
		_ int,
		_ string,
		_ bool,
		_ float64,
	) // want "Multi-line signature can be collapsed to one line"
}

// OuterFunction checks nested anonymous functions.
func OuterFunction() {
	// Anonymous function inside.
	callback := func(
		a int,
		b int,
	) int { // want "Multi-line signature can be collapsed to one line"
		return a + b
	}
	_ = callback
}
