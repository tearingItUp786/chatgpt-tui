package util

import (
	"slices"
	"strings"
)

var (
	openAiChatModelsPrefixes = []string{"gpt-", "o1", "o3"}
	openAiExclusionKeywords  = []string{"audio", "realtime"}
)

var (
	openAiApiPrefixes = []string{"https://api.openai.com"}
	localApiPrefixes  = []string{"localhost", "127.0.0.1", "::1"}
)

type ApiProvider int

const (
	OpenAi ApiProvider = iota
	Local
)

func GetFilteredModelList(apiUrl string, models []string) []string {
	var modelNames []string
	provider := GetInferenceProvider(apiUrl)

	for _, model := range models {
		switch provider {
		case Local:
			modelNames = append(modelNames, model)
		case OpenAi:
			if isOpenAiChatModel(model) {
				modelNames = append(modelNames, model)
			}
		}
	}

	return modelNames
}

func GetInferenceProvider(apiUrl string) ApiProvider {
	if slices.ContainsFunc(openAiApiPrefixes, func(p string) bool {
		return strings.Contains(apiUrl, p)
	}) {
		return OpenAi
	}

	if slices.ContainsFunc(localApiPrefixes, func(p string) bool {
		return strings.Contains(apiUrl, p)
	}) {
		return Local
	}

	return Local
}

func isOpenAiChatModel(model string) bool {
	for _, keyword := range openAiExclusionKeywords {
		if strings.Contains(model, keyword) {
			return false
		}
	}

	for _, prefix := range openAiChatModelsPrefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}

	return false
}
