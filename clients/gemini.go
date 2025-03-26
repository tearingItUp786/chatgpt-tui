package clients

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/generative-ai-go/genai"
	"github.com/tearingItUp786/chatgpt-tui/util"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const modelNamePrefix = "models/"

type GeminiClient struct {
	systemMessage string
}

func NewGeminiClient(systemMessage string) *GeminiClient {
	return &GeminiClient{
		systemMessage: systemMessage,
	}
}

func (c GeminiClient) RequestCompletion(
	ctx context.Context,
	chatMsgs []util.MessageToSend,
	modelSettings util.Settings,
	resultChan chan util.ProcessApiCompletionResponse,
) tea.Cmd {

	return func() tea.Msg {
		client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
		if err != nil {
			return util.ErrorEvent{Message: err.Error()}
		}
		defer client.Close()

		model := client.GenerativeModel(modelNamePrefix + modelSettings.Model)
		log.Println("Gemini: sending message to " + modelSettings.Model)
		model.SetMaxOutputTokens(int32(modelSettings.MaxTokens))
		// TODO: system message

		currentPrompt := chatMsgs[len(chatMsgs)-1].Content
		cs := model.StartChat()
		cs.History = buildChatHistory(chatMsgs)

		iter := cs.SendMessageStream(ctx, genai.Text(currentPrompt))

		processResultID := 0
		for {
			resp, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				resultChan <- util.ProcessApiCompletionResponse{ID: processResultID, Err: err}
				break
			}

			result, isFinished := toCompletionChunk(resp, processResultID)
			resultChan <- util.ProcessApiCompletionResponse{
				ID:     processResultID,
				Result: result,
				Err:    nil,
			}

			processResultID++

			if isFinished {
				sendCompensationChunk(resultChan, processResultID)
			}
		}

		return nil
	}
}

func (c GeminiClient) RequestModelsList() util.ProcessModelsResponse {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return util.ProcessModelsResponse{Err: err}
	}
	defer client.Close()

	modelsIter := client.ListModels(ctx)

	var modelsList []util.ModelDescription
	for {
		model, err := modelsIter.Next()
		if err == iterator.Done {
			return util.ProcessModelsResponse{
				Result: util.ModelsListResponse{
					Data: modelsList,
				},
				Err: nil,
			}
		}

		if err != nil {
			return util.ProcessModelsResponse{Err: err}
		}

		formattedName := strings.TrimPrefix(model.Name, modelNamePrefix)
		modelsList = append(modelsList, util.ModelDescription{Id: formattedName})
	}
}

func sendCompensationChunk(resultChan chan util.ProcessApiCompletionResponse, id int) {
	var chunk util.CompletionChunk
	chunk.ID = fmt.Sprint(id)

	choice := util.Choice{
		Index: id,
		Delta: map[string]interface{}{
			"content": "",
		},
		FinishReason: "stop",
	}

	chunk.Choices = []util.Choice{choice}

	resultChan <- util.ProcessApiCompletionResponse{
		ID:     id,
		Result: chunk,
		Final:  true,
	}
}

func toCompletionChunk(response *genai.GenerateContentResponse, id int) (util.CompletionChunk, bool) {
	var chunk util.CompletionChunk
	chunk.ID = fmt.Sprint(id)

	isFinished := false
	for _, candidate := range response.Candidates {
		if candidate.Content == nil {
			break
		}

		finishReason := handleFinishReason(candidate.FinishReason)

		choice := util.Choice{
			Index:        int(candidate.Index),
			FinishReason: finishReason,
		}

		if len(candidate.Content.Parts) > 0 {
			choice.Delta = map[string]interface{}{
				"content": fromatResponsePart(candidate.Content.Parts[0]),
			}
		} else {
			choice.Delta = map[string]interface{}{
				"content": "",
			}
		}

		if finishReason != "" {
			choice.FinishReason = ""
			chunk.Usage = &util.TokenUsage{
				Prompt:     int(response.UsageMetadata.PromptTokenCount),
				Completion: int(response.UsageMetadata.CandidatesTokenCount),
			}
			isFinished = true
		}

		chunk.Choices = append(chunk.Choices, choice)
	}

	return chunk, isFinished
}

func fromatResponsePart(part genai.Part) string {
	switch v := part.(type) {
	case genai.Text:
		response := string(v)
		if strings.HasPrefix(response, "```") {
			response = "\n" + response
		}
		return response
	default:
		panic("Only text type is supported")
	}
}

func handleFinishReason(reason genai.FinishReason) string {
	switch reason {
	case genai.FinishReasonStop:
		return "stop"
	case genai.FinishReasonMaxTokens:
		return "length"
	case genai.FinishReasonOther:
	case genai.FinishReasonUnspecified:
	case genai.FinishReasonRecitation:
	// TODO: handle recitation
	case genai.FinishReasonSafety:
	default:
		log.Println(fmt.Sprintf("unexpected genai.FinishReason: %#v", reason))
	}

	return ""
}

func buildChatHistory(msgs []util.MessageToSend) []*genai.Content {
	chat := []*genai.Content{}

	for _, singleMessage := range msgs {
		role := "user"
		if singleMessage.Role == "assistant" {
			role = "model"
		}

		if singleMessage.Content != "" {
			message := genai.Content{
				Parts: []genai.Part{
					genai.Text(singleMessage.Content),
				},
				Role: role,
			}
			chat = append(chat, &message)
		}
	}

	return chat
}
