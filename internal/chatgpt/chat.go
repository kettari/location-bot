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

const systemMessage = `
Проанализируй HTML документ, который начинается после строки HTML-ДОКУМЕНТ. 
Найди в документе расписание игр, каждая игра заключена в тег "div" с классом "event-single".

Найди для каждой игры атрибуты:
id - идентификатор игры, например "game17418"
joinable - булевый признак, можно ли присоединиться. Если можно то в элементе "div" есть класс "can-join". Если нельзя присоединиться, то класс "cannot-join"
url - ссылка на сайт https://rolecon.ru и путь к ресурсу игры
title - название игры
date - дата и время игры в формате ISO 8601 с таймзоной Москвы

Верни результат в виде валидного JSON, без своих комментариев, пример:

{
	"data": [
		{
			"id": 123,
			"joinable": false,
			"url": "https://rolecon.ru/path",
			"title": "Название игры 1",
			"date": "Дата и время игры 1"
		},
		{
			"id": 456,
			"joinable": true,
			"url": "https://rolecon.ru/path",
			"title": "Название игры 2",
			"date": "Дата и время игры 2"
		},
	]
}

HTML-ДОКУМЕНТ`

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
		openai.UserMessage(systemMessage),
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
