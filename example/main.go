package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

// CheckSignature is a function with a signature that fits in one line but is split across multiple lines.
func CheckSignature(
	a int,
	b int,
) {
	fmt.Println(a, b)
}

// CheckSignature2 is a function with a signature that is too long for one line but formatter keeps it
func CheckSignature2(
	param1 string,
	param2 string,
	param3 string,
	param4 string,
	param5 string,
	param6 string,
	param7 string,
) {
	fmt.Println(param1)
}

// InterfaceMethods demonstrates interface method signatures.
type InterfaceMethods interface {
	// Should be packed: (a, b int)
	Method1(
		a int,
		b int,
	)

	// Should be packed: (a, b int, c string) or similar depending on length
	Method2(
		a int,
		b int,
		c string,
	)

	// Long method signature that stays multi-line
	Method3(
		param1 string,
		param2 string,
		param3 string,
		param4 string,
		param5 string,
		param6 string,
	)
}

// StructFields shows function types in struct fields.
type StructFields struct {
	// Callback that fits in one line
	CallbackShort func(
		a int,
		b int,
	)

	// Callback that is long
	CallbackLong func(
		p1 string,
		p2 string,
		p3 string,
		p4 string,
		p5 string,
	)
}

// GenericFunction demonstrates generics support.
func GenericFunction[T any](
	val T,
	count int,
) {
	fmt.Println(val, count)
}
