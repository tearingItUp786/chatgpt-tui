-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
  id INTEGER PRIMARY KEY,
  messages JSON NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  session_name VARCHAR(255) NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
-- +goose StatementEnd
