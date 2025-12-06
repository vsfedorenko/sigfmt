package length160

// --- Inline Structs & Complex Types ---

// InlineStructShort проверяет встроенную структуру, которая влезает в одну строку.
func InlineStructShort(
	s struct{ X, Y int },
) {} // want "Signature fits in one line"

// InlineStructLong проверяет встроенную структуру, которая НЕ влезает в одну строку (из-за имен).
// Должна остаться без изменений (multiline), так как это обычная функция.
func InlineStructLong(
	param1 struct{ Field1, Field2, Field3 int },
	param2 struct{ A, B, C string },
	param3 map[string]func(int) error,
	param4 interface{ M() },
) { // want "Signature fits in one line"
}

// MethodWithInlineStructsInterface проверяет метод интерфейса с встроенными структурами.
// Должен переформатироваться (упаковаться).
type MethodWithInlineStructsInterface interface {
	MethodWithInlineStructs(
		p1 struct{ A, B int },
		p2 struct{ C, D string },
		p3 struct{ E, F bool },
		p4 struct{ G, H float64 },
	) // want "Signature fits in one line"
}

// NestedTypeShort проверяет вложенные типы, влезающие в одну строку.
func NestedTypeShort(
	m map[string]func(int) []byte,
) {} // want "Signature fits in one line"

// FuncParamShort проверяет функцию как параметр.
func FuncParamShort(
	handler func(string) (int, error),
) {} // want "Signature fits in one line"

// VariadicComplex проверяет вариадический параметр сложного типа.
func VariadicComplex(
	items ...struct{ ID string },
) {} // want "Signature fits in one line"

// ComplexStructField проверяет поле-функцию со сложными типами.
type ComplexStructField struct {
	ComplexField func(
		a map[string]struct{ X int },
		b interface{ String() string },
		c chan<- func(),
		d ...*int,
	) // want "Signature fits in one line"
}
