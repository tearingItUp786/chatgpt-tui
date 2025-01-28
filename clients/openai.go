package clients

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

type OpenAiClient struct {
	apiUrl        string
	systemMessage string
	client        http.Client
}

func NewOpenAiClient(apiUrl, systemMessage string) *OpenAiClient {
	return &OpenAiClient{
		apiUrl:        apiUrl,
		systemMessage: systemMessage,
		client:        http.Client{},
	}
}

func (c OpenAiClient) RequestCompletion(
	chatMsgs []util.MessageToSend,
	modelSettings util.Settings,
	resultChan chan ProcessApiCompletionResponse,
) tea.Cmd {
	apiKey := os.Getenv("OPENAI_API_KEY")
	path := "v1/chat/completions"
	processResultID := 0 // Initialize a counter for ProcessResult IDs

	return func() tea.Msg {
		body, err := c.constructCompletionRequestPayload(chatMsgs, modelSettings)
		if err != nil {
			return util.ErrorEvent{Message: err.Error()}
		}

		resp, err := c.postOpenAiAPI(apiKey, path, body)
		if err != nil {
			return util.ErrorEvent{Message: err.Error()}
		}

		c.processCompletionResponse(resp, resultChan, &processResultID)
		return nil // Or return a specific message indicating completion or next steps
	}
}

func (c OpenAiClient) RequestModelsList() ProcessModelsResponse {
	apiKey := os.Getenv("OPENAI_API_KEY")
	path := "v1/models"

	resp, err := c.getOpenAiAPI(apiKey, path)
	if err != nil {
		return ProcessModelsResponse{Err: err}
	}

	return processModelsListResponse(resp)
}

func ConstructUserMessage(content string) util.MessageToSend {
	return util.MessageToSend{
		Role:    "user",
		Content: content,
	}
}

func constructSystemMessage(content string) util.MessageToSend {
	return util.MessageToSend{
		Role:    "system",
		Content: content,
	}
}

func (c OpenAiClient) constructCompletionRequestPayload(chatMsgs []util.MessageToSend, modelSettings util.Settings) ([]byte, error) {
	messages := []util.MessageToSend{}
	messages = append(messages, constructSystemMessage(c.systemMessage))
	for _, singleMessage := range chatMsgs {
		messages = append(messages, singleMessage)
	}
	log.Println("Constructing message: ", modelSettings.Model)
	// log.Println("Messages: ", messages)
	body, err := json.Marshal(map[string]interface{}{
		"model":             modelSettings.Model, // Use string literals for keys
		"frequency_penalty": modelSettings.Frequency,
		"max_tokens":        modelSettings.MaxTokens,
		"stream":            true,
		"messages":          messages,
	})
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
		return nil, err
	}

	return body, nil
}

func getBaseUrl(configUrl string) string {
	parsedUrl, err := url.Parse(configUrl)
	if err != nil {
		util.Log("Failed to parse openAi api url from config")
	}
	baseUrl := fmt.Sprintf("%s://%s", parsedUrl.Scheme, parsedUrl.Host)
	return baseUrl
}

func (c OpenAiClient) getOpenAiAPI(apiKey string, path string) (*http.Response, error) {
	baseUrl := getBaseUrl(c.apiUrl)
	requestUrl := fmt.Sprintf("%s/%s", baseUrl, path)

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	return client.Do(req)
}

func (c OpenAiClient) postOpenAiAPI(apiKey, path string, body []byte) (*http.Response, error) {
	baseUrl := getBaseUrl(c.apiUrl)
	requestUrl := fmt.Sprintf("%s/%s", baseUrl, path)

	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	return client.Do(req)
}

func processModelsListResponse(resp *http.Response) ProcessModelsResponse {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return ProcessModelsResponse{Err: err}
		}
		return ProcessModelsResponse{Err: fmt.Errorf(string(bodyBytes))}
	}

	resBody, err := io.ReadAll(resp.Body)

	if err != nil {
		util.Log("response body read failed", err)
		return ProcessModelsResponse{Err: err}
	}

	var models ModelsListResponse
	if err = json.Unmarshal(resBody, &models); err != nil {
		util.Log("response parsing failed", err)
		return ProcessModelsResponse{Err: err}
	}

	return ProcessModelsResponse{Result: models, Err: nil}
}

func (c OpenAiClient) processCompletionResponse(
	resp *http.Response,
	resultChan chan ProcessApiCompletionResponse,
	processResultID *int,
) {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			resultChan <- ProcessApiCompletionResponse{ID: *processResultID, Err: err}
			return
		}
		resultChan <- ProcessApiCompletionResponse{ID: *processResultID, Err: fmt.Errorf(string(bodyBytes))}
		return
	}

	scanner := bufio.NewReader(resp.Body)
	for {
		line, err := scanner.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break // End of the stream
			}
			resultChan <- ProcessApiCompletionResponse{ID: *processResultID, Err: err}
			return
		}

		if line == "data: [DONE]\n" {
			resultChan <- ProcessApiCompletionResponse{ID: *processResultID, Err: nil, Final: true}
			return
		}

		if strings.HasPrefix(line, "data:") {
			jsonStr := strings.TrimPrefix(line, "data:")
			resultChan <- processChunk(jsonStr, *processResultID)
			*processResultID++ // Increment the ID for each processed chunk
		}
	}
}

func processChunk(chunkData string, id int) ProcessApiCompletionResponse {
	var chunk CompletionChunk
	err := json.Unmarshal([]byte(chunkData), &chunk)
	if err != nil {
		log.Println("Error unmarshalling:", chunkData, err)
		return ProcessApiCompletionResponse{ID: id, Result: CompletionChunk{}, Err: err}
	}

	return ProcessApiCompletionResponse{ID: id, Result: chunk, Err: nil}
}
