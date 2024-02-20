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
	return Settings{}, nil
}
