-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions ADD COLUMN prompt_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN completion_tokens INTEGER NOT NULL DEFAULT 0; 
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions DROP COLUMN prompt_tokens;
ALTER TABLE sessions DROP COLUMN completion_tokens;
-- +goose StatementEnd
