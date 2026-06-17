-- +goose Up
-- +goose StatementBegin
ALTER TABLE segments
    ADD COLUMN status TEXT NOT NULL DEFAULT 'applied'
    CHECK (status IN ('pending', 'applied'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE segments DROP COLUMN status;
-- +goose StatementEnd
