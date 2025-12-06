package length120

// MaxLineLen: 120 (default)

// Функция чуть больше 120 - не должна схлопываться (122 символа)
func ProcessWithMultipleParameters(
	input string,
	output string,
	options map[string]any,
	callback func(string) error,
) error {
	return callback(input)
}

// Функция ровно 120 символов - уже в одну строку
func ExactlyOneTwenty(a1 int, a2 int, a3 int, a4 int, a5 int, a6 int, a7 int, a8 int, a9 int, a10 int) int {
	return 0
}

// Функция больше 120 символов - не должна схлопываться
func VeryLongFunctionName(
	parameterWithVeryLongName1 string,
	parameterWithVeryLongName2 string,
	parameterWithVeryLongName3 string,
	parameterWithVeryLongName4 string,
) error {
	return nil
}

// Сложные дженерики - должна схлопнуться (< 120)
func MapWithConstraints[
	T comparable,
	R any,
](
	items []T,
	mapper func(T) R,
) []R { // want "Signature fits in one line"
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = mapper(item)
	}
	return result
}

// Метод с именованными возвращаемыми значениями
type Handler struct{}

func (h *Handler) Handle(
	request string,
	metadata map[string]string,
) (response string, err error) { // want "Signature fits in one line"
	return request, nil
}

// Интерфейс с множественными методами
type ComplexService interface {
	Process(
		ctx any,
		data []byte,
	) ([]byte, error) // want "Signature fits in one line"

	Validate(input string) error
}
