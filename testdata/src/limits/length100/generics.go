package length100

// --- Generics (Type Parameters) ---

// GenericFunction проверяет простую функцию с дженериками.
func GenericFunction[T any](
	v T,
) T { // want "Multi-line signature can be collapsed to one line"
	return v
}

// LongConstraintsInterface проверяет интерфейс с очень длинным списком типов в ограничении.
// Линтер должен пытаться упаковать параметры метода, но не трогать само ограничение дженерика.
type LongConstraintsInterface[T interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | uintptr | float32 | float64 | complex64 | complex128 | string
}] interface {
	Method(
		p1 T,
		p2 T,
		p3 T,
		p4 T,
	) // want "Multi-line signature can be collapsed to one line"
}

// MethodWithComplexGenericsInterface проверяет метод интерфейса со сложными дженериками.
type MethodWithComplexGenericsInterface[T interface{ int | string | ~bool }, R interface{ M() }] interface {
	MethodWithComplexGenerics(
		a T,
		b R,
		c T,
		d R,
	) // want "Multi-line signature can be collapsed to one line"
}

// --- Receiver with Generics ---

type Container[T any] struct {
	val T
}

// MethodWithGenericReceiver проверяет метод на типе с дженериком.
// Должен схлопнуться, если влазит.
func (c *Container[T]) GetVal(
	ctx interface{},
) T { // want "Multi-line signature can be collapsed to one line"
	return c.val
}
