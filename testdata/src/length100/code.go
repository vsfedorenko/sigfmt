package length100

// MaxLineLen: 100

// Короткая функция - должна схлопнуться (< 100)
func ProcessData(
	input string,
	output string,
	options map[string]any,
) error { // want "Signature fits in one line"
	return nil
}

// Функция ровно 100 символов - уже в одну строку
func ExactlyOneHundred(aaaa int, bbbb int, cccc int, dddd int, eeee int, ffff int, gggg int) int {
	return 0
}

// Функция больше 100 символов - не должна схлопываться
func OverOneHundred(
	parameterWithLongName1 string,
	parameterWithLongName2 string,
	parameterWithLongName3 string,
) error {
	return nil
}

// Функция с дженериками - должна схлопнуться (< 100)
func Transform[
	T any,
	R any,
](
	input T,
	fn func(T) R,
) R { // want "Signature fits in one line"
	return fn(input)
}

// Метод с множественными возвращаемыми значениями
type Service struct{}

func (s *Service) Execute(
	ctx any,
	data string,
) (string, error) { // want "Signature fits in one line"
	return data, nil
}

// Интерфейс с дженериками
type Repository[T any] interface {
	Save(
		entity T,
	) error // want "Signature fits in one line"

	FindByID(id string) (T, error)
}
