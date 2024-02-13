-- +goose Up
-- +goose StatementBegin
CREATE TABLE settings (
  settings_id INTEGER PRIMARY KEY,
  settings_model VARCHAR(255) NOT NULL,
  settings_max_tokens TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
