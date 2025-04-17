package chatgpt

import (
	"context"
	"errors"
	"fmt"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"log/slog"
	"net/http"
)

const (
	MaxInputLength = 60000

	systemMessage = `
Проанализируй HTML документ, который начинается после строки HTML-ДОКУМЕНТ. 
Найди в документе расписание игр, каждая игра заключена в тег "div" с классом "event-single".

Найди для каждой игры атрибуты:
id - идентификатор игры, например "game17418"
joinable - булевый признак, можно ли присоединиться. Если можно то в элементе "div" есть класс "can-join". Если нельзя присоединиться, то класс "cannot-join"
url - ссылка на сайт https://rolecon.ru и путь к ресурсу игры
title - название игры. Если в названии есть диапазон времени в круглых скобках, убери его. Например «(19:00 – 23:00 )»
date - дата и время игры в формате ISO 8601 с таймзоной Москвы
setting - сеттинг игры
system - система игры
genre - жанр игры
master_name - ведущий игры
master_link - ссылка на страницу мастера игры
description - описание игры из элемента <div class="event-single-about-block">
notes - заметки мастера об игре из элемента <div class="event-single-about-inline">
seats_total - всего мест на игру
seats_free - свободных мест на игру. Если joinable=false то свободных мест равно 0

Верни результат в виде чистого валидного JSON, без своих комментариев и без форматирования Markdown. 
Если в тексте встречаются коды UTF-символов или непечатные символы, замени их на HTML-entities. Если не получается
заменить - то удали такие символы. Символ "\u00a0" замени на "&nbsp;".

Пример:

{
	"games": [
		{
			"id": "game123",
			"joinable": true,
			"url": "https://rolecon.ru/path",
			"title": "Название игры 1",
			"date": "2025-04-20T11:00:00+03:00",
			"setting": "Eberron",
			"system": "D&D 2024",
			"genre": "Экшн, расследование.",
			"master_name": "kauzt",
			"master_link": "https://rolecon.ru/user/24001",
			"description": "Когда заточённые в подземелье хтонические существа из другой реальности решают объединиться, жители поверхности сначала теряются, а потом — находят самых неожиданных союзников.",
			"notes": "Ваншот из серии ваншотов",
			"seats_total": 6,
			"seats_free": 0
		},
		{
			"id": "game456",
			"joinable": false,
			"url": "https://rolecon.ru/path",
			"title": "Название игры 2",
			"date": "2025-04-20T11:00:00+03:00",
			"setting": "Авторский сеттинг",
			"system": "D&D 2024",
			"genre": "триллер на выживание",
			"master_name": "Tindomerel",
			"master_link": "https://rolecon.ru/user/3647",
			"description": "Партия набрана",
			"notes": "4+мастер",
			"seats_total": 0,
			"seats_free": 0
		}
	]
}

HTML-ДОКУМЕНТ`
)

type MessageNotifier func(string, bool) error
type TypingNotifier func() error

type ChatGPT struct {
	openAIApiKey  string
	languageModel string
}

func NewChatGPT(openaiApiKey, langModule string) *ChatGPT {
	return &ChatGPT{openAIApiKey: openaiApiKey, languageModel: langModule}
}

func (c *ChatGPT) NewParseCompletion(events string) (*string, error) {
	client := openai.NewClient(option.WithAPIKey(c.openAIApiKey))
	ctx := context.Background()

	question := systemMessage + "\n" + events

	slog.Info("Sending a message to ChatGPT")

	// Prepare prompt
	prompt := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(question),
	}
	params := openai.ChatCompletionNewParams{
		Messages: prompt,
		Model:    c.languageModel,
	}

	// Ask OpenAI
	completion, err := client.Chat.Completions.New(ctx, params)

	// Check for errors
	if err != nil {
		var e *openai.Error
		if errors.As(err, &e) {
			slog.Error(e.Error())
			switch e.StatusCode {
			case http.StatusTooManyRequests:
				return nil, errors.New("OpenAI API error: 429 Too many requests")
			case http.StatusForbidden:
				return nil, errors.New("OpenAI API error: 403 Forbidden")
			default:
				return nil, errors.New("OpenAI API error: unknown error")
			}
		}
		slog.Error("Failed to create completion", "question", question, "error", err)
		return nil, errors.New(fmt.Sprintf("failed to create completion: %s", e.Error()))
	}

	if len(completion.Choices) > 0 {
		return &completion.Choices[0].Message.Content, nil
	}

	return nil, errors.New("got empty choices from the OpenAI API")
}
