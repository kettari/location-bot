# Архитектура проекта Location Bot

## Обзор

Location Bot - это система мониторинга и уведомления о событиях на платформе Rolecon.ru. Система собирает информацию об играх (столах), отслеживает изменения статусов и отправляет уведомления через Telegram Bot API.

## Структурные модули

### 1. Entry Point (`cmd/console/main.go`)

Точка входа в приложение. Регистрирует команды и выполняет их по CLI аргументам.

**Команды:**
- `help` - выводит справку по командам
- `schedule:fetch` - загружает события с Rolecon сервера и парсит их в БД
- `schedule:report:full` - формирует полный отчет об играх
- `bot:poll` - запускает Telegram бота для обработки команд
- `migrate` - выполняет миграции базы данных

### 2. Config (`internal/config/config.go`)

Модуль конфигурации приложения. Загружает параметры из переменных окружения:

- `BOT_DEBUG` - режим отладки
- `BOT_TELEGRAM_TOKEN` - токен Telegram бота
- `BOT_TELEGRAM_NAME` - имя бота
- `BOT_OPENAI_API_KEY` - API ключ OpenAI
- `BOT_OPENAI_LANGUAGE_MODEL` - модель языка OpenAI
- `BOT_DB_STRING` - строка подключения к БД
- `BOT_NOTIFICATION_CHAT_ID` - идентификаторы чатов для уведомлений

### 3. Scraper (`internal/scraper/`)

Модуль для получения данных с веб-сайта Rolecon.ru.

#### Подмодули:

**`page.go`** - загрузка HTML страниц
```go
type Page struct {
    URL     string
    Html    string
    Cookies []*http.Cookie
}
```

**`csrf.go`** - извлечение CSRF токенов для авторизации
```go
type Csrf struct {
    page   *Page
    Token  string
    Cookie string
}
```

**`events.go`** - загрузка списка событий через JSON API
```go
type Events struct {
    URL    string
    Csrf   *Csrf
    JSON   string
    Events []RoleconEvent
}
```

### 4. Parser (`internal/parser/`)

Модуль парсинга HTML контента.

**`parser.go`** - основной парсер, использует Engine для обработки
```go
type Parser struct {
    engine Engine
}
```

**`engine_html.go`** - HTML парсинг с использованием `golang.org/x/net/html`
- Извлекает информацию о датах и слотах времени
- Парсит таблицы с деталями игр
- Определяет статус доступности игры (joinable)

### 5. Entity (`internal/entity/`)

Доменная модель и паттерн Observer для уведомлений.

#### Основные типы:

**`game.go`** - модель игрового события
```go
type Game struct {
    gorm.Model
    ExternalID  string    // ID события на Rolecon
    Joinable    bool      // Доступна ли регистрация
    URL         string    // Ссылка на событие
    Title       string    // Название
    Date        time.Time // Дата и время
    Setting     string    // Сеттинг
    System      string    // Игровая система
    Genre       string    // Жанр
    MasterName  string    // Имя мастера
    MasterLink  string    // Ссылка на профиль мастера
    Description string    // Описание
    Notes       string    // Заметки
    SeatsTotal  int       // Всего мест
    SeatsFree   int       // Свободных мест
    Slot        int       // Слот времени
}
```

**`observer.go`** - интерфейс Observer
```go
type Observer interface {
    Update(game *Game, subject SubjectType)
}
```

**`subject.go`** - типы событий
```go
type SubjectType string

const (
    SubjectTypeNew            = "new"
    SubjectTypeBecomeJoinable = "become_joinable"
    SubjectTypeCancelled      = "cancelled"
)
```

#### Реализации Observer:

**`observer_new.go`** - уведомление о новых играх
**`observer_become_joinable.go`** - уведомление о появлении мест
**`observer_cancelled.go`** - уведомление об отмене игры

### 6. Schedule (`internal/schedule/`)

Модуль оркестрации расписания и управления жизненным циклом игр.

**`schedule.go`** - основная логика работы с расписанием
- Добавление игр в коллекцию
- Загрузка joinable событий из БД
- Сохранение игр с обработкой изменений
- Проверка отсутствующих игр (отмена)
- Форматирование для отправки в Telegram

### 7. Storage (`internal/storage/manager.go`)

Модуль работы с базой данных PostgreSQL через GORM.

```go
type Manager struct {
    connectionString string
    db               *gorm.DB
}
```

Особенности:
- Префикс таблиц: `loc_`
- Подключение через PostgreSQL driver
- Миграции через GORM AutoMigrate

### 8. Bot (`internal/bot/bot.go`)

Модуль отправки уведомлений через Telegram Bot API.

```go
type Bot struct {
    bot         *tele.Bot
    destination []Recipient
}
```

Поддержка:
- Отправка в несколько чатов
- Отправка в треды (ThreadID)
- HTML форматирование

### 9. Handler (`internal/handler/`)

Обработчики команд Telegram бота.

**`start.go`** - команда `/start`
**`help.go`** - команда `/help`
**`games.go`** - команда `/games` (список доступных игр)
**`common.go`** - общие утилиты

### 10. Console (`internal/console/`)

Команды для CLI интерфейса.

**`schedule_fetch.go`** - команда загрузки расписания:
- Загрузка главной страницы
- Извлечение CSRF токена
- Загрузка списка событий через JSON API
- Параллельная загрузка страниц событий (worker pool с 5 воркерами)
- Парсинг HTML контента
- Сохранение в БД с обработкой событий

**`bot_poll.go`** - запуск Telegram бота с polling

**`schedule_report_full.go`** - формирование полного отчета

**`migrate.go`** - миграции БД

## Потоки данных

### 1. Загрузка расписания (`schedule:fetch`)

```
┌─────────────┐
│ Rolecon.ru  │
└──────┬──────┘
       │ HTML + Cookies
       ↓
┌─────────────┐
│ Scraper     │ → Извлечение CSRF токена
└──────┬──────┘
       │ JSON events list
       ↓
┌─────────────┐
│ Events      │ → URL списка событий
└──────┬──────┘
       │ Parallel worker pool (5 workers)
       ↓
┌─────────────┐
│ Page        │ → HTML контент каждого события
└──────┬──────┘
       │
       ↓
┌─────────────┐
│ Parser      │ → Парсинг HTML в Game entities
└──────┬──────┘
       │ []Game
       ↓
┌─────────────┐
│ Schedule    │ → Регистрация Observer'ов
└──────┬──────┘
       │
       ↓
┌─────────────┐
│ Database    │ → Сохранение и сравнение с существующими
└──────┬──────┘
       │ Events fired
       ↓
┌─────────────┐
│ Observers   │ → Уведомления через Telegram
└─────────────┘
```

### 2. Обработка команд бота (`bot:poll`)

```
User (/start, /help, /games)
       ↓
┌─────────────┐
│ Handler     │ → Обработка команды
└──────┬──────┘
       │ /games requires DB
       ↓
┌─────────────┐
│ Schedule    │ → LoadJoinableEvents()
└──────┬──────┘
       │ Format() → HTML
       ↓
┌─────────────┐
│ Bot         │ → Отправка ответа
└─────────────┘
```

### 3. Проверка отмененных игр

```
schedule:fetch
       ↓
┌─────────────┐
│ CheckAbsent │ → Сравнение загруженных с сохраненными
│   Games     │
└──────┬──────┘
       │ Absent games
       ↓
┌─────────────┐
│ Observer    │ → OnCancelled()
│ Cancelled   │
└──────┬──────┘
       ↓
┌─────────────┐
│ Telegram    │ → Уведомление об отмене
└─────────────┘
```

### 4. События Observer

Три типа событий обрабатываются Observer паттерном:

1. **New** - новая игра стала доступной для записи
   - Условие: `game.NewJoinable()` → дата в будущем, joinable=true, seats_free > 0
   
2. **BecomeJoinable** - в игре освободились места или она стала доступной
   - Условие: `game.FreeSeatsAdded()` или `game.BecomeJoinable()`
   
3. **Cancelled** - игра была отменена
   - Условие: игра больше не присутствует в загруженных событиях

## Модели данных

### Таблица `loc_games`

```sql
CREATE TABLE loc_games (
    id               BIGSERIAL PRIMARY KEY,
    created_at       TIMESTAMP,
    updated_at       TIMESTAMP,
    deleted_at       TIMESTAMP,
    external_id      VARCHAR(255) UNIQUE NOT NULL,
    joinable         BOOLEAN DEFAULT FALSE NOT NULL,
    url              VARCHAR(1024),
    title            VARCHAR(1024),
    date             TIMESTAMP WITH INDEX,
    setting          VARCHAR(100),
    system           VARCHAR(100),
    genre            VARCHAR(100),
    master_name      VARCHAR(100),
    master_link      VARCHAR(1024),
    description      TEXT,
    notes            TEXT,
    seats_total      INTEGER DEFAULT 0 NOT NULL,
    seats_free       INTEGER DEFAULT 0 NOT NULL
);
```

### Модели в памяти

**Scraper:**
- `Page` - HTML страница с cookies
- `Csrf` - CSRF токен и cookie
- `Events` - список событий в JSON формате
- `RoleconEvent` - одно событие из списка

**Parser:**
- `dateSlots` - карта слотов времени к датам
- `HtmlEngine` - парсер HTML

**Bot:**
- `Recipient` - получатель уведомления (chat_id + thread_id)

## Зависимости

### Внешние библиотеки

- `gorm.io/gorm` - ORM для работы с БД
- `gorm.io/driver/postgres` - PostgreSQL драйвер
- `gopkg.in/telebot.v4` - Telegram Bot API клиент
- `golang.org/x/net/html` - HTML парсер
- Standard library: `log/slog`, `net/http`, `time`, `regexp`

### Бизнес-логика

1. **Worker Pool Pattern** - параллельная загрузка страниц событий (5 воркеров)
2. **Observer Pattern** - уведомления о событиях с играми
3. **Strategy Pattern** - `Engine` интерфейс для различных парсеров (HTML)
4. **Repository Pattern** - `storage.Manager` абстрагирует доступ к БД

## Особенности реализации

### Concurrency

Используется worker pool для параллельной загрузки страниц событий:
- Канал `jobs` для распределения задач
- Канал `results` для сбора результатов
- 5 воркеров обрабатывают задачи параллельно
- Горутина `collector` собирает результаты

### Обработка ошибок

- Флаги ошибок при сборе результатов
- Возврат ошибок через интерфейс `Command.Run()`
- Логирование через `slog`

### Форматирование сообщений

HTML форматирование для Telegram:
- Жирный текст для дат
- Ссылки на события
- Эмодзи для списков игр

### Timezone handling

Все даты обрабатываются в часовом поясе `Europe/Moscow`.

## Архитектурные принципы

1. **Разделение ответственности** - каждый модуль отвечает за свою область
2. **Инверсия зависимостей** - интерфейсы для абстракций
3. **Clean Architecture** - доменная модель в `entity`, адаптеры в `scraper`, `bot`, `storage`
4. **Command Pattern** - CLI команды через интерфейс `Command`
5. **Observer Pattern** - уведомления о событиях с играми

