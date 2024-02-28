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
	"github.com/tearingItUp786/golang-tui/util"
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

func (m Model) constructJsonBody() ([]byte, error) {
	messages := []MessageToSend{}
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

func (m Model) CallChatGpt(resultChan chan ProcessResult) tea.Cmd {
	apiKey := os.Getenv("API_KEY")
	processResultID := 0 // Initialize a counter for ProcessResult IDs

	return func() tea.Msg {
		body, err := m.constructJsonBody()

		// API endpoint to call -- should be an env variable
		req, err := http.NewRequest(
			"POST",
			"https://api.openai.com/v1/chat/completions",
			bytes.NewBuffer(body),
		)
		if err != nil {
			// Handle error
			log.Println("Error creating request:", err)
			resultChan <- ProcessResult{ID: processResultID, Err: err}
		}

		// Set the Content-Type header
		req.Header.Set("Content-Type", "application/json")
		// Set any other headers you need
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
		// Create a new HTTP client and send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			// Handle error
			log.Println("Error sending request:", err)
		}
		defer resp.Body.Close()

		// any kind of error, just break out man
		if resp.StatusCode >= 400 {
			// Read the response body
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading response body: %v\n", err)
				return util.ErrorEvent{
					Message: err.Error(),
				}
			}
			bodyString := string(bodyBytes)
			log.Printf("Error response (status code %d): %s\n", resp.StatusCode, bodyString)
			return util.ErrorEvent{
				Message: bodyString,
			}
		}

		scanner := bufio.NewReader(resp.Body)

		for {
			line, err := scanner.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// log.Println("end of file")
					break // End of the stream
				}
				// TODO: proper error handler when the stream breaks
				log.Fatal(err) // Handle other errors
			}

			// This should be a constant (checking to see if the stream is done)
			if line == "data: [DONE]\n" {
				// log.Println("Stream ended with [DONE] message.")
				resultChan <- ProcessResult{ID: processResultID, Err: nil, Final: true}
				break
			}

			if strings.HasPrefix(line, "data:") {
				// log.Printf("Process Array: %v", line)
				jsonStr := strings.TrimPrefix(line, "data:")
				// Create a channel to receive the results
				// Start the goroutine, passing the channel for communication
				processChunk(jsonStr, resultChan, processResultID)
				processResultID++ // Increment the ID for each processed chunk
			}
		}

		// log.Println("Process Array: ")
		return ProcessResult{Err: nil}
	}
}

// Converts the array of json messages into a single Message
func constructJsonMessage(arrayOfProcessResult []ProcessResult) (MessageToSend, error) {
	newMessage := MessageToSend{Role: "assistant", Content: ""}
	for _, aMessage := range arrayOfProcessResult {
		if len(aMessage.Result.Choices) > 0 {
			choice := aMessage.Result.Choices[0]
			if choice.FinishReason == "stop" || aMessage.Final {
				break
			}

			if content, ok := choice.Delta["content"].(string); ok {
				newMessage.Content += content
			} else {
				// Handle the case where the type assertion fails, e.g., log an error or return
				formattedError := fmt.Errorf("type assertion to string failed for choice.Delta[\"content\"]")
				return MessageToSend{}, formattedError
			}

		}
	}
	return newMessage, nil
}

func processChunk(chunkData string, resultChan chan<- ProcessResult, id int) {
	var chunk CompletionChunk
	err := json.Unmarshal([]byte(chunkData), &chunk)
	if err != nil {
		log.Println("Error unmarshalling:", err)
		resultChan <- ProcessResult{ID: id, Result: CompletionChunk{}, Err: err}
	}

	// Process the chunk as needed
	resultChan <- ProcessResult{ID: id, Result: chunk, Err: nil}
}
