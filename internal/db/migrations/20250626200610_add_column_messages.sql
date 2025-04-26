-- +goose Up
-- +goose StatementBegin
ALTER TABLE messages ADD COLUMN attachment_paths TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE messages DROP COLUMN attachment_paths;
-- +goose StatementEnd
