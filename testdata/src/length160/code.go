package length160

// MaxLineLen: 160

// Очень длинная функция - НЕ должна схлопываться (> 160)
func ProcessWithVeryManyParameters(
	inputDataStream string,
	outputFilePath string,
	configurationOptions map[string]any,
	callbackFunction func(string) error,
	metadataInformation map[string]string,
	loggingEnabled bool,
) error {
	return callbackFunction(inputDataStream)
}

// Функция ровно 160 символов - уже в одну строку
func ExactlyOneSixty(parameter1 int, parameter2 int, parameter3 int, parameter4 int, parameter5 int, parameter6 int, parameter7 int, parameter8 int) int {
	return 0
}

// Функция больше 160 символов - не должна схлопываться
func ExtremelyLongFunctionNameWithVeryManyParameters(
	firstParameterWithExtremelyLongName string,
	secondParameterWithExtremelyLongName string,
	thirdParameterWithExtremelyLongName string,
	fourthParameterWithExtremelyLongName string,
	fifthParameterWithExtremelyLongName string,
) error {
	return nil
}

// Сложные дженерики с ограничениями - НЕ должна схлопываться (> 160)
func FilterTransformAndReduce[
	TInput comparable,
	TIntermediate any,
	TOutput any,
](
	items []TInput,
	filter func(TInput) bool,
	transform func(TInput) TIntermediate,
	reduce func([]TIntermediate) TOutput,
) TOutput {
	var intermediate []TIntermediate
	for _, item := range items {
		if filter(item) {
			intermediate = append(intermediate, transform(item))
		}
	}
	return reduce(intermediate)
}

// Метод с очень длинной сигнатурой - НЕ должен схлопываться (> 160)
type ComplexDataProcessor struct{}

func (cdp *ComplexDataProcessor) ProcessValidateAndTransform(
	rawInputData []byte,
	validationRulesSet map[string]func([]byte) bool,
	transformationFunction func([]byte) []byte,
	outputFormatSpecification string,
) (processedData []byte, validationErrors []error, transformErrors []error) {
	return rawInputData, nil, nil
}

// Интерфейс с очень длинными методами
type AdvancedGenericProcessor[TInput any, TOutput any, TError error] interface {
	TransformWithContext(
		contextData any,
		inputValue TInput,
		configurationMap map[string]any,
	) (TOutput, TError) // want "Signature fits in one line"

	BatchProcessWithValidation(items []TInput, validate func(TInput) bool) ([]TOutput, []TError)
}
