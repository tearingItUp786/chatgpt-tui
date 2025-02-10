package util

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func GetMessagesAsPrettyString(msgsToRender []MessageToSend, w int) string {
	var messages string
	for _, message := range msgsToRender {
		messageToUse := message.Content

		switch {
		case message.Role == "user":
			messageToUse = RenderUserMessage(messageToUse, w)
		case message.Role == "assistant":
			messageToUse = RenderBotMessage(messageToUse, w)
		}

		if messages == "" {
			messages = messageToUse
			continue
		}

		messages = messages + "\n" + messageToUse
	}

	return messages
}

func RenderUserMessage(msg string, width int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Pink100)).
		Render("💁 " + msg)
}

func RenderErrorMessage(msg string, width int) string {
	msg = strings.TrimSpace(msg)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Red)).
		Width(width / 3 * 2).
		Render("⛔ " + "Encountered error:\n" + msg)
}

func RenderBotMessage(msg string, width int) string {
	if msg == "" {
		return ""
	}

	lol, _ := glamour.RenderWithEnvironmentConfig(msg)
	output := strings.TrimSpace(lol)
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderLeftForeground(lipgloss.Color(Pink300)).
		Render(output)
}
