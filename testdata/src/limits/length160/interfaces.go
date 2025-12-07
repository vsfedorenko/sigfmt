package length160

import "context"

// Simple interface with a method that should be collapsed.
type MyInterface interface {
	Method(
		ctx context.Context,
	) error // want "Signature can be formatted more compactly"

	// Method already on one line
	OneLineMethod()
}

// Interface with multiple methods.
type ComplexInterface interface {
	// Short method - should be collapsed
	Get(
		id string,
	) error // want "Signature can be formatted more compactly"

	// Long method - should remain split
	ProcessWithVeryLongNameAndManyParameters(
		parameterOne string,
		parameterTwo string,
		parameterThree string,
		parameterFour string,
	) error // want "Signature can be formatted more compactly"

	// Method with context
	ProcessWithContext(
		ctx context.Context,
		parameterOne string,
		parameterTwo string,
		parameterThree string,
		parameterFour string,
	) error // want "Signature can be formatted more compactly"

	// Very many parameters
	ProcessManyParams(
		parameterOne string,
		parameterTwo string,
		parameterThree string,
		parameterFour string,
		parameterFive string,
		parameterSix string,
		parameterSeven string,
		parameterEight string,
	) error // want "Signature can be formatted more compactly"

	// Method with multiple return values - should be collapsed
	GetMultiple(
		id string,
	) (string, error) // want "Signature can be formatted more compactly"

	// Method with named return values - should be collapsed
	GetNamed(
		id string,
	) (result string, err error) // want "Signature can be formatted more compactly"
}

// Interface with generics.
type GenericInterface[T any] interface {
	Process(
		item T,
	) error // want "Signature can be formatted more compactly"

	GetAll() []T
}

// Interface with multiple type parameters.
type MultiGenericInterface[K comparable, V any] interface {
	Get(
		key K,
	) (V, bool) // want "Signature can be formatted more compactly"

	Set(
		key K,
		value V,
	) error // want "Signature can be formatted more compactly"

	Delete(key K)
}

// Very long interface method (should not be collapsed).
type VeryLongInterface interface {
	VeryLongMethodNameThatShouldNotBeCollapsed(
		parameterWithVeryLongName string,
		anotherParameterWithVeryLongName string,
		yetAnotherParameterWithVeryLongName string,
	) error // want "Signature can be formatted more compactly"
}

// Interface with variadic parameters.
type VariadicInterface interface {
	Process(
		items ...string,
	) error // want "Signature can be formatted more compactly"
}

// Interface with functional types in parameters.
type HandlerInterface interface {
	Handle(
		ctx context.Context,
		handler func(string) error,
	) error // want "Signature can be formatted more compactly"

	HandleMultiple(ctx context.Context, handlers ...func(string) error) error
}

// Empty interface.
type EmptyInterface interface{}

// Interface with mixed parameters.
type MixedInterface interface {
	Process(
		a, b int,
		c string,
	) error // want "Signature can be formatted more compactly"
}
