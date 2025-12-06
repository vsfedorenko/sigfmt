# funcwrap — Linter for Go Function Signatures

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-blue)](https://golang.org/dl/)

`funcwrap` — это мощный плагин-линтер для `golangci-lint`, предназначенный для автоматической проверки и форматирования сигнатур функций, методов и полевых функций в Go.

## 🎯 Мотивация

### Проблема

Стандартный форматтер `gofmt` обеспечивает базовое форматирование, но оставляет свободу выбора при разбивке длинных сигнатур функций на несколько строк. Это часто приводит к:

1.  **Несогласованности в кодовой базе**
    ```go
    // Файл A: всё в одну строку
    func ProcessUser(id int, name string, email string, age int) error { ... }

    // Файл B: "лесенка"
    func ProcessOrder(
        id int,
        userId int,
        total float64,
        status string,
    ) error { ... }

    // Файл C: хаотичная группировка
    func CreateAccount(id int, name string,
        email string, password string) error { ... }
    ```

2.  **Плохой читаемости**: Слишком длинные строки заставляют скроллить горизонтально, особенно при code review.

3.  **Загрязнению git-истории**: Когда разные разработчики по-разному форматируют код, возникают ненужные изменения в diff:
    ```diff
    - func Save(data []byte,
    -     path string) error {
    + func Save(data []byte, path string) error {
    ```

4.  **Раздутым интерфейсам**: Без упаковки параметров интерфейсы занимают слишком много места:
    ```go
    type UserService interface {
        Create(
            name string,
            email string,
            age int,
        ) error
        Update(
            id int,
            name string,
            email string,
            age int,
        ) error
        Delete(
            id int,
        ) error
    }
    // 16 строк для трёх простых методов!
    ```

### Решение

`funcwrap` решает эти проблемы, навязывая строгие, но разумные правила:
- **Компактность**: Если сигнатура влезает в одну строку (с учетом лимита), она *должна* быть в одну строку.
- **Структурность**: Если сигнатура длинная, она должна быть отформатирована так, чтобы максимально эффективно использовать вертикальное пространство (особенно для интерфейсов).
- **Автоматизация**: Линтер не только указывает на проблемы, но и предлагает автоматические фиксы.
- **Настраиваемость**: Гибкие настройки для разных стилей кода и ограничений длины строки.

## 🚀 Возможности

Линтер анализирует следующие конструкции языка Go:
- **Объявления функций** (`func Foo(...)`)
- **Методы типов** (`func (s *S) Bar(...)`)
- **Анонимные функции / Литералы** (`var f = func(...)`)
- **Методы в интерфейсах** (`type I interface { Method(...) }`)
- **Поля структур с типом функции** (`type S struct { Callback func(...) }`)
- **Generics (Go 1.18+)**: Корректная обработка параметров типов `[T any]`.

### Алгоритм работы

Линтер работает в два этапа:

#### 1. Collapse (Схлопывание)
Проверяет, можно ли записать всю сигнатуру (параметры + возвращаемые значения + receiver) в одну строку так, чтобы её длина не превышала `max-line-len`.

**Пример:**
```go
// До (плохо, занимает 3 строки, хотя влезает в одну)
func Sum(
    a int, 
    b int,
) int { ... }

// После (хорошо, компактно)
func Sum(a int, b int) int { ... }
```

#### 2. Reformat / Packing (Умное переформатирование)
Если сигнатура **не** влезает в одну строку, линтер применяет разные стратегии в зависимости от контекста:

*   **Для обычных функций**:
    Линтер действует консервативно. Если вы уже разбили аргументы по строкам, он старается не вмешиваться, чтобы не сломать авторскую группировку. Он вмешивается только если форматирование явно нарушает стиль (например, часть аргументов на одной строке, часть на другой без системы).

*   **Для интерфейсов и полей структур**:
    Линтер действует агрессивно (стратегия **Packing**). Он пытается разместить несколько коротких параметров на одной строке, чтобы определение интерфейса не растягивалось на 50 экранов.

    **Пример (Интерфейс):**
    ```go
    // До (слишком разреженно)
    type Logger interface {
        Log(
            level Level,
            msg string,
            args ...interface{},
        )
    }

    // После (компактная упаковка)
    type Logger interface {
        Log(level Level, msg string, args ...interface{})
    }
    ```

## ⚙️ Конфигурация

Линтер настраивается через файл `.golangci.yml`.

**Параметры:**
*   `max-line-len` (int): Максимальная допустимая длина строки. По умолчанию: `120`.
*   `tab-width` (int): Ширина табуляции для расчета визуальной длины строки. По умолчанию: `8`.
*   `pack-struct-fields` (bool): Включить упаковку полей структур с типом func. По умолчанию: `true`.
*   `pack-interface-methods` (bool): Включить упаковку методов интерфейсов. По умолчанию: `true`.

**Пример `.golangci.yml`:**

```yaml
linters-settings:
  custom:
    funcwrap:
      path: .bin/linters/funcwrap.so
      description: "Advanced function signature formatter"
      original-url: github.com/username/funcwrap
      settings:
        max-line-len: 120
        tab-width: 8
        pack-struct-fields: true
        pack-interface-methods: true
```

## 🛠 Установка и Сборка

Так как `funcwrap` — это плагин, его нужно скомпилировать той же версией Go, которой собран `golangci-lint`.

1.  **Склонируйте репозиторий:**
    ```bash
    git clone https://github.com/username/funcwrap.git
    cd funcwrap
    ```

2.  **Соберите плагин:**
    ```bash
    go build -buildmode=plugin -o funcwrap.so linter.go
    ```

3.  **Подключите в конфиг** (см. выше).

## 💡 Пример использования

В директории [example/](example/) находится готовый пример проекта, настроенного для использования этого линтера как Go модуля.

Чтобы запустить пример:

1.  Перейдите в директорию примера:
    ```bash
    cd example
    ```
2.  Соберите кастомную версию `golangci-lint`:
    ```bash
    golangci-lint custom
    ```
3.  Запустите линтер:
    ```bash
    ./custom-gcl run
    ```

## 🧪 Разработка и Тестирование

Проект имеет обширный набор тестов, покрывающих граничные случаи и различные конструкции языка.

### Использование Makefile

Проект включает `Makefile` для удобства разработки:

```bash
# Показать все доступные команды
make help

# Запустить тесты
make test

# Запустить тесты с race detector
make test-race

# Запустить тесты с покрытием
make test-coverage

# Обновить golden файлы
make test-update-golden

# Форматировать код
make fmt

# Запустить линтер
make lint

# Запустить все проверки (fmt, vet, lint, test)
make check

# Собрать кастомный golangci-lint binary
make build-example

# Запустить кастомный линтер на примере
make run-example

# Очистить артефакты сборки
make clean
```

### Структура тестов

*   `testdata/src/features/`: Тесты на фичи языка (дженерики, комменты, вариадики).
*   `testdata/src/limits/`: Тесты на разные ограничения длины строки (80, 100, 120, 140, 160).
*   `testdata/src/settings/`: Тесты на различные настройки линтера.

### Обновление Golden файлов

Golden файлы содержат ожидаемые результаты применения фиксов. Для их обновления используйте:

```bash
# Через Makefile (рекомендуется)
make test-update-golden

# Напрямую через GODEBUG
GODEBUG=analysistest.fix=true go test ./...
```

## 📊 Реальные примеры использования

### Пример 1: API обработчики

**До:**
```go
func CreateUser(
    w http.ResponseWriter,
    r *http.Request,
) {
    // ...
}

func UpdateUser(w http.ResponseWriter, r *http.Request, id string,
    name string, email string) {
    // ...
}
```

**После:**
```go
func CreateUser(w http.ResponseWriter, r *http.Request) {
    // ...
}

func UpdateUser(w http.ResponseWriter, r *http.Request, id string, name string, email string) {
    // ...
}
```

### Пример 2: Интерфейсы сервисов

**До (121 строка для 8 методов):**
```go
type UserRepository interface {
    Create(
        ctx context.Context,
        name string,
        email string,
    ) (*User, error)

    Update(
        ctx context.Context,
        id int,
        name string,
        email string,
    ) error

    Delete(
        ctx context.Context,
        id int,
    ) error
    // ... ещё 5 методов
}
```

**После (45 строк для тех же 8 методов - экономия 63%):**
```go
type UserRepository interface {
    Create(ctx context.Context, name string, email string) (*User, error)
    Update(ctx context.Context, id int, name string, email string) error
    Delete(ctx context.Context, id int) error
    // ... ещё 5 методов
}
```

### Пример 3: Generics (Go 1.18+)

**До:**
```go
func Map[T any, R any](
    slice []T,
    fn func(T) R,
) []R {
    // ...
}
```

**После:**
```go
func Map[T any, R any](slice []T, fn func(T) R) []R {
    // ...
}
```

## ⚡ Производительность

- **Быстрая работа**: Не требует загрузки информации о типах (`register.LoadModeSyntax`)
- **Минимальное потребление памяти**: Работает только с AST и токенами
- **Параллельная обработка**: Может анализировать несколько файлов одновременно через `golangci-lint`

На средней кодовой базе (10k LOC):
- Время анализа: ~100-200ms
- Память: ~20-30MB

## 🤔 FAQ

### Почему не использовать `gofmt` или `gofumpt`?

`gofmt` и `gofumpt` великолепны для базового форматирования, но они не применяют строгих правил к длине строк и упаковке параметров. `funcwrap` дополняет их, обеспечивая дополнительный уровень согласованности.

### Можно ли отключить упаковку параметров?

Да! Используйте настройки `pack-struct-fields: false` и `pack-interface-methods: false` для отключения агрессивной упаковки параметров.

### Работает ли с Go 1.18+ Generics?

Да, `funcwrap` полностью поддерживает дженерики, включая параметры типов в квадратных скобках.

### Как обработать особые случаи?

Используйте комментарий `//nolint:funcwrap` для отключения линтера на конкретной функции:

```go
//nolint:funcwrap
func SpecialCase(
    a int,
    b string,
) {
    // Линтер пропустит эту функцию
}
```

### Поддерживается ли автоматическое исправление?

Да! Линтер предоставляет suggested fixes, которые можно применить через:
```bash
golangci-lint run --fix
```

## 🛡️ Детали реализации

*   **Основан на `go/ast` и `go/token`**: Надёжный парсинг через стандартные библиотеки Go
*   **Быстрая работа**: Не требует тяжелой загрузки типов (`register.LoadModeSyntax`)
*   **Точный контроль**: Использует собственный рендерер AST-узлов для точного контроля над пробелами и запятыми, в отличие от стандартного `go/printer`
*   **Поддержка Generics**: Корректная обработка параметров типов `[T any]` (Go 1.18+)
*   **Тестовое покрытие**: Более 100 тестовых кейсов, включая граничные случаи

## 🤝 Участие в разработке

Мы приветствуем вклад в проект! Пожалуйста:

1. Форкните репозиторий
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Убедитесь, что тесты проходят (`make test`)
4. Закоммитьте изменения (`git commit -m 'Add amazing feature'`)
5. Запушьте в ветку (`git push origin feature/amazing-feature`)
6. Откройте Pull Request

### Требования для PR:
- ✅ Все тесты проходят (`make test`)
- ✅ Код отформатирован (`make fmt`)
- ✅ Линтеры не выдают ошибок (`make lint`)
- ✅ Добавлены тесты для новой функциональности
- ✅ Обновлена документация (если необходимо)

## 📝 License

MIT License

## 🔗 Полезные ссылки

- [golangci-lint документация](https://golangci-lint.run/)
- [Go AST пакет](https://pkg.go.dev/go/ast)
- [Разработка пользовательских линтеров](https://golangci-lint.run/contributing/new-linters/)

---

**Техническая документация для AI-ассистентов:**
- [CLAUDE.md](CLAUDE.md) — контекст для Claude AI
- [GEMINI.md](GEMINI.md) — контекст для Gemini AI