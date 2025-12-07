package length140

// --- Generics (Type Parameters) ---

// GenericFunction checks a simple function with generics.
func GenericFunction[T any](
	v T,
) T { // want "Signature can be formatted more compactly"
	return v
}

// LongConstraintsInterface checks interface with a very long list of types in constraint.
// Linter should try to pack method parameters but not touch the generic constraint itself.
type LongConstraintsInterface[T interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | uintptr | float32 | float64 | complex64 | complex128 | string
}] interface {
	Method(
		p1 T,
		p2 T,
		p3 T,
		p4 T,
	) // want "Signature can be formatted more compactly"
}

// MethodWithComplexGenericsInterface checks interface method with complex generics.
type MethodWithComplexGenericsInterface[T interface{ int | string | ~bool }, R interface{ M() }] interface {
	MethodWithComplexGenerics(
		a T,
		b R,
		c T,
		d R,
	) // want "Signature can be formatted more compactly"
}

// --- Receiver with Generics ---

type Container[T any] struct {
	val T
}

// MethodWithGenericReceiver checks method on type with generic.
// Should be collapsed if it fits.
func (c *Container[T]) GetVal(
	ctx interface{},
) T { // want "Signature can be formatted more compactly"
	return c.val
}
