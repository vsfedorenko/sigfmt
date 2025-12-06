package length140

// MaxLineLen: 140

// Длинная функция - НЕ должна схлопываться (> 140)
func ProcessWithManyParameters(
	inputData string,
	outputPath string,
	configOptions map[string]any,
	callbackFunction func(string) error,
	metadata map[string]string,
) error {
	return callbackFunction(inputData)
}

// Функция ровно 140 символов - уже в одну строку
func ExactlyOneFourty(param1 int, param2 int, param3 int, param4 int, param5 int, param6 int, param7 int, param8 int, param9 int) int {
	return 0
}

// Функция больше 140 символов - не должна схлопываться
func VeryLongFunctionNameWithManyParameters(
	firstParameterWithVeryLongName string,
	secondParameterWithVeryLongName string,
	thirdParameterWithVeryLongName string,
	fourthParameterWithVeryLongName string,
) error {
	return nil
}

// Дженерики с ограничениями - должна схлопнуться (< 140)
func FilterAndTransform[
	T comparable,
	R any,
](
	items []T,
	predicate func(T) bool,
	transformer func(T) R,
) []R { // want "Signature fits in one line"
	var result []R
	for _, item := range items {
		if predicate(item) {
			result = append(result, transformer(item))
		}
	}
	return result
}

// Метод с receiver и многими параметрами - НЕ должен схлопываться (> 140)
type DataProcessor struct{}

func (dp *DataProcessor) ProcessAndValidate(
	rawData []byte,
	validationRules map[string]func([]byte) bool,
	outputFormat string,
) (processedData []byte, validationErrors []error) {
	return rawData, nil
}

// Интерфейс с generic методами
type GenericProcessor[T any, R any] interface {
	Transform(
		input T,
		config map[string]any,
	) (R, error) // want "Signature fits in one line"

	Batch(items []T) ([]R, error)
}
