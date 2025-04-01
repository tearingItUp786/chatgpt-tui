-- +goose Up
-- +goose StatementBegin
ALTER TABLE settings ADD COLUMN system_msg TEXT NULL;
ALTER TABLE settings ADD COLUMN temperature REAL NULL; 
ALTER TABLE settings ADD COLUMN top_p REAL NULL; 

ALTER TABLE settings DROP COLUMN settings_frequency; 
ALTER TABLE settings ADD COLUMN settings_frequency REAL NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE settings DROP COLUMN system_msg;
ALTER TABLE settings DROP COLUMN temperature;
ALTER TABLE settings DROP COLUMN top_p;
ALTER TABLE settings DROP COLUMN settings_frequency;
ALTER TABLE settings ADD COLUMN settings_frequency REAL NOT NULL;
-- +goose StatementEnd
