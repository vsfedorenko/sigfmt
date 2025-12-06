package length140

import "context"

// Простой интерфейс с методом, который должен схлопнуться.
type MyInterface interface {
	Method(
		ctx context.Context,
	) error // want "Multi-line signature can be collapsed to one line"

	// Метод уже в одну строку
	OneLineMethod()
}

// Интерфейс с несколькими методами.
type ComplexInterface interface {
	// Короткий метод - должен схлопнуться
	Get(
		id string,
	) error // want "Multi-line signature can be collapsed to one line"

	// Длинный метод - должен остаться разбитым
	ProcessWithVeryLongNameAndManyParameters(
		parameterOne string,
		parameterTwo string,
		parameterThree string,
		parameterFour string,
	) error // want "Multi-line signature can be collapsed to one line"

	// Метод с контекстом
	ProcessWithContext(
		ctx context.Context,
		parameterOne string,
		parameterTwo string,
		parameterThree string,
		parameterFour string,
	) error // want "Multi-line signature can be collapsed to one line"

	// Очень много параметров
	ProcessManyParams(
		parameterOne string,
		parameterTwo string,
		parameterThree string,
		parameterFour string,
		parameterFive string,
		parameterSix string,
		parameterSeven string,
		parameterEight string,
	) error // want "Signature can be reformatted more compactly"

	// Метод с несколькими возвращаемыми значениями - должен схлопнуться
	GetMultiple(
		id string,
	) (string, error) // want "Multi-line signature can be collapsed to one line"

	// Метод с именованными возвращаемыми значениями - должен схлопнуться
	GetNamed(
		id string,
	) (result string, err error) // want "Multi-line signature can be collapsed to one line"
}

// Интерфейс с дженериками.
type GenericInterface[T any] interface {
	Process(
		item T,
	) error // want "Multi-line signature can be collapsed to one line"

	GetAll() []T
}

// Интерфейс с несколькими type parameters.
type MultiGenericInterface[K comparable, V any] interface {
	Get(
		key K,
	) (V, bool) // want "Multi-line signature can be collapsed to one line"

	Set(
		key K,
		value V,
	) error // want "Multi-line signature can be collapsed to one line"

	Delete(key K)
}

// Очень длинный метод интерфейса (не должен схлопываться).
type VeryLongInterface interface {
	VeryLongMethodNameThatShouldNotBeCollapsed(
		parameterWithVeryLongName string,
		anotherParameterWithVeryLongName string,
		yetAnotherParameterWithVeryLongName string,
	) error // want "Signature can be reformatted more compactly"
}

// Интерфейс с вариадическими параметрами.
type VariadicInterface interface {
	Process(
		items ...string,
	) error // want "Multi-line signature can be collapsed to one line"
}

// Интерфейс с функциональными типами в параметрах.
type HandlerInterface interface {
	Handle(
		ctx context.Context,
		handler func(string) error,
	) error // want "Multi-line signature can be collapsed to one line"

	HandleMultiple(ctx context.Context, handlers ...func(string) error) error
}

// Пустой интерфейс.
type EmptyInterface interface{}

// Интерфейс с mixed параметрами.
type MixedInterface interface {
	Process(
		a, b int,
		c string,
	) error // want "Multi-line signature can be collapsed to one line"
}
