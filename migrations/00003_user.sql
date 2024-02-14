-- +goose Up
-- +goose StatementBegin
CREATE TABLE user (
  user_id INTEGER PRIMARY KEY,
  user_active_session_id INTEGER,

  FOREIGN KEY (user_active_session_id) REFERENCES sessions (sessions_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user;
-- +goose StatementEnd
