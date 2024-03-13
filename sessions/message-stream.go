package sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

type Choice struct {
	Index        int                    `json:"index"`
	Delta        map[string]interface{} `json:"delta"`
	FinishReason string                 `json:"finish_reason"`
}

type CompletionChunk struct {
	ID               string   `json:"id"`
	Object           string   `json:"object"`
	Created          int      `json:"created"`
	Model            string   `json:"model"`
	SystemFingerpint string   `json:"system_fingerprint"`
	Choices          []Choice `json:"choices"`
}

type CompletionResponse struct {
	Data CompletionChunk `json:"data"`
}

// Define a type for the data you want to return, if needed
type ProcessResult struct {
	ID     int
	Result CompletionChunk // or whatever type you need
	Err    error
	Final  bool
}

type MessageToSend struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func ConstructUserMessage(content string) MessageToSend {
	return MessageToSend{
		Role:    "user",
		Content: content,
	}
}

func constructSystemMessage(content string) MessageToSend {
	return MessageToSend{
		Role:    "system",
		Content: content,
	}
}

func (m Model) constructJsonBody() ([]byte, error) {
	messages := []MessageToSend{}
	messages = append(messages, constructSystemMessage(m.config.SystemMessage))
	for _, singleMessage := range m.ArrayOfMessages {
		messages = append(messages, singleMessage)
	}
	// log.Println("Messages: ", messages)
	body, err := json.Marshal(map[string]interface{}{
		"model":             m.Settings.Model, // Use string literals for keys
		"frequency_penalty": m.Settings.Frequency,
		"max_tokens":        m.Settings.MaxTokens,
		"stream":            true,
		"messages":          messages,
	})
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
		return nil, err
	}

	return body, nil
}

func (m *Model) callChatGptAPI(apiKey string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", m.config.ChatGPTApiUrl, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	return client.Do(req)
}

func (m *Model) processAPIResponse(
	resp *http.Response,
	resultChan chan ProcessResult,
	processResultID *int,
) {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			resultChan <- ProcessResult{ID: *processResultID, Err: err}
			return
		}
		resultChan <- ProcessResult{ID: *processResultID, Err: fmt.Errorf(string(bodyBytes))}
		return
	}

	scanner := bufio.NewReader(resp.Body)
	for {
		line, err := scanner.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break // End of the stream
			}
			resultChan <- ProcessResult{ID: *processResultID, Err: err}
			return
		}

		if line == "data: [DONE]\n" {
			resultChan <- ProcessResult{ID: *processResultID, Err: nil, Final: true}
			return
		}

		if strings.HasPrefix(line, "data:") {
			jsonStr := strings.TrimPrefix(line, "data:")
			resultChan <- processChunk(jsonStr, *processResultID)
			*processResultID++ // Increment the ID for each processed chunk
		}
	}
}

func (m *Model) CallChatGpt(resultChan chan ProcessResult) tea.Cmd {
	apiKey := os.Getenv("OPENAI_API_KEY")
	processResultID := 0 // Initialize a counter for ProcessResult IDs

	return func() tea.Msg {
		body, err := m.constructJsonBody()
		if err != nil {
			return util.ErrorEvent{Message: err.Error()}
		}

		resp, err := m.callChatGptAPI(apiKey, body)
		if err != nil {
			return util.ErrorEvent{Message: err.Error()}
		}

		m.processAPIResponse(resp, resultChan, &processResultID)
		return nil // Or return a specific message indicating completion or next steps
	}
}

// Converts the array of json messages into a single Message
func constructJsonMessage(arrayOfProcessResult []ProcessResult) (MessageToSend, error) {
	newMessage := MessageToSend{Role: "assistant", Content: ""}

	for _, aMessage := range arrayOfProcessResult {
		if aMessage.Final {
			util.Log("Hit final in construct", aMessage.Result)
			break
		}

		if len(aMessage.Result.Choices) > 0 {
			choice := aMessage.Result.Choices[0]
			// TODO: we need a helper for this
			if choice.FinishReason == "stop" || choice.FinishReason == "length" {
				util.Log("Hit stop or length in construct")
				break
			}

			if content, ok := choice.Delta["content"].(string); ok {
				newMessage.Content += content
			} else {
				// Handle the case where the type assertion fails, e.g., log an error or return
				util.Log("type assertion to string failed for choice.Delta[\"content\"]")
				formattedError := fmt.Errorf("type assertion to string failed for choice.Delta[\"content\"]")
				return MessageToSend{}, formattedError
			}

		}
	}
	return newMessage, nil
}

func processChunk(chunkData string, id int) ProcessResult {
	var chunk CompletionChunk
	err := json.Unmarshal([]byte(chunkData), &chunk)
	if err != nil {
		log.Println("Error unmarshalling:", chunkData, err)
		return ProcessResult{ID: id, Result: CompletionChunk{}, Err: err}
	}

	// log.Println("wtf", id, chunk.Choices)
	// Process the chunk as needed
	return ProcessResult{ID: id, Result: chunk, Err: nil}
}
