package util

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func GetMessagesAsPrettyString(msgsToRender []MessageToSend, tw, th int) string {
	var messages string
	for _, message := range msgsToRender {
		messageToUse := message.Content

		switch {
		case message.Role == "user":
			messageToUse = RenderUserMessage(messageToUse, tw/3*2)
		case message.Role == "assistant":
			messageToUse = RenderBotMessage(messageToUse, tw/3*2)
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
