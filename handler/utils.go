package handler

import (
	"GPTBot/api/telegram"
	"strings"
)

// authorName returns a human-readable name for the sender of an update.
func authorName(update telegram.Update) string {
	u := update.Message.From
	if u == nil {
		return "Unknown"
	}
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name != "" {
		return name
	}
	return u.UserName
}
