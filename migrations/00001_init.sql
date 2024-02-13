-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
  sessions_id INTEGER PRIMARY KEY,
  sessions_messages JSON NOT NULL,
  sessions_created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  sessions_session_name VARCHAR(255) NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
-- +goose StatementEnd
