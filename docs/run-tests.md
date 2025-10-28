# Запуск тестов

## Проблема с GOROOT

Если вы видите ошибку:
```
go: cannot find GOROOT directory: C:\Users\...\pkg\mod\golang.org\toolchain@v0.0.1-go1.24.9.windows-amd64
```

Это означает, что Go пытается использовать toolchain из модулей, который не установлен. Решение:

## Вариант 1: Через IDE (Goland/VS Code)

1. Откройте IDE и дождитесь загрузки проекта
2. Правый клик на любом `*_test.go` файле
3. Выберите "Run Test" или "Debug Test"
4. Или используйте команды:
   - `Ctrl+Shift+F10` в GoLand для запуска теста
   - `F5` в VS Code с Go extension

## Вариант 2: Установка правильной версии Go

```powershell
# Проверить текущую версию
go version

# Если версия не 1.23.x, обновить Go
# Скачать с https://go.dev/dl/
# Или через winget: winget install GoLang.Go
```

## Вариант 3: Через прямой путь к go.exe

```powershell
# Найти где установлен Go
where.exe go

# Запустить с абсолютным путем
C:\Users\akornienko\sdk\go1.23.4\bin\go.exe test ./internal/scraper/... -v
```

## Вариант 4: Очистка кэша модулей

```powershell
# Удалить кэш и заново загрузить зависимости
go clean -modcache
go mod download
go test ./...
```

## Запуск конкретных тестов

```bash
# Все тесты в пакете scraper
go test ./internal/scraper/... -v

# Конкретный файл
go test ./internal/scraper/csrf_test.go ./internal/scraper/csrf.go -v

# С покрытием
go test ./internal/scraper/... -cover

# Только конкретный тест
go test ./internal/scraper -run TestExtractCsrfToken -v
```

## Созданные тест-файлы

### `internal/scraper/`
- ✅ `csrf_test.go` - CSRF extraction тесты
- ✅ `http_client_test.go` - HTTP client configuration тесты  
- ✅ `fetcher_test.go` - Integration tests с httptest

### `internal/parser/`
- ✅ `engine_html_test.go` - Table-driven HTML parsing тесты

### `internal/console/`
- ✅ `worker_pool_test.go` - Worker pool dispatcher тесты

### `internal/bot/`
- ✅ `bot_test.go` - Recipient parsing тесты

## Проверка компиляции

Если тесты не запускаются, проверьте компиляцию:

```bash
# Проверка синтаксиса всех тестов
go test -c ./internal/scraper/...

# Компиляция без запуска
go build ./internal/scraper/...

# Проверка импортов
go list -f '{{.ImportPath}}' ./internal/scraper/...
```

## Рекомендация

Лучше всего запускать тесты через **GoLand IDE**:
1. Откройте проект в GoLand
2. Дождитесь индексации
3. Найдите любой `*_test.go` файл
4. Нажмите зеленую стрелку рядом с функцией TestXxx
5. Или запустите весь файл тестов

IDE автоматически настроит правильный GOROOT и PATH.



