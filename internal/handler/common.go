package handler

import (
	"fmt"
	tele "gopkg.in/telebot.v4"
	"strings"
)

// isPrivate returns true if current chat is private and false if group
func isPrivate(c tele.Context) (bool, error) {
	if chat, err := c.Bot().ChatByID(c.Chat().ID); err != nil {
		return false, err
	} else {
		return chat.Type == tele.ChatPrivate, nil
	}
}

func formatHumanName(guest any) string {
	name := ""
	// guest is telegram user object
	user, ok := guest.(*tele.User)
	if user != nil && ok {
		if len(user.FirstName) > 0 {
			name = user.FirstName
			if len(user.LastName) > 0 {
				name += fmt.Sprintf(" %s", user.LastName)
			}
		}
		if len(user.Username) > 0 {
			name += fmt.Sprintf(" (@%s)", user.Username)
		}
	}
	// guest is telegram chat object
	chat, ok := guest.(*tele.Chat)
	if chat != nil && ok {
		if len(chat.Title) > 0 {
			name = fmt.Sprintf("'%s'", chat.Title)
		}
		if len(chat.Username) > 0 {
			name += fmt.Sprintf(" (@%s)", chat.Username)
		}
	}
	return strings.Trim(name, " ")
}
