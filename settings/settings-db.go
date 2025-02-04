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

	"github.com/tearingItUp786/chatgpt-tui/clients"
	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

const ModelsCacheTtl = time.Hour * 24 * 14 // 14 days
const ModelsSeparator = ";"

type SettingsService struct {
	DB *sql.DB
}

func NewSettingsService(db *sql.DB) *SettingsService {
	return &SettingsService{
		DB: db,
	}
}

func (ss *SettingsService) GetSettings(ctx context.Context, cfg config.Config) (util.Settings, error) {
	settings := util.Settings{}
	row := ss.DB.QueryRow(
		`select settings_id, settings_model, settings_max_tokens, settings_frequency from settings`,
	)
	err := row.Scan(&settings.ID, &settings.Model, &settings.MaxTokens, &settings.Frequency)

	availableModels := ss.GetProviderModels(cfg.ChatGPTApiUrl)
	isModelFromSettingsAvailable := slices.Contains(availableModels, settings.Model)

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return util.Settings{}, err
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

	return settings, nil
}

func (ss *SettingsService) GetProviderModels(apiUrl string) []string {
	provider := util.GetInferenceProvider(apiUrl)
	availableModels := []string{}

	if provider != util.Local {
		availableModels, _ = ss.TryGetModelsCache(int(provider))
	}

	if len(availableModels) == 0 {
		openAiClient := clients.NewOpenAiClient(apiUrl, "")
		modelsResponse := openAiClient.RequestModelsList()
		if modelsResponse.Err != nil {
			panic(modelsResponse.Err)
		}

		availableModels := util.GetFilteredModelList(apiUrl, modelsResponse.Result.GetModelNames())

		if provider == util.Local {
			return availableModels
		}

		err := ss.CacheModelsForProvider(int(provider), availableModels)
		if err != nil {
			return []string{}
		}
	}

	return availableModels
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

	layout := "2025-02-01 12:01:25"
	if parsedDate, err := time.Parse(layout, cachedAt); err != nil {
		log.Println("Failed to check cache expiration")
	} else {
		if parsedDate.Before(time.Now().UTC().Add(ModelsCacheTtl)) {
			return []string{}, errors.New("cache expired")
		}
	}

	response := strings.Split(cachedModels, ModelsSeparator)
	return response, nil
}

func (ss *SettingsService) CacheModelsForProvider(provider int, models []string) error {
	mergedString := strings.Join(models, ModelsSeparator)

	upsert := `
		INSERT INTO models
			(provider, models)
		VALUES
			($1, $2)
		ON CONFLICT(provider) DO UPDATE SET
			models=$2;
	`

	_, err := ss.DB.Exec(
		upsert,
		provider,
		mergedString,
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
