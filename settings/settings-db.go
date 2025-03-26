package settings

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

// const ModelsCacheTtl = time.Second * 5

const ModelsCacheTtl = time.Hour * 24 * 14 // 14 days
const ModelsSeparator = ";"
const DateLayout = "2006-01-02 15:04:05"

type SettingsService struct {
	DB *sql.DB
}

func NewSettingsService(db *sql.DB) *SettingsService {
	return &SettingsService{
		DB: db,
	}
}

func (ss *SettingsService) GetSettings(ctx context.Context, cfg config.Config) tea.Msg {
	settings := util.Settings{}
	row := ss.DB.QueryRow(
		`select settings_id, settings_model, settings_max_tokens, settings_frequency from settings`,
	)
	err := row.Scan(&settings.ID, &settings.Model, &settings.MaxTokens, &settings.Frequency)

	availableModels, modelsError := ss.GetProviderModels(cfg.Provider, cfg.ChatGPTApiUrl)

	if modelsError != nil {
		return util.ErrorEvent{Message: modelsError.Error()}
	}

	isModelFromSettingsAvailable := slices.Contains(availableModels, settings.Model)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return util.ErrorEvent{Message: err.Error()}
		}

		settings = util.Settings{
			Model:     availableModels[0],
			MaxTokens: 3000,
			Frequency: 0,
		}

		// if default model is set in config.json - use it instead
		if len(cfg.DefaultModel) > 0 {
			settings.Model = cfg.DefaultModel
		}
	}

	if !isModelFromSettingsAvailable && len(availableModels) > 0 {
		modelIdx := rand.IntN(len(availableModels) - 1)
		settings.Model = availableModels[modelIdx]
		ss.UpdateSettings(settings)
	}

	return UpdateSettingsEvent{
		Settings: settings,
		Err:      nil,
	}
}

func (ss *SettingsService) GetProviderModels(providerType string, apiUrl string) ([]string, error) {
	provider := util.GetOpenAiInferenceProvider(providerType, apiUrl)
	availableModels := []string{}

	if provider != util.Local {
		var cacheErr error
		availableModels, cacheErr = ss.TryGetModelsCache(int(provider))
		if cacheErr != nil {
			log.Println("Faild to get models cache: ", cacheErr)
		}
	}

	if len(availableModels) == 0 {
		llmClient := clients.ResolveLlmClient(providerType, apiUrl, "")
		modelsResponse := llmClient.RequestModelsList()
		if modelsResponse.Err != nil {
			return []string{}, modelsResponse.Err
		}

		availableModels = util.GetFilteredModelList(providerType, apiUrl, modelsResponse.Result.GetModelNamesFromResponse())

		if provider == util.Local {
			return availableModels, nil
		}

		err := ss.CacheModelsForProvider(int(provider), availableModels)
		if err != nil {
			log.Println("Cache update error:", err)
		}
	}

	return availableModels, nil
}

func (ss *SettingsService) TryGetModelsCache(provider int) ([]string, error) {
	var cachedModels string
	var cachedAt string
	row := ss.DB.QueryRow(
		`select models, cached_at from models where provider = $1`,
		provider,
	)
	err := row.Scan(&cachedModels, &cachedAt)

	if err != nil {
		return []string{}, err
	}

	expireDate := time.Now().UTC().Add(-ModelsCacheTtl)
	parsedDate, err := time.Parse(DateLayout, cachedAt)

	if err == nil && parsedDate.Before(expireDate) {
		return []string{}, errors.New("Models cache expired")
	}

	modelsList := strings.Split(cachedModels, ModelsSeparator)
	filteredModels := []string{}
	for _, model := range modelsList {
		if len(model) != 0 {
			filteredModels = append(filteredModels, model)
		}
	}

	return filteredModels, nil
}

func (ss *SettingsService) CacheModelsForProvider(provider int, models []string) error {
	mergedString := strings.Join(models, ModelsSeparator)

	upsert := `
		INSERT INTO models
			(provider, models, cached_at)
		VALUES
			($1, $2, $3)
		ON CONFLICT(provider) DO UPDATE SET
			models=$2,
			cached_at=$3;
	`

	_, err := ss.DB.Exec(
		upsert,
		provider,
		mergedString,
		time.Now().UTC().Format(DateLayout),
	)
	return err
}

func (ss *SettingsService) UpdateSettings(newSettings util.Settings) (util.Settings, error) {
	upsert := `
		INSERT INTO settings 
			(settings_id, settings_model, settings_max_tokens, settings_frequency)
		VALUES
			($1, $2, $3, $4)
		ON CONFLICT(settings_id) DO UPDATE SET
			settings_model=$2,
			settings_max_tokens=$3,
			settings_frequency=$4;
	`

	_, err := ss.DB.Exec(
		upsert,
		newSettings.ID,
		newSettings.Model,
		newSettings.MaxTokens,
		newSettings.Frequency,
	)
	if err != nil {
		return newSettings, err
	}
	return newSettings, nil
}
