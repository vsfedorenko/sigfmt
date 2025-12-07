package no_pack

// Test packing logic with disabled packing settings.

// MyStruct has fields that should NOT be packed if packing is disabled.
type MyStruct struct {
	// Should NOT be packed if PackStructFields is false
	// We use LONG signature so it doesn't collapse to one line.
	// And since packing is false, it should remain as is (multi-line).
	Field1 func(
		param1VeryLongName int,
		param2VeryLongName int,
		param3VeryLongName int,
		param4VeryLongName int,
		param5VeryLongName int,
	)

	Field2 func(
		param1VeryLongName int,
		param2VeryLongName int,
		param3VeryLongName int,
		param4VeryLongName int,
		param5VeryLongName int,
	)
}

// Interface method packing
type MyInterface interface {
	// Should NOT be packed if PackInterfaceMethods is false
	Method1(
		param1VeryLongName int,
		param2VeryLongName int,
		param3VeryLongName int,
		param4VeryLongName int,
		param5VeryLongName int,
	)

	Method2(
		param1VeryLongName int,
		param2VeryLongName int,
		param3VeryLongName int,
		param4VeryLongName int,
		param5VeryLongName int,
	)
}
