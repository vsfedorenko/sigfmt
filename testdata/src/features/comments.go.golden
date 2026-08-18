package features

// --- Comments inside Signature ---

// FunctionWithComments checks a function with comments inside the signature.
// Linter should collapse this function since it fits on one line.
// Signatures with internal comments are left untouched (comment preservation).
func FunctionWithComments(
	a int, // parameter a
	b int, // parameter b
) {
}

// InterfaceWithComments checks interface with comments.
// Since the linter works at the text replacement level, it will remove comments when collapsing.
// In this case the signature is short, so the linter will suggest "Signature can be formatted more compactly".
// And the fix will be applied (removing comments).
type InterfaceWithComments interface {
	Method(
		a int, // comment for a
		b int, // comment for b
	)
}
