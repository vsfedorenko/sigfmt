package pack_interfaces

// MyStruct should NOT be packed.
type MyStruct struct {
	Field1 func(
		param1VeryLongName int,
		param2VeryLongName int,
		param3VeryLongName int,
		param4VeryLongName int,
		param5VeryLongName int,
	) error // The linter should not touch this (PackStructFields: false)
}

// MyInterface should be packed.
type MyInterface interface {
	Method1(
		param1VeryLongName int,
		param2VeryLongName int,
		param3VeryLongName int,
		param4VeryLongName int,
		param5VeryLongName int,
	) error // want "Signature can be formatted more compactly"
}
