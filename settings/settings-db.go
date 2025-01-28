package settings

import (
	"context"
	"database/sql"
	"errors"

	"github.com/tearingItUp786/chatgpt-tui/config"
	"github.com/tearingItUp786/chatgpt-tui/util"
)

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

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return util.Settings{}, err
		}

		// TODO replace with request to v1/models
		settings = util.Settings{
			Model:     "gpt-3.5-turbo",
			MaxTokens: 300,
			Frequency: 0,
		}

		// if default model is set in config.json - use it instead
		if len(cfg.DefaultModel) > 0 {
			settings.Model = cfg.DefaultModel
		}
	}

	return settings, nil
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
