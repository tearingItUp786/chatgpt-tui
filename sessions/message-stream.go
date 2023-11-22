package sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	Result CompletionChunk // or whatever type you need
	Err    error
}

type ArrayProccessResult []ProcessResult

const API_KEY = "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"

func CallChatGpt() tea.Cmd {
	resultChannel := make(chan ProcessResult)
	processArray := []ProcessResult{}

	return func() tea.Msg {
		body := []byte(
			`{
    "model": "gpt-3.5-turbo-1106",
    "stream": true,
    "messages": [

      {
        "role": "user",
        "content": "Explain why popcorn pops to a kid who loves watching it in the microwave"
      }
    ]
  }
`)

		req, err := http.NewRequest(
			"POST",
			"https://api.openai.com/v1/chat/completions",
			bytes.NewBuffer(body),
		)
		if err != nil {
			// Handle error
			fmt.Println("Error creating request:", err)
			return ArrayProccessResult{}
		}

		// Set the Content-Type header
		req.Header.Set("Content-Type", "application/json")
		// Set any other headers you need
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", API_KEY))
		// Create a new HTTP client and send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			// Handle error
			log.Println("Error sending request:", err)
		}
		defer resp.Body.Close()

		scanner := bufio.NewReader(resp.Body)

		for {
			line, err := scanner.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					log.Println("end of file")
					break // End of the stream
				}
				log.Fatal(err) // Handle other errors
			}

			if line == "data: [DONE]\n" {
				log.Println("Stream ended with [DONE] message.")
				processArray = append(processArray, ProcessResult{Err: nil})
				break
			}

			if strings.HasPrefix(line, "data: ") {
				jsonStr := strings.TrimPrefix(line, "data: ")
				// Create a channel to receive the results
				// Start the goroutine, passing the channel for communication
				go processChunk(jsonStr, resultChannel)
				x := <-resultChannel
				processArray = append(processArray, x)
			}
		}

		close(resultChannel)
		log.Println("Process Array: ")
		return ArrayProccessResult(processArray)
	}
}

func processChunk(chunkData string, resultChan chan<- ProcessResult) {
	var chunk CompletionChunk
	err := json.Unmarshal([]byte(chunkData), &chunk)
	if err != nil {
		log.Println("Error unmarshalling:", err)
		resultChan <- ProcessResult{Result: CompletionChunk{}, Err: err}
	}

	// Process the chunk as needed
	// log.Printf("Processed Data :v %v", chunk)
	resultChan <- ProcessResult{Result: chunk, Err: nil}
	// Send the result back through the channel
}
