package util

type Settings struct {
	ID        int
	Model     string
	MaxTokens int
	Frequency int
}

type MessageToSend struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
