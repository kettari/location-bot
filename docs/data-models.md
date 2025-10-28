# Модели данных Location Bot

## Модель Game

### Структура

```go
type Game struct {
    gorm.Model
    ExternalID  string    `json:"id" gorm:"unique;not null"`
    Joinable    bool      `json:"joinable" gorm:"default:false;not null"`
    URL         string    `json:"url" gorm:"size:1024"`
    Title       string    `json:"title" gorm:"size:1024"`
    Date        time.Time `json:"date" gorm:"index"`
    Setting     string    `json:"setting" gorm:"size:100"`
    System      string    `json:"system" gorm:"size:100"`
    Genre       string    `json:"genre" gorm:"size:100"`
    MasterName  string    `json:"master_name" gorm:"size:100"`
    MasterLink  string    `json:"master_link" gorm:"size:1024"`
    Description string    `json:"description"`
    Notes       string    `json:"notes"`
    SeatsTotal  int       `json:"seats_total" gorm:"default:0;not null"`
    SeatsFree   int       `json:"seats_free" gorm:"default:0;not null"`
    Slot        int       `json:"-" gorm:"-:all"`
}
```

### Поля

| Поле | Тип | Описание | Пример |
|------|-----|----------|---------|
| `ID` | uint | Внутренний ID в БД | 1 |
| `CreatedAt` | time.Time | Дата создания записи | 2025-01-19 10:00:00 |
| `UpdatedAt` | time.Time | Дата последнего обновления | 2025-01-19 10:30:00 |
| `DeletedAt` | *time.Time | Дата удаления (soft delete) | null |
| `ExternalID` | string | ID события на Rolecon.ru | "12345" |
| `Joinable` | bool | Доступна ли регистрация | true |
| `URL` | string | Ссылка на страницу события | "https://rolecon.ru/event/12345" |
| `Title` | string | Название игры | "Игра в Dungeons & Dragons" |
| `Date` | time.Time | Дата и время проведения | 2025-02-01 19:00:00 MSK |
| `Setting` | string | Сеттинг игры | "Средневековье" |
| `System` | string | Игровая система | "D&D 5e" |
| `Genre` | string | Жанр игры | "Фэнтези" |
| `MasterName` | string | Имя мастера | "Иван Иванов" |
| `MasterLink` | string | Ссылка на профиль мастера | "https://rolecon.ru/user/123" |
| `Description` | string | Описание игры | "Эпическое приключение..." |
| `Notes` | string | Заметки к игре | "Для новичков" |
| `SeatsTotal` | int | Всего мест за игровым столом | 6 |
| `SeatsFree` | int | Свободных мест | 3 |
| `Slot` | int | Слот времени (для weekend событий) | 0, 1, 2... |

### Бизнес-логика

#### Методы проверки состояния

**`NewJoinable() bool`**
```go
// Проверяет, является ли игра новой и доступной для записи
return game.Date.After(time.Now()) && game.Joinable && game.SeatsFree > 0
```

**`FreeSeatsAdded(game *Game) bool`**
```go
// Проверяет, освободились ли места в уже существующей игре
// (изначально seats_free был 0)
return game.Date.After(time.Now()) && game.Joinable && 
       game.SeatsFree > 0 && game.SeatsFree == 0
```

**`BecomeJoinable(game *Game) bool`**
```go
// Проверяет, стала ли игра доступной для записи
// (ранее была недоступна)
return game.Date.After(time.Now()) && game.Joinable && 
       game.SeatsFree > 0 && !game.Joinable
```

**`WasJoinable() bool`**
```go
// Проверяет, была ли игра ранее доступна для записи
// Используется для определения отмененных игр
return game.Date.After(time.Now()) && game.SeatsTotal > 0
```

**`EqualDate(game *Game) bool`**
```go
// Сравнивает даты двух игр
return game.Date.In(time.UTC).String() == game.Date.In(time.UTC).String()
```

#### Форматирование для Telegram

**`FormatNew() string`**
Формирует сообщение о новой игре:
```html
<b>СУББОТА</b> (01.02, 19:00)
3/6 <a href="https://rolecon.ru/event/123">Игра в D&D</a> [D&D 5e; Средневековье]
```

**`FormatFreeSeatsAdded() string`**
Формирует сообщение об освобождении мест:
```html
Освободилось место:

<b>СУББОТА</b> (01.02, 19:00)
3/6 <a href="https://rolecon.ru/event/123">Игра в D&D</a> [D&D 5e; Средневековье]
```

**`FormatCancelled() string`**
Формирует сообщение об отмене игры:
```html
Игра отменена:

<b>СУББОТА</b> (01.02, 19:00)
Игра в D&D [D&D 5e; Средневековье]
```

## Модели Scraper

### Page

```go
type Page struct {
    URL     string
    Html    string
    Cookies []*http.Cookie
}
```

Хранит HTML контент страницы и cookies для последующих запросов.

### Csrf

```go
type Csrf struct {
    page   *Page
    Token  string
    Cookie string
}
```

CSRF защита для авторизованных запросов к API Rolecon.ru.

### Events

```go
type Events struct {
    URL    string
    Csrf   *Csrf
    JSON   string
    Events []RoleconEvent
}

type RoleconEvent struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
    URL   string `json:"url"`
}
```

Список событий в формате JSON, полученный через API.

## Модели Bot

### Bot

```go
type Bot struct {
    bot         *tele.Bot
    destination []Recipient
}
```

Обертка над Telegram Bot API.

### Recipient

```go
type Recipient struct {
    User     tele.User
    ThreadID int
}
```

Получатель уведомлений (чат ID и номер треда).

Формат конфигурации: `chat_id1,thread_id1;chat_id2,thread_id2`

## Интерфейсы

### Collection

```go
type Collection interface {
    Add(games ...Game)
}
```

Интерфейс для коллекций игр.

### Observer

```go
type Observer interface {
    Update(game *Game, subject SubjectType)
}
```

Интерфейс для уведомлений о событиях с играми.

### MessageDispatcher

```go
type MessageDispatcher interface {
    Send([]string) error
}
```

Интерфейс для отправки сообщений.

### Engine

```go
type Engine interface {
    Process(*scraper.Page) (*[]entity.Game, error)
}
```

Интерфейс для парсинга HTML в структуры Game.

## Типы событий

```go
type SubjectType string

const (
    SubjectTypeNew            SubjectType = "new"
    SubjectTypeBecomeJoinable SubjectType = "become_joinable"
    SubjectTypeCancelled      SubjectType = "cancelled"
)
```

Типы событий для паттерна Observer:
- `new` - новая игра стала доступной
- `become_joinable` - освободились места или игра стала доступной
- `cancelled` - игра была отменена

## Lifecycle игры

1. **Новое событие загружено**
   - Парсинг HTML → создание Game
   - Проверка `NewJoinable()` → событие OnNew()

2. **Изменение статуса**
   - `SeatsFree` увеличился с 0 → событие OnBecomeJoinable()
   - `Joinable` изменился false → true → событие OnBecomeJoinable()

3. **Отмена события**
   - Событие отсутствует в новом загруженном расписании
   - Установка `Joinable = false`
   - Событие OnCancelled()

## Схема базы данных

### Таблица `loc_games`

```sql
CREATE TABLE loc_games (
    id               BIGSERIAL PRIMARY KEY,
    created_at       TIMESTAMP NOT NULL,
    updated_at       TIMESTAMP NOT NULL,
    deleted_at       TIMESTAMP,
    external_id      VARCHAR(255) UNIQUE NOT NULL,
    joinable         BOOLEAN DEFAULT FALSE NOT NULL,
    url              VARCHAR(1024),
    title            VARCHAR(1024),
    date             TIMESTAMP NOT NULL,
    setting          VARCHAR(100),
    system           VARCHAR(100),
    genre            VARCHAR(100),
    master_name      VARCHAR(100),
    master_link      VARCHAR(1024),
    description      TEXT,
    notes            TEXT,
    seats_total      INTEGER DEFAULT 0 NOT NULL,
    seats_free       INTEGER DEFAULT 0 NOT NULL,
    
    INDEX idx_date (date),
    INDEX idx_joinable (joinable)
);
```

Примечание: таблица `loc_games` использует GORM auto-migration с префиксом `loc_` для всех таблиц проекта.

