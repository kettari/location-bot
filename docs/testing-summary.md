# Тестирование: сводка

## Созданные тесты

### 1. `internal/scraper/` - Тесты для scraper

#### `csrf_test.go`
- Табличные тесты для `ExtractCsrfToken()`
- Тестирование парсинга токена с различными форматами
- Проверка обработки ошибок при отсутствии токена

#### `http_client_test.go`
- Тесты timeout поведения
- Тесты connection pooling
- Проверка HTTP/2 поддержки
- Проверка propagation контекста
- Проверка на утечки соединений

#### `fetcher_test.go`
- Интеграционные тесты с `httptest`
- Тестирование полного цикла fetch (CSRF + Events + Pages)
- Тестирование обработки ошибок (500, пустые события)
- Мокирование Rolecon.ru endpoints

### 2. `internal/parser/` - Тесты для парсера

#### `engine_html_test.go`
- Табличные тесты для `HtmlEngine.Process()`
- Тестирование парсинга одиночных событий
- Тестирование weekend событий с несколькими слотами
- Проверка парсинга дат и времени
- Проверка вычисления `Joinable` статуса
- Тестирование parsing атрибутов событий

### 3. `internal/console/` - Тесты для команд

#### `worker_pool_test.go`
- Тесты dispatcher worker pool
- Тестирование параллельной загрузки страниц
- Проверка обработки ошибок
- Тестирование context cancellation
- Проверка пустых результатов
- Тестирование различных конфигураций workers

### 4. `internal/bot/` - Тесты для бота

#### `bot_test.go`
- Табличные тесты для `prepareDestination()`
- Тестирование парсинга recipients формата `chat_id,thread_id;...`
- Проверка обработки некорректных форматов
- Тестирование number parsing
- Проверка на паники при invalid input

## Запуск тестов

Для запуска тестов используйте стандартные команды Go:

```bash
# Все тесты в пакете
go test ./internal/scraper/... -v

# Конкретный пакет
go test ./internal/parser/... -v
go test ./internal/console/... -v
go test ./internal/bot/... -v

# С покрытием
go test ./internal/scraper/... -cover

# Параллельно
go test ./internal/... -parallel 4

# Все тесты проекта
go test ./...
```

### Примечание

Если возникают проблемы с GOROOT, убедитесь что:
1. Go установлен и находится в PATH
2. Версия Go соответствует требованиям в `go.mod` (1.23)
3. Выполните `go mod tidy` для синхронизации зависимостей

## Покрытие

Тесты покрывают:
- ✅ CSRF extraction (scraper)
- ✅ HTTP client configuration (timeout, pooling)
- ✅ Fetcher service integration (httptest)
- ✅ HTML parsing (table-driven tests)
- ✅ Worker pool dispatcher
- ✅ Bot recipient parsing

## Следующие шаги

1. Улучшить тесты для `internal/entity/` - Observer паттерн
2. Добавить тесты для `internal/schedule/` - бизнес-логика
3. Добавить тесты для `internal/storage/` - GORM операции
4. Интеграционные тесты для полного flow `schedule:fetch`
5. Performance/load тесты для worker pool

## Примечания

- Используется `httptest` для изоляции внешних зависимостей
- Table-driven tests для улучшения читаемости и покрытия
- Тесты используют стандартные библиотеки Go (`testing`, `net/http/httptest`)
- Линтер проверен, ошибок нет

