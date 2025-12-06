package length140

// --- Comments inside Signature ---

// FunctionWithComments проверяет функцию с комментариями внутри сигнатуры.
// Линтер должен схлопнуть эту функцию, так как она помещается в одну строку.
// При этом комментарии внутри сигнатуры будут удалены.
func FunctionWithComments(
	a int, // parameter a
	b int, // parameter b
) { // want "Multi-line signature can be collapsed to one line"
}

// InterfaceWithComments проверяет интерфейс с комментариями.
// Так как линтер работает на уровне замены текста, он удалит комментарии при схлопывании.
// В данном случае сигнатура короткая, поэтому линтер предложит "Multi-line signature can be collapsed to one line".
// И фикс применится (удалив комментарии).
type InterfaceWithComments interface {
	Method(
		a int, // comment for a
		b int, // comment for b
	) // want "Multi-line signature can be collapsed to one line"
}
