package settings

import "database/sql"

type Settings struct {
	ID        int
	Model     string
	MaxTokens int
	Frequency int
}

type SettingsService struct {
	DB *sql.DB
}

func NewSettingsService(db *sql.DB) *SettingsService {
	return &SettingsService{
		DB: db,
	}
}

func (ss *SettingsService) GetSettings() (Settings, error) {
	settings := Settings{}
	row := ss.DB.QueryRow(
		`select settings_id, settings_model, settings_max_tokens, settings_frequency from settings`,
	)
	err := row.Scan(&settings.ID, &settings.Model, &settings.MaxTokens, &settings.Frequency)
	if err != nil {
		return Settings{}, err
	}

	return settings, nil
}

func (ss *SettingsService) UpdateSettings(newSettings Settings) (Settings, error) {
	_, err := ss.DB.Exec(
		`UPDATE settings SET settings_model=$1, settings_max_tokens=$2, settings_frequency=$3 WHERE settings_id=$4`,
		newSettings.Model,
		newSettings.MaxTokens,
		newSettings.Frequency,
		newSettings.ID,
	)
	if err != nil {
		return newSettings, err
	}
	return newSettings, nil
}
