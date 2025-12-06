package features

import "context"

// Простая структура с полем типа func, которое должно схлопнуться.
type Handler struct {
	Process func(
		ctx context.Context,
	) error // want "Signature fits in one line"
}

// Структура с несколькими полями типа func.
type MultiHandler struct {
	// Короткое поле - должно схлопнуться
	OnStart func(
		id string,
	) error // want "Signature fits in one line"

	// Поле уже в одну строку
	OnStop func() error

	// Длинное поле - должно остаться разбитым
	OnProcessWithVeryLongNameAndManyParameters func(
		parameterOne string,
		parameterTwo string,
		parameterThree string,
		parameterFour string,
	) error // want "Signature can be formatted better"

	// Поле с несколькими возвращаемыми значениями - должно схлопнуться
	GetData func(
		key string,
	) (string, error) // want "Signature fits in one line"
}

// Структура с полями типа func с дженериками.
type GenericHandler[T any] struct {
	Process func(
		item T,
	) error // want "Signature fits in one line"

	Transform func(item T) T
}

// Структура с обычными полями и полями типа func.
type MixedStruct struct {
	Name string
	Age  int

	Validate func(
		value string,
	) bool // want "Signature fits in one line"

	Transform func(input string) string
}

// Структура с вариадическими параметрами в поле типа func.
type VariadicHandler struct {
	Process func(
		items ...string,
	) error // want "Signature fits in one line"
}

// Структура с полем типа func, имеющим именованные возвращаемые значения.
type NamedReturnsHandler struct {
	Process func(
		id string,
	) (result string, err error) // want "Signature fits in one line"
}

// Структура с полем типа func с mixed параметрами.
type MixedParamsHandler struct {
	Process func(
		a, b int,
		c string,
	) error // want "Signature fits in one line"
}

// Сложный кейс: структура объявлена внутри функции,
// а внутри структуры поле с многострочной функцией.
func ComplexCase() {
	type LocalStruct struct {
		// Это поле должно схлопнуться
		Handler func(
			ctx context.Context,
			id string,
		) error // want "Signature fits in one line"

		// Это поле уже в одну строку
		Simple func() error

		// Это длинное поле должно остаться разбитым
		VeryLongHandler func(
			parameterWithVeryLongName string,
			anotherParameterWithVeryLongName string,
			yetAnotherParameterWithVeryLongName string,
		) error // want "Signature can be formatted better"
	}
	_ = LocalStruct{}
}

// Еще один сложный кейс: вложенные структуры.
func NestedStructCase() {
	type Outer struct {
		Inner struct {
			// Поле во вложенной структуре (анонимная структура не поддерживается линтером)
			Process func(
				data string,
			) error
		}

		// Поле в основной структуре - должно схлопнуться
		Transform func(
			input string,
		) string // want "Signature fits in one line"
	}

	_ = Outer{}
}

// Анонимная структура в переменной с полем типа func.
// (анонимные структуры не поддерживаются линтером)
var anonymousStruct = struct {
	Handler func(
		id string,
	) error

	Simple func() error
}{
	Handler: func(id string) error { return nil },
	Simple:  func() error { return nil },
}

// Структура с полем типа func, возвращающим функцию.
type HigherOrderHandler struct {
	GetHandler func(
		config string,
	) func(string) error // want "Signature fits in one line"
}

// Структура с полем типа func, принимающим функцию в параметрах.
type CallbackHandler struct {
	Process func(
		callback func(string) error,
	) error // want "Signature fits in one line"

	ProcessMultiple func(callback func(string) error, fallback func() error) error
}
