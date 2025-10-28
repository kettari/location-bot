# Use Cases: возникновение и обработка событий

Документ описывает сквозные сценарии (use case) возникновения и обработки событий в системе, основанные на паттерне Observer для сущности `Game`.

## Общие участники и границы

- **Акторы**:
  - Источник данных: Rolecon.ru
  - Сервис: `schedule:fetch` (CLI команда)
  - Подсистемы: `scraper`, `parser`, `schedule`, `storage`, `entity`
  - Интеграция: Telegram Bot (`internal/bot`)
- **Точка входа**: команда `schedule:fetch`
- **Хранилище**: PostgreSQL (`loc_games` через GORM)
- **Коммуникации**: Telegram (HTML формат, поддержка ThreadID)

## Общий поток (для всех событий)

```text
User/CRON
  ↓
Команда schedule:fetch
  ↓                     (HTTP, cookies)
Scraper: Page → CSRF → Events(JSON)
  ↓                     (N URL)
Worker Pool (5 workers): загрузка HTML страниц событий
  ↓                     ([]Page)
Parser(HtmlEngine): HTML → []Game
  ↓                     (регистрация observers)
Schedule.SaveGames(): find-or-create + сравнение
  ↓                     (решение о событии)
Game.On<Event>(): notifyAll(subject)
  ↓
Observers → Bot.Send() → Telegram
```

- Регистрация наблюдателей выполняется в `schedule:fetch` для каждого `Game` до сохранения в БД:
  - `NewGameObserver`
  - `BecomeJoinableGameObserver`
  - `CancelledGameObserver`

---

## UC-1: Новая игра доступна для записи (New)

- **Цель**: Уведомить о новой игре, доступной для записи.
- **Предусловия**:
  - Игра отсутствует в БД по `ExternalID`.
  - Парсер установил поля даты и мест, а также `Joinable` в `true` (есть свободные места и дата в будущем).
- **Триггер**: `schedule:fetch` обрабатывает новую игру.

### Основной сценарий
1. `Parser` возвращает `Game` с заполненными полями.
2. В `Schedule.SaveGames()` выполняется поиск по `ExternalID`.
3. Запись не найдена → `freshGame = true` → создаётся новая запись.
4. После `Save()` вызывается проверка условий события:
   - `freshGame && game.NewJoinable()` → `game.OnNew()`.
5. `Game.OnNew()` вызывает `notifyAll(SubjectTypeNew)`.
6. `NewGameObserver.Update()` формирует HTML через `Game.FormatNew()` и отправляет через `Bot.Send()`.

### Постусловия
- Запись создана в `loc_games`.
- Уведомление доставлено адресатам.

### Исключения/ошибки
- Ошибка БД при `Save()` → сценарий прерывается, логируется Error.
- Ошибка отправки в Telegram → логируется Error, повтор не выполняется в рамках этого сценария.

---

## UC-2: Освободились места / игра стала доступной (BecomeJoinable)

- **Цель**: Уведомить, что появилась возможность записаться (места > 0) или игра стала `Joinable`.
- **Предусловия**:
  - Игра уже существует в БД (найдена по `ExternalID`).
- **Триггер**: `schedule:fetch` получил актуальные данные о той же игре.

### Основной сценарий
1. `Schedule.SaveGames()` загружает «старую» версию записи `storedGame` по `ExternalID`.
2. Применяются новые данные `game`, затем `Save()`.
3. Выбирается событие:
   - Если `game.FreeSeatsAdded(&storedGame)` ИЛИ `game.BecomeJoinable(&storedGame)` → `game.OnBecomeJoinable()`.
4. `Game.OnBecomeJoinable()` вызывает `notifyAll(SubjectTypeBecomeJoinable)`.
5. `BecomeJoinableGameObserver.Update()` формирует HTML через `Game.FormatFreeSeatsAdded()` и отправляет через `Bot.Send()`.

### Постусловия
- Запись обновлена в `loc_games`.
- Уведомление доставлено адресатам.

### Исключения/ошибки
- Ошибка БД при `Save()` → сценарий прерывается.
- Ошибка отправки в Telegram → логируется Error.

---

## UC-3: Игра отменена (Cancelled)

- **Цель**: Уведомить, что ранее доступная игра отменена/исчезла из расписания.
- **Предусловия**:
  - В БД есть записи с будущей датой и `Joinable = true`.
- **Триггер**: `schedule:fetch` завершил загрузку и парсинг текущего листинга, затем вызвал `Schedule.CheckAbsentGames()`.

### Основной сценарий
1. `Schedule.CheckAbsentGames()` получает из БД список «актуально joinable» игр в будущем.
2. Для каждой такой игры регистрируется `CancelledGameObserver`.
3. Сравнивается список из БД с новыми загруженными `s.Games`.
4. Если игра из БД не найдена среди новых:
   - Логируется Warn, выставляется `Joinable = false`, выполняется `Save()`.
   - Если `sg.WasJoinable()` → `sg.OnCancelled()`.
5. `Game.OnCancelled()` вызывает `notifyAll(SubjectTypeCancelled)`.
6. `CancelledGameObserver.Update()` формирует HTML через `Game.FormatCancelled()` и отправляет через `Bot.Send()`.

### Постусловия
- Запись обновлена (`Joinable = false`).
- Уведомление доставлено адресатам.

### Исключения/ошибки
- Ошибка БД при `Save()` → сценарий прерывается.
- Ошибка отправки в Telegram → логируется Error.

---

## Форматы уведомлений (сокращенно)

```html
<b>DOW</b> (DD.MM, HH:MM)
F/T <a href="URL">TITLE</a> [SYSTEM; SETTING]
```

- New: без префикса.
- BecomeJoinable: префикс «Освободилось место:». 
- Cancelled: префикс «Игра отменена:», без F/T.

## Нефункциональные требования/заметки

- Worker Pool: 5 воркеров, единый collector; при наличии ошибок в сборе страниц выставляется флаг и результаты отбрасываются.
- Временные зоны: все даты формируются и выводятся в `Europe/Moscow`.
- Логирование: `slog` с уровнями Info/Debug/Warn/Error; не логировать секреты.
- Идемпотентность: поиск по `ExternalID` + `FirstOrCreate` в `SaveGames()` снижает риск дублей.
- Ограничения Telegram: длина сообщения сегментируется в `Schedule.Format()` (до ~4000 символов).
- Конфигурация получателей: `chat_id,thread_id` пары через `;`.
