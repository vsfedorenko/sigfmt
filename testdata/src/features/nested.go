package features

import "context"

// Function with nested function literals.
func OuterWithNested(
	ctx context.Context,
) error { // want "Signature can be formatted more compactly"
	// Nested function literal that can be collapsed.
	handler := func(
		a int,
		b string,
	) error { // want "Signature can be formatted more compactly"
		return nil
	}

	// Another nested function.
	process := func(
		x int,
	) int { // want "Signature can be formatted more compactly"
		return x * 2
	}

	_ = handler
	_ = process
	return nil
}

// Function returning a function.
func FunctionReturningFunction(
	multiplier int,
) func(int) int { // want "Signature can be formatted more compactly"
	return func(
		x int,
	) int { // want "Signature can be formatted more compactly"
		return x * multiplier
	}
}

// Closure with captured variables.
func ClosureExample() {
	counter := 0
	increment := func(
		delta int,
	) int { // want "Signature can be formatted more compactly"
		counter += delta
		return counter
	}
	_ = increment
}

// Nested function with long parameters that should be reformatted.
func ComplexNested() {
	handler := func(
		paramOne string, paramTwo int, paramThree bool,
	) error { // want "Signature can be formatted more compactly"
		return nil
	}
	_ = handler
}
