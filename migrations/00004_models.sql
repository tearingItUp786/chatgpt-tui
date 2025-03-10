-- +goose Up
-- +goose StatementBegin
CREATE TABLE models (
  provider INTEGER PRIMARY KEY,
  models VARCHAR(5000) NOT NULL,
	cached_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE models;
-- +goose StatementEnd
