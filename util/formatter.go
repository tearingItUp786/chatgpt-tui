package util

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func GetMessagesAsPrettyString(msgsToRender []MessageToSend, w int, colors SchemeColors) string {
	var messages string
	for _, message := range msgsToRender {
		messageToUse := message.Content

		switch {
		case message.Role == "user":
			messageToUse = RenderUserMessage(messageToUse, w, colors)
		case message.Role == "assistant":
			messageToUse = RenderBotMessage(messageToUse, w, colors)
		}

		if messages == "" {
			messages = messageToUse
			continue
		}

		messages = messages + "\n" + messageToUse
	}

	return messages
}

func RenderUserMessage(msg string, width int, colors SchemeColors) string {
	msg = "\n💁 " + msg + "\n"
	userMsg, _ := glamour.Render(msg, colors.RendererTheme)
	output := strings.TrimSpace(userMsg)
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.InnerHalfBlockBorder()).
		BorderLeftForeground(colors.NormalTabBorderColor).
		Render("\n" + output + "\n")
}

func RenderErrorMessage(msg string, width int, colors SchemeColors) string {
	msg = "```json\n" + msg + "\n```"
	errMsg, _ := glamour.Render(msg, colors.RendererTheme)
	output := strings.TrimSpace(errMsg)
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.InnerHalfBlockBorder()).
		BorderLeftForeground(colors.ErrorColor).
		Width(width).
		Foreground(colors.HighlightColor).
		Render(" ⛔ " + "Encountered error:\n" + output)
}

func RenderBotMessage(msg string, width int, colors SchemeColors) string {
	if msg == "" {
		return ""
	}

	msg = "\n🤖 " + msg + "\n"
	aiResponse, _ := glamour.Render(msg, colors.RendererTheme)
	output := strings.TrimSpace(aiResponse)
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.InnerHalfBlockBorder()).
		BorderLeftForeground(colors.ActiveTabBorderColor).
		Render(output)
}
